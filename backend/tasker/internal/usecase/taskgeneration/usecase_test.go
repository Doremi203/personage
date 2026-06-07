package taskgeneration

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/libs/go/tx"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func TestProcessClosableClustersUsesRuntimeSettings(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{},
		&stubTaskRepo{},
		stubModerationRepo{},
		stubActionabilityService{},
		stubTaskGenerationService{},
		stubUserProfileService{},
		stubEmbeddingService{},
		stubTxProvider{},
		log.Stub{},
		stubSettingsProvider{settings: domain.GenerationSettings{
			MaxEventCount:     9,
			InactivityTimeout: 7 * time.Minute,
			BatchSize:         3,
		}},
		time.Now,
	)

	if err := uc.ProcessClosableClusters(t.Context()); err != nil {
		t.Fatalf("ProcessClosableClusters returned error: %v", err)
	}

	if clusterRepo.recoverThreshold != 7*time.Minute {
		t.Fatalf("expected recover threshold 7m, got %v", clusterRepo.recoverThreshold)
	}
	if len(clusterRepo.findClosableArgs) != 1 {
		t.Fatalf("expected 1 FindClosableClusters call, got %d", len(clusterRepo.findClosableArgs))
	}
	call := clusterRepo.findClosableArgs[0]
	if call.maxEventCount != 9 || call.inactivityDuration != 7*time.Minute || call.limit != 3 {
		t.Fatalf("settings not threaded into FindClosableClusters: %#v", call)
	}
}

func TestProcessClusterFinalizesNonActionableCluster(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	taskRepo := &stubTaskRepo{}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{events: []domain.Event{{ID: "event-1", ClusterID: "cluster-1"}}},
		taskRepo,
		stubModerationRepo{},
		stubActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: false, Reason: new("promo email")}},
		stubTaskGenerationService{},
		stubUserProfileService{profile: domain.UserProfile{Email: "owner@example.com", Name: "Owner"}},
		stubEmbeddingService{embeddings: [][]float32{{0.1, 0.2, 0.3}}},
		stubTxProvider{},
		log.Stub{},
		stubSettingsProvider{},
		time.Now,
	)

	err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"}, 0.97)
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

func TestProcessClusterStoresGeneratedTask(t *testing.T) {
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
		stubModerationRepo{},
		stubActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: true, Reason: new("explicit task request")}},
		stubTaskGenerationService{result: domain.GeneratedTask{
			Title:           "Review PR #47",
			Description:     "Review the API integration changes.",
			DurationMinutes: 30,
			Priority:        7,
			Category:        "work",
		}},
		stubUserProfileService{profile: domain.UserProfile{Email: "owner@example.com", Name: "Owner"}},
		stubEmbeddingService{embeddings: [][]float32{{0.1, 0.2, 0.3}}},
		stubTxProvider{},
		log.Stub{},
		stubSettingsProvider{},
		time.Now,
	)

	err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"}, 0.97)
	if err != nil {
		t.Fatalf("processCluster returned error: %v", err)
	}

	if len(taskRepo.createdTasks) != 1 {
		t.Fatalf("expected 1 created task, got %d", len(taskRepo.createdTasks))
	}

	createdTask := taskRepo.createdTasks[0]
	if createdTask.Title != "Review PR #47" {
		t.Fatalf("unexpected task title: %q", createdTask.Title)
	}

	if !createdTask.IsApproved {
		t.Fatalf("expected task to be auto-approved for user without manual moderation")
	}

	if len(clusterRepo.finalized) != 1 || clusterRepo.finalized[0].outcome != domain.ClusterGenerationOutcomeTaskGenerated {
		t.Fatalf("unexpected finalize calls: %#v", clusterRepo.finalized)
	}

	if clusterRepo.finalized[0].reason == nil || *clusterRepo.finalized[0].reason != "explicit task request" {
		t.Fatalf("unexpected finalize reason: %#v", clusterRepo.finalized[0].reason)
	}
}

