package runner

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func TestPollWaitsUntilTimeoutAndReturnsLatestTasks(t *testing.T) {
	previousMinTaskWaitTimeout := minTaskWaitTimeout
	minTaskWaitTimeout = 40 * time.Millisecond
	t.Cleanup(func() {
		minTaskWaitTimeout = previousMinTaskWaitTimeout
	})

	tasker := &stubTaskerClient{}
	r := &Runner{
		Tasker: tasker,
		Cfg: Config{
			PollInterval:   5 * time.Millisecond,
			OverallTimeout: 10 * time.Millisecond,
		},
	}

	start := time.Now()
	tasks, err := r.poll(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("poll returned error: %v", err)
	}

	if waited := time.Since(start); waited < 35*time.Millisecond {
		t.Fatalf("poll returned too early after %s", waited)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected latest task list, got %d tasks", len(tasks))
	}

	if calls := tasker.calls(); calls < 3 {
		t.Fatalf("expected multiple polls before timeout, got %d", calls)
	}
}

func TestBuildClusterDiagnostics(t *testing.T) {
	reason := "promo email"
	nonActionable := domain.ClusterGenerationOutcomeNonActionable
	diagnostics := []domain.ClusterGenerationDiagnostic{
		{
			ClusterID:          "cluster-1",
			UserID:             "user-1",
			Status:             domain.ClusterStatusClosed,
			GenerationOutcome:  &nonActionable,
			GenerationReason:   &reason,
			GeneratedTaskCount: 0,
		},
		{
			ClusterID:          "cluster-2",
			UserID:             "user-1",
			Status:             domain.ClusterStatusClosed,
			GeneratedTaskCount: 1,
		},
		{
			ClusterID:          "cluster-3",
			UserID:             "user-1",
			Status:             domain.ClusterStatusOpen,
			GeneratedTaskCount: 0,
		},
	}

	got := buildClusterDiagnostics(diagnostics)

	if got.Total != 3 {
		t.Fatalf("unexpected total clusters: %d", got.Total)
	}

	if got.Closed != 2 {
		t.Fatalf("unexpected closed clusters: %d", got.Closed)
	}

	if got.SkippedNonActionable != 1 {
		t.Fatalf("unexpected skipped non-actionable count: %d", got.SkippedNonActionable)
	}

	if got.TasklessClusterRate != 0.5 {
		t.Fatalf("unexpected taskless cluster rate: %f", got.TasklessClusterRate)
	}
}

type stubTaskerClient struct {
	mu    sync.Mutex
	count int
}

func (s *stubTaskerClient) ListTasks(context.Context, string) ([]domain.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count++
	if s.count < 3 {
		return nil, nil
	}

	return []domain.Task{{Title: "generated task"}}, nil
}

func (s *stubTaskerClient) ListClusterDiagnostics(context.Context, string) ([]domain.ClusterGenerationDiagnostic, error) {
	return nil, nil
}

func (s *stubTaskerClient) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}
