package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Doremi203/personage/backend/tasker/eval/internal/fixture"
	"github.com/Doremi203/personage/backend/tasker/eval/internal/report"
	"github.com/Doremi203/personage/backend/tasker/eval/internal/runner"
	processingpb "github.com/Doremi203/personage/backend/tasker/gen/api/processing"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const prodTraitexGRPC = "grpc-traitex.persomanage.ru:443"

func main() {
	fixturePath := flag.String("fixture", "", "path to fixture JSON (required)")
	taskerHTTP := flag.String("tasker-http", "", "base URL of eval tasker, e.g. http://localhost:8091 (required)")
	evalQueueURL := flag.String("eval-queue-url", "", "eval SQS queue URL (required)")
	reportPath := flag.String("report", "", "output report JSON path (default: stdout only)")
	reportOnly := flag.Bool("report-only", false, "skip DB reset and snapshot replay; list tasks and generate report from current state")
	runs := flag.Int("runs", 1, "number of independent runs to average")
	pollInterval := flag.Duration("poll-interval", 10*time.Second, "task poll interval")
	overallTimeout := flag.Duration("overall-timeout", 20*time.Minute, "max time per run")
	embeddingModel := flag.String("embedding-model", "", "embedding model (default: openai/text-embedding-3-small)")
	embeddingBaseURL := flag.String("embedding-base-url", "", "embedding API base URL (default: https://openrouter.ai/api/v1)")
	matchTokenThreshold := flag.Float64("match-token-threshold", 0, "max cost (1−tokenF1) for a title match; default 0.7 (tokenF1 ≥ 0.3)")
	matchEmbedThreshold := flag.Float64("match-embed-threshold", 0, "max cost (1−cosine) for a title match when embedding matching is used; default 0.45 (cosine ≥ 0.55)")
	flag.Parse()

	embeddingAPIKey := os.Getenv("EMBEDDING_API_KEY")

	if *fixturePath == "" || *taskerHTTP == "" || *evalQueueURL == "" {
		flag.Usage()
		os.Exit(1)
	}

	fix, err := fixture.Load(*fixturePath)
	if err != nil {
		fatalf("load fixture: %v", err)
	}

	traitexConn, err := grpc.NewClient(prodTraitexGRPC, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})))
	if err != nil {
		fatalf("connect to traitex: %v", err)
	}
	defer func() { _ = traitexConn.Close() }()

	r := &runner.Runner{
		Traitex: &traitexClient{client: processingpb.NewProcessingServiceClient(traitexConn)},
		Tasker:  &taskerClient{baseURL: *taskerHTTP},
		DB: &dbResetter{
			// backendDir: directory containing the Makefile (go up from this binary's source)
			backendDir: findBackendDir(),
		},
		Cfg: runner.Config{
			EvalQueueURL:        *evalQueueURL,
			PollInterval:        *pollInterval,
			OverallTimeout:      *overallTimeout,
			ReportOnly:          *reportOnly,
			EmbeddingAPIKey:     embeddingAPIKey,
			EmbeddingModel:      *embeddingModel,
			EmbeddingBaseURL:    *embeddingBaseURL,
			MatchTokenThreshold: *matchTokenThreshold,
			MatchEmbedThreshold: *matchEmbedThreshold,
		},
	}

	fixtureName := filepath.Base(*fixturePath)

	if *runs <= 1 {
		rep, err := r.Run(context.Background(), fix, fixtureName)
		if err != nil {
			fatalf("run: %v", err)
		}
		printAndSave(rep, *reportPath)
		return
	}

	// Multiple runs: average P/R/F1.
	type runResult struct {
		p, rc, f1 float64
	}
	results := make([]runResult, 0, *runs)
	for i := range *runs {
		fmt.Fprintf(os.Stderr, "--- Run %d/%d ---\n", i+1, *runs)
		rep, err := r.Run(context.Background(), fix, fixtureName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "run %d failed: %v\n", i+1, err)
			continue
		}
		results = append(results, runResult{rep.Precision, rep.Recall, rep.F1})
		fmt.Fprintf(os.Stderr, "  P=%.3f R=%.3f F1=%.3f\n", rep.Precision, rep.Recall, rep.F1)
	}

	if len(results) == 0 {
		fatalf("all runs failed")
	}

	var sumP, sumR, sumF1 float64
	for _, rr := range results {
		sumP += rr.p
		sumR += rr.rc
		sumF1 += rr.f1
	}
	n := float64(len(results))
	meanP, meanR, meanF1 := sumP/n, sumR/n, sumF1/n

	var varP, varR, varF1 float64
	for _, rr := range results {
		varP += (rr.p - meanP) * (rr.p - meanP)
		varR += (rr.rc - meanR) * (rr.rc - meanR)
		varF1 += (rr.f1 - meanF1) * (rr.f1 - meanF1)
	}

	fmt.Printf("=== %d-run average ===\n", len(results))
	fmt.Printf("Precision: %.3f ± %.3f\n", meanP, math.Sqrt(varP/n))
	fmt.Printf("Recall   : %.3f ± %.3f\n", meanR, math.Sqrt(varR/n))
	fmt.Printf("F1       : %.3f ± %.3f\n", meanF1, math.Sqrt(varF1/n))
}