func TestProcessClusterSkipsDuplicateGeneratedTask(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	taskRepo := &stubTaskRepo{
		similarTaskID: "existing-task",
		similarScore:  0.98,
		similarFound:  true,
	}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{events: []domain.Event{{ID: "event-1", ClusterID: "cluster-1"}}},
		taskRepo,
		stubModerationRepo{},
		stubActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: true}},
		stubTaskGenerationService{result: domain.GeneratedTask{Title: "Reply to invite", Description: "Reply"}},
		stubUserProfileService{profile: domain.UserProfile{Email: "owner@example.com", Name: "Owner"}},
		stubEmbeddingService{embeddings: [][]float32{{0.1, 0.2, 0.3}}},
		stubTxProvider{},
		log.Stub{},
		stubSettingsProvider{settings: domain.GenerationSettings{TaskDuplicateThreshold: 0.97}},
		time.Now,
	)

	if err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"}, 0.97); err != nil {
		t.Fatalf("processCluster returned error: %v", err)
	}

	if len(taskRepo.createdTasks) != 0 {
		t.Fatalf("expected no tasks to be created for duplicate, got %d", len(taskRepo.createdTasks))
	}

	if len(clusterRepo.finalized) != 1 || clusterRepo.finalized[0].outcome != domain.ClusterGenerationOutcomeDuplicate {
		t.Fatalf("expected duplicate finalize, got %#v", clusterRepo.finalized)
	}

	if clusterRepo.finalized[0].reason == nil || *clusterRepo.finalized[0].reason == "" {
		t.Fatalf("expected a non-empty duplicate reason, got %#v", clusterRepo.finalized[0].reason)
	}
}

func TestProcessClusterCreatesTaskWithEmbeddingWhenNotDuplicate(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	taskRepo := &stubTaskRepo{similarFound: false}
	embedding := []float32{0.4, 0.5, 0.6}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{events: []domain.Event{{ID: "event-1", ClusterID: "cluster-1"}}},
		taskRepo,
		stubModerationRepo{},
		stubActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: true}},
		stubTaskGenerationService{result: domain.GeneratedTask{Title: "Review PR", Description: "Review"}},
		stubUserProfileService{profile: domain.UserProfile{Email: "owner@example.com", Name: "Owner"}},
		stubEmbeddingService{embeddings: [][]float32{embedding}},
		stubTxProvider{},
		log.Stub{},
		stubSettingsProvider{settings: domain.GenerationSettings{TaskDuplicateThreshold: 0.97}},
		time.Now,
	)

	if err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"}, 0.97); err != nil {
		t.Fatalf("processCluster returned error: %v", err)
	}

	if len(taskRepo.createdTasks) != 1 {
		t.Fatalf("expected 1 created task, got %d", len(taskRepo.createdTasks))
	}

	if len(taskRepo.createdTasks[0].Embedding) != len(embedding) {
		t.Fatalf("expected task embedding to be set, got %#v", taskRepo.createdTasks[0].Embedding)
	}

	if len(clusterRepo.finalized) != 1 || clusterRepo.finalized[0].outcome != domain.ClusterGenerationOutcomeTaskGenerated {
		t.Fatalf("expected task_generated finalize, got %#v", clusterRepo.finalized)
	}
}

func TestProcessClusterCreatesTaskWhenEmbeddingFails(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	taskRepo := &stubTaskRepo{}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{events: []domain.Event{{ID: "event-1", ClusterID: "cluster-1"}}},
		taskRepo,
		stubModerationRepo{},
		stubActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: true}},
		stubTaskGenerationService{result: domain.GeneratedTask{Title: "Review PR", Description: "Review"}},
		stubUserProfileService{profile: domain.UserProfile{Email: "owner@example.com", Name: "Owner"}},
		stubEmbeddingService{embeddings: nil},
		stubTxProvider{},
		log.Stub{},
		stubSettingsProvider{settings: domain.GenerationSettings{TaskDuplicateThreshold: 0.97}},
		time.Now,
	)

	if err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"}, 0.97); err != nil {
		t.Fatalf("processCluster returned error: %v", err)
	}

	if len(taskRepo.createdTasks) != 1 {
		t.Fatalf("expected task to still be created when embedding fails, got %d", len(taskRepo.createdTasks))
	}

	if taskRepo.createdTasks[0].Embedding != nil {
		t.Fatalf("expected nil embedding when embedding generation yields none, got %#v", taskRepo.createdTasks[0].Embedding)
	}

	if taskRepo.findSimilarCalls != 0 {
		t.Fatalf("expected no dedup query when embedding fails, got %d calls", taskRepo.findSimilarCalls)
	}

	if len(clusterRepo.finalized) != 1 || clusterRepo.finalized[0].outcome != domain.ClusterGenerationOutcomeTaskGenerated {
		t.Fatalf("expected task_generated finalize, got %#v", clusterRepo.finalized)
	}
}

