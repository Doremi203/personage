package userprofile

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func TestCachedServiceHitWithinTTLSkipsUnderlying(t *testing.T) {
	inner := &recordingService{
		profile: domain.UserProfile{Email: "a@b.com", Name: "A"},
	}
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }

	svc := NewCachedService(inner, time.Minute, 10*time.Second, clock)

	if _, err := svc.GetUserProfile(t.Context(), "user-1"); err != nil {
		t.Fatalf("first GetUserProfile error: %v", err)
	}
	if _, err := svc.GetUserProfile(t.Context(), "user-1"); err != nil {
		t.Fatalf("second GetUserProfile error: %v", err)
	}

	if inner.calls != 1 {
		t.Fatalf("expected 1 underlying call, got %d", inner.calls)
	}
}

func TestCachedServiceReFetchesAfterTTLExpiry(t *testing.T) {
	inner := &recordingService{
		profile: domain.UserProfile{Email: "a@b.com", Name: "A"},
	}
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }

	svc := NewCachedService(inner, time.Minute, 10*time.Second, clock)

	if _, err := svc.GetUserProfile(t.Context(), "user-1"); err != nil {
		t.Fatalf("first error: %v", err)
	}

	now = now.Add(2 * time.Minute)

	if _, err := svc.GetUserProfile(t.Context(), "user-1"); err != nil {
		t.Fatalf("second error: %v", err)
	}

	if inner.calls != 2 {
		t.Fatalf("expected 2 underlying calls, got %d", inner.calls)
	}
}

func TestCachedServiceNegativeCachingWithSeparateTTL(t *testing.T) {
	inner := &recordingService{err: domain.ErrUserProfileNotFound}
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }

	svc := NewCachedService(inner, time.Minute, 10*time.Second, clock)

	if _, err := svc.GetUserProfile(t.Context(), "user-1"); !errors.Is(err, domain.ErrUserProfileNotFound) {
		t.Fatalf("expected not-found, got %v", err)
	}
	if _, err := svc.GetUserProfile(t.Context(), "user-1"); !errors.Is(err, domain.ErrUserProfileNotFound) {
		t.Fatalf("expected cached not-found, got %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("expected 1 underlying call within negative TTL, got %d", inner.calls)
	}

	now = now.Add(11 * time.Second)
	if _, err := svc.GetUserProfile(t.Context(), "user-1"); !errors.Is(err, domain.ErrUserProfileNotFound) {
		t.Fatalf("expected not-found after expiry, got %v", err)
	}
	if inner.calls != 2 {
		t.Fatalf("expected 2 underlying calls after expiry, got %d", inner.calls)
	}
}

func TestCachedServiceDoesNotCacheTransientErrors(t *testing.T) {
	transient := errors.Error("transient")
	inner := &recordingService{err: transient}
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }

	svc := NewCachedService(inner, time.Minute, 10*time.Second, clock)

	if _, err := svc.GetUserProfile(t.Context(), "user-1"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := svc.GetUserProfile(t.Context(), "user-1"); err == nil {
		t.Fatal("expected error")
	}
	if inner.calls != 2 {
		t.Fatalf("expected 2 underlying calls on transient errors, got %d", inner.calls)
	}
}

type recordingService struct {
	profile domain.UserProfile
	err     error
	calls   int
}

func (s *recordingService) GetUserProfile(context.Context, domain.UserID) (domain.UserProfile, error) {
	s.calls++
	if s.err != nil {
		return domain.UserProfile{}, s.err
	}
	return s.profile, nil
}