func printAndSave(rep report.Report, path string) {
	report.Summarize(os.Stdout, rep)
	if path == "" {
		return
	}
	if err := report.Write(path, rep); err != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "report written to %s\n", path)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "eval-f1: "+format+"\n", args...)
	os.Exit(1)
}

// findBackendDir walks up from the binary source until it finds a Makefile.
func findBackendDir() string {
	// When built with go run, the working directory is typically the repo root.
	// The Makefile lives in backend/.
	for _, candidate := range []string{".", "backend", "../backend", "../../backend"} {
		if _, err := os.Stat(filepath.Join(candidate, "Makefile")); err == nil {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}
	return "."
}

// traitexClient is the gRPC implementation of runner.TraitexClient.
type traitexClient struct {
	client processingpb.ProcessingServiceClient
}

func (c *traitexClient) SendProcessingSnapshot(ctx context.Context, snapshotID, targetQueueURL string) (int, error) {
	req := &processingpb.SendProcessingSnapshotRequest{
		SnapshotId:     snapshotID,
		TargetQueueUrl: &targetQueueURL,
	}
	resp, err := c.client.SendProcessingSnapshot(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("SendProcessingSnapshot: %w", err)
	}
	return int(resp.GetEventsCount()), nil
}

// taskerClient calls the eval tasker's test list endpoint (no auth required).
type taskerClient struct {
	baseURL string
}

type testListItem struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	ClusterID       *string    `json:"cluster_id,omitempty"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	DurationMinutes int        `json:"duration_minutes"`
	Priority        int        `json:"priority"`
	Deadline        *time.Time `json:"deadline,omitempty"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	Status          string     `json:"status"`
	Category        string     `json:"category"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func parseClusterID(s *string) *domain.ClusterID {
	if s == nil {
		return nil
	}
	c := domain.ClusterID(*s)
	return &c
}

func (c *taskerClient) ListTasks(ctx context.Context, userID string) ([]domain.Task, error) {
	url := c.baseURL + "/v1/test/tasks/list?user_id=" + userID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // eval tool, operator-controlled URL
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list tasks HTTP %d: %s", resp.StatusCode, body)
	}

	var items []testListItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("decode tasks: %w", err)
	}

	tasks := make([]domain.Task, len(items))
	for i, it := range items {
		tasks[i] = domain.Task{
			ID:          domain.TaskID(it.ID),
			UserID:      domain.UserID(it.UserID),
			ClusterID:   parseClusterID(it.ClusterID),
			Title:       it.Title,
			Description: it.Description,
			Duration:    time.Duration(it.DurationMinutes) * time.Minute,
			Priority:    it.Priority,
			Deadline:    it.Deadline,
			StartTime:   it.StartTime,
			EndTime:     it.EndTime,
			Status:      domain.TaskStatus(it.Status),
			Category:    domain.TaskCategory(it.Category),
			CreatedAt:   it.CreatedAt,
			UpdatedAt:   it.UpdatedAt,
		}
	}
	return tasks, nil
}

// dbResetter resets the eval DB by shelling out to make.
type dbResetter struct {
	backendDir string
}

func (d *dbResetter) Reset(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "make", "-C", d.backendDir, "tasker-eval/migrate/reset") //nolint:gosec // eval tool, fixed args
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tasker-eval/migrate/reset: %w", err)
	}
	return nil
}

// fakeJWT builds a minimal JWT carrying {"user_id": "<uid>"} in the payload.
// The signature is meaningless; the eval tasker uses a stub verifier that
// accepts any token as valid.
func fakeJWT(userID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":"` + userID + `"}`))
	return header + "." + payload + ".eval_stub_signature"
}

// withToken adds a fake JWT as gRPC metadata (used if needed for gRPC endpoints).
func withToken(ctx context.Context, userID string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "user-token", fakeJWT(userID))
}

// keep the function referenced so the compiler doesn't remove it.
var _ = withToken