func TestProcessClusterMarksTaskUnapprovedWhenUserRequiresModeration(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	taskRepo := &stubTaskRepo{}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{events: []domain.Event{{ID: "event-1", ClusterID: "cluster-1"}}},
		taskRepo,
		stubModerationRepo{required: true},
		stubActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: true}},
		stubTaskGenerationService{result: domain.GeneratedTask{Title: "Reply to invite"}},
		stubUserProfileService{profile: domain.UserProfile{Email: "owner@example.com", Name: "Owner"}},
		stubEmbeddingService{embeddings: [][]float32{{0.1, 0.2, 0.3}}},
		stubTxProvider{},
		log.Stub{},
		stubSettingsProvider{},
		time.Now,
	)

	err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"}, 0.97)
	if err != nil {
		t.Fatalf("processCluster returned error: %v", err)
	}

	if len(taskRepo.createdTasks) != 1 {
		t.Fatalf("expected 1 created task, got %d", len(taskRepo.createdTasks))
	}

	if taskRepo.createdTasks[0].IsApproved {
		t.Fatalf("expected task to be unapproved for moderated user")
	}
}

func TestProcessClusterPassesUserProfileToTaskGenerator(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	taskRepo := &stubTaskRepo{}
	taskGen := &recordingTaskGenerationService{result: domain.GeneratedTask{Title: "Review PR #47", DurationMinutes: 30, Priority: 7, Category: "work"}}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{events: []domain.Event{{ID: "event-1", ClusterID: "cluster-1"}}},
		taskRepo,
		stubModerationRepo{},
		stubActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: true, Reason: new("explicit task request")}},
		taskGen,
		stubUserProfileService{profile: domain.UserProfile{Email: "owner@example.com", Name: "Owner", ConnectedEmails: []string{"alt@example.com"}}},
		stubEmbeddingService{embeddings: [][]float32{{0.1, 0.2, 0.3}}},
		stubTxProvider{},
		log.Stub{},
		stubSettingsProvider{},
		time.Now,
	)

	if err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"}, 0.97); err != nil {
		t.Fatalf("processCluster returned error: %v", err)
	}

	if taskGen.received.Email != "owner@example.com" || taskGen.received.Name != "Owner" || len(taskGen.received.ConnectedEmails) != 1 || taskGen.received.ConnectedEmails[0] != "alt@example.com" {
		t.Fatalf("task generator did not receive owner profile: %#v", taskGen.received)
	}
}

func TestProcessClusterPassesUserProfileToClassifier(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	taskRepo := &stubTaskRepo{}
	actionability := &recordingActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: false, Reason: new("no addressee")}}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{events: []domain.Event{{ID: "event-1", ClusterID: "cluster-1"}}},
		taskRepo,
		stubModerationRepo{},
		actionability,
		stubTaskGenerationService{},
		stubUserProfileService{profile: domain.UserProfile{Email: "owner@example.com", Name: "Owner"}},
		stubEmbeddingService{embeddings: [][]float32{{0.1, 0.2, 0.3}}},
		stubTxProvider{},
		log.Stub{},
		stubSettingsProvider{},
		time.Now,
	)

	if err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"}, 0.97); err != nil {
		t.Fatalf("processCluster returned error: %v", err)
	}

	if actionability.received.Email != "owner@example.com" || actionability.received.Name != "Owner" {
		t.Fatalf("classifier did not receive owner profile: %#v", actionability.received)
	}
}

