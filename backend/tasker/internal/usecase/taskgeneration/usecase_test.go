package taskgeneration

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/libs/go/tx"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func TestProcessClusterFinalizesNonActionableCluster(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	taskRepo := &stubTaskRepo{}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{events: []domain.Event{{ID: "event-1", ClusterID: "cluster-1"}}},
		taskRepo,
		stubActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: false, Reason: new("promo email")}},
		stubTaskGenerationService{},
		stubTxProvider{},
		log.Stub{},
		5,
		time.Minute,
	)

	err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"})
	if err != nil {
		t.Fatalf("processCluster returned error: %v", err)
	}

	if len(taskRepo.createdTasks) != 0 {
		t.Fatalf("expected no tasks to be created, got %d", len(taskRepo.createdTasks))
	}

	if len(clusterRepo.finalized) != 1 {
		t.Fatalf("expected 1 finalized cluster, got %d", len(clusterRepo.finalized))
	}

	finalized := clusterRepo.finalized[0]
	if finalized.outcome != domain.ClusterGenerationOutcomeNonActionable {
		t.Fatalf("unexpected outcome: %s", finalized.outcome)
	}

	if finalized.reason == nil || *finalized.reason != "promo email" {
		t.Fatalf("unexpected reason: %#v", finalized.reason)
	}
}

func TestProcessClusterStoresGeneratedEvidenceEventIDs(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	taskRepo := &stubTaskRepo{}
	events := []domain.Event{
		{ID: "event-1", ClusterID: "cluster-1"},
		{ID: "event-2", ClusterID: "cluster-1"},
		{ID: "event-3", ClusterID: "cluster-1"},
	}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{events: events},
		taskRepo,
		stubActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: true}},
		stubTaskGenerationService{result: domain.GeneratedTask{
			Title:            "Review PR #47",
			Description:      "Review the API integration changes.",
			DurationMinutes:  30,
			Priority:         7,
			Category:         "work",
			EvidenceEventIDs: []domain.EventID{"event-1", "event-3"},
		}},
		stubTxProvider{},
		log.Stub{},
		5,
		time.Minute,
	)

	err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"})
	if err != nil {
		t.Fatalf("processCluster returned error: %v", err)
	}

	if len(taskRepo.createdTasks) != 1 {
		t.Fatalf("expected 1 created task, got %d", len(taskRepo.createdTasks))
	}

	createdTask := taskRepo.createdTasks[0]
	if got, want := len(createdTask.EvidenceEventIDs), 2; got != want {
		t.Fatalf("unexpected evidence count: got %d want %d", got, want)
	}

	if createdTask.EvidenceEventIDs[0] != "event-1" || createdTask.EvidenceEventIDs[1] != "event-3" {
		t.Fatalf("unexpected evidence ids: %#v", createdTask.EvidenceEventIDs)
	}

	if len(clusterRepo.finalized) != 1 || clusterRepo.finalized[0].outcome != domain.ClusterGenerationOutcomeTaskGenerated {
		t.Fatalf("unexpected finalize calls: %#v", clusterRepo.finalized)
	}
}

type stubEventRepo struct {
	events []domain.Event
	err    error
}

func (s stubEventRepo) UpsertEvent(context.Context, domain.EventWithEmbedding) error { return nil }
func (s stubEventRepo) GetEventsByClusterID(context.Context, domain.ClusterID) ([]domain.Event, error) {
	return s.events, s.err
}
func (s stubEventRepo) DeleteEventsByClusterID(context.Context, domain.ClusterID) error { return nil }

type finalizeCall struct {
	outcome domain.ClusterGenerationOutcome
	reason  *string
}

type stubClusterRepo struct {
	finalized       []finalizeCall
	updatedStatuses []domain.ClusterStatus
}

func (s *stubClusterRepo) FindSimilarClusters(context.Context, domain.UserID, []float32, int) ([]domain.ClusterWithSimilarity, error) {
	return nil, nil
}
func (s *stubClusterRepo) UpsertCluster(context.Context, domain.Cluster) error { return nil }
func (s *stubClusterRepo) FindClosableClusters(context.Context, int, time.Duration, int) ([]domain.Cluster, error) {
	return nil, nil
}
func (s *stubClusterRepo) UpdateClusterStatus(_ context.Context, _ domain.ClusterID, status domain.ClusterStatus) error {
	s.updatedStatuses = append(s.updatedStatuses, status)
	return nil
}
func (s *stubClusterRepo) FinalizeCluster(
	_ context.Context,
	_ domain.ClusterID,
	outcome domain.ClusterGenerationOutcome,
	reason *string,
) error {
	s.finalized = append(s.finalized, finalizeCall{outcome: outcome, reason: reason})
	return nil
}
func (s *stubClusterRepo) ListGenerationDiagnosticsByUserID(context.Context, domain.UserID) ([]domain.ClusterGenerationDiagnostic, error) {
	return nil, nil
}
func (s *stubClusterRepo) DeleteCluster(context.Context, domain.ClusterID) error { return nil }
func (s *stubClusterRepo) RecoverStaleClusters(context.Context, time.Duration) (int, error) {
	return 0, nil
}

type stubTaskRepo struct {
	createdTasks []domain.Task
}

func (s *stubTaskRepo) CreateTask(_ context.Context, task domain.Task) error {
	s.createdTasks = append(s.createdTasks, task)
	return nil
}
func (s *stubTaskRepo) GetTaskByID(context.Context, domain.TaskID, domain.UserID) (domain.Task, error) {
	return domain.Task{}, nil
}
func (s *stubTaskRepo) GetTasksByUserID(context.Context, domain.UserID) ([]domain.Task, error) {
	return nil, nil
}
func (s *stubTaskRepo) GetTasksByStatus(context.Context, domain.UserID, domain.TaskStatus) ([]domain.Task, error) {
	return nil, nil
}
func (s *stubTaskRepo) GetUsersWithUnplannedTasks(context.Context) ([]domain.UserID, error) {
	return nil, nil
}
func (s *stubTaskRepo) UpdateTaskSchedule(context.Context, domain.TaskID, time.Time, time.Time, domain.TaskStatus) error {
	return nil
}
func (s *stubTaskRepo) UpdateTaskStatus(context.Context, domain.TaskID, domain.TaskStatus) error {
	return nil
}
func (s *stubTaskRepo) UpdateTask(context.Context, domain.TaskID, domain.UserID, domain.TaskUpdate) (domain.Task, error) {
	return domain.Task{}, nil
}
func (s *stubTaskRepo) DeleteTask(context.Context, domain.TaskID) error { return nil }
func (s *stubTaskRepo) ListTasks(context.Context, domain.TaskFilter, domain.Pagination) ([]domain.Task, int, error) {
	return nil, 0, nil
}

type stubActionabilityService struct {
	result domain.TaskGenerationDecision
	err    error
}

func (s stubActionabilityService) GetTaskGenerationDecision(context.Context, []domain.Event) (domain.TaskGenerationDecision, error) {
	return s.result, s.err
}

type stubTaskGenerationService struct {
	result domain.GeneratedTask
	err    error
}

func (s stubTaskGenerationService) GenerateTask(context.Context, []domain.Event) (domain.GeneratedTask, error) {
	return s.result, s.err
}

type stubTxProvider struct{}

func (stubTxProvider) RunWithTx(ctx context.Context, _ tx.Isolation, op func(context.Context) error) error {
	return op(ctx)
}
