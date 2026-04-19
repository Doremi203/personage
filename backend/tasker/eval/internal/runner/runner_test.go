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

func (s *stubTaskerClient) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}