func TestProcessClusterDegradesWhenUserProfileLookupFails(t *testing.T) {
	clusterRepo := &stubClusterRepo{}
	taskRepo := &stubTaskRepo{}
	actionability := &recordingActionabilityService{result: domain.TaskGenerationDecision{ShouldGenerate: false, Reason: new("no addressee")}}
	uc := NewUseCase(
		clusterRepo,
		stubEventRepo{events: []domain.Event{{ID: "event-1", ClusterID: "cluster-1"}}},
		taskRepo,
		stubModerationRepo{},
		actionability,
		stubTaskGenerationService{},
		stubUserProfileService{err: domain.ErrUserProfileNotFound},
		stubEmbeddingService{embeddings: [][]float32{{0.1, 0.2, 0.3}}},
		stubTxProvider{},
		log.Stub{},
		stubSettingsProvider{},
		time.Now,
	)

	if err := uc.processCluster(t.Context(), domain.Cluster{ID: "cluster-1", UserID: "user-1"}, 0.97); err != nil {
		t.Fatalf("processCluster returned error: %v", err)
	}

	if actionability.received.Email != "" || actionability.received.Name != "" || len(actionability.received.ConnectedEmails) != 0 {
		t.Fatalf("expected empty profile on lookup failure, got %#v", actionability.received)
	}

	if len(clusterRepo.finalized) != 1 || clusterRepo.finalized[0].outcome != domain.ClusterGenerationOutcomeNonActionable {
		t.Fatalf("expected non-actionable finalize, got %#v", clusterRepo.finalized)
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
func (s stubEventRepo) GetEventsByUserID(context.Context, domain.UserID, int) ([]domain.Event, error) {
	return s.events, s.err
}
func (s stubEventRepo) DeleteEventsByClusterID(context.Context, domain.ClusterID) error { return nil }
func (s stubEventRepo) MaxSimilarityByClusters(
	context.Context,
	[]domain.ClusterID,
	[]float32,
) (map[domain.ClusterID]float64, error) {
	return nil, nil
}

type finalizeCall struct {
	outcome domain.ClusterGenerationOutcome
	reason  *string
}

type findClosableCall struct {
	maxEventCount      int
	inactivityDuration time.Duration
	limit              int
}

type stubClusterRepo struct {
	finalized        []finalizeCall
	updatedStatuses  []domain.ClusterStatus
	findClosableArgs []findClosableCall
	recoverThreshold time.Duration
}

func (s *stubClusterRepo) FindSimilarClusters(context.Context, domain.UserID, []float32, int) ([]domain.ClusterWithSimilarity, error) {
	return nil, nil
}
func (s *stubClusterRepo) FindSimilarClosedClusters(context.Context, domain.UserID, []float32, int) ([]domain.ClusterWithSimilarity, error) {
	return nil, nil
}
func (s *stubClusterRepo) UpsertCluster(context.Context, domain.Cluster) error { return nil }
func (s *stubClusterRepo) FindClosableClusters(_ context.Context, maxEventCount int, inactivityDuration time.Duration, limit int) ([]domain.Cluster, error) {
	s.findClosableArgs = append(s.findClosableArgs, findClosableCall{
		maxEventCount:      maxEventCount,
		inactivityDuration: inactivityDuration,
		limit:              limit,
	})
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
func (s *stubClusterRepo) ListAdminClustersByUserID(context.Context, domain.UserID, int) ([]domain.AdminClusterListItem, error) {
	return nil, nil
}
func (s *stubClusterRepo) GetAdminClusterByID(context.Context, domain.ClusterID) (domain.AdminClusterListItem, error) {
	return domain.AdminClusterListItem{}, nil
}
func (s *stubClusterRepo) DeleteCluster(context.Context, domain.ClusterID) error { return nil }
func (s *stubClusterRepo) RecoverStaleClusters(_ context.Context, threshold time.Duration) (int, error) {
	s.recoverThreshold = threshold
	return 0, nil
}

type stubTaskRepo struct {
	createdTasks []domain.Task

	similarTaskID    domain.TaskID
	similarScore     float64
	similarFound     bool
	similarErr       error
	findSimilarCalls int
}

func (s *stubTaskRepo) CreateTask(_ context.Context, task domain.Task) error {
	s.createdTasks = append(s.createdTasks, task)
	return nil
}
func (s *stubTaskRepo) FindMostSimilarActiveTask(_ context.Context, _ domain.UserID, _ []float32) (domain.TaskID, float64, bool, error) {
	s.findSimilarCalls++
	return s.similarTaskID, s.similarScore, s.similarFound, s.similarErr
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
func (s *stubTaskRepo) GetPlannedTasksInRange(context.Context, domain.UserID, time.Time, time.Time) ([]domain.Task, error) {
	return nil, nil
}
func (s *stubTaskRepo) GetUsersWithUnplannedTasks(context.Context) ([]domain.UserID, error) {
	return nil, nil
}
func (s *stubTaskRepo) GetUsersWithPlannedTasks(context.Context) ([]domain.UserID, error) {
	return nil, nil
}
func (s *stubTaskRepo) UpdateTaskSchedule(context.Context, domain.TaskID, time.Time, time.Time, domain.TaskStatus) error {
	return nil
}
func (s *stubTaskRepo) UpdateTaskStatus(context.Context, domain.TaskID, domain.TaskStatus) error {
	return nil
}
func (s *stubTaskRepo) UnscheduleTask(context.Context, domain.TaskID) error {
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

func (s stubActionabilityService) GetTaskGenerationDecision(context.Context, []domain.Event, domain.UserProfile) (domain.TaskGenerationDecision, error) {
	return s.result, s.err
}

type recordingActionabilityService struct {
	result   domain.TaskGenerationDecision
	err      error
	received domain.UserProfile
}

func (s *recordingActionabilityService) GetTaskGenerationDecision(_ context.Context, _ []domain.Event, profile domain.UserProfile) (domain.TaskGenerationDecision, error) {
	s.received = profile
	return s.result, s.err
}

type stubTaskGenerationService struct {
	result domain.GeneratedTask
	err    error
}

func (s stubTaskGenerationService) GenerateTask(context.Context, []domain.Event, domain.UserProfile) (domain.GeneratedTask, error) {
	return s.result, s.err
}

type recordingTaskGenerationService struct {
	result   domain.GeneratedTask
	err      error
	received domain.UserProfile
}

func (s *recordingTaskGenerationService) GenerateTask(_ context.Context, _ []domain.Event, profile domain.UserProfile) (domain.GeneratedTask, error) {
	s.received = profile
	return s.result, s.err
}

type stubUserProfileService struct {
	profile domain.UserProfile
	err     error
}

func (s stubUserProfileService) GetUserProfile(context.Context, domain.UserID) (domain.UserProfile, error) {
	if s.err != nil {
		return domain.UserProfile{}, errors.WrapFail(s.err, "stub user profile")
	}
	return s.profile, nil
}

type stubTxProvider struct{}

func (stubTxProvider) RunWithTx(ctx context.Context, _ tx.Isolation, op func(context.Context) error) error {
	return op(ctx)
}

type stubSettingsProvider struct {
	settings domain.GenerationSettings
}

func (s stubSettingsProvider) GenerationSettings(context.Context) (domain.GenerationSettings, error) {
	return s.settings, nil
}

type stubModerationRepo struct {
	required bool
	err      error
}

func (s stubModerationRepo) RequiresModeration(context.Context, domain.UserID) (bool, error) {
	return s.required, s.err
}
func (s stubModerationRepo) AddUser(context.Context, domain.UserID) error    { return nil }
func (s stubModerationRepo) RemoveUser(context.Context, domain.UserID) error { return nil }
func (s stubModerationRepo) ListUsers(context.Context) ([]domain.UserID, error) {
	return nil, nil
}

type stubEmbeddingService struct {
	embeddings [][]float32
	err        error
}

func (s stubEmbeddingService) GenerateEmbeddings(context.Context, []string) ([][]float32, error) {
	return s.embeddings, s.err
}
