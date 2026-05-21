package userprofile

import (
	"context"
	"testing"

	authpb "github.com/Doremi203/personage/backend/libs/go/auth/gen/api/auth"
	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCServiceMapsSuccess(t *testing.T) {
	client := &stubAuthClient{response: &authpb.UserProfile{
		UserId: "user-1",
		Email:  "a@b.com",
		Name:   "A",
	}}
	svc := NewGRPCService(client)

	profile, err := svc.GetUserProfile(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("GetUserProfile error: %v", err)
	}

	if profile.Email != "a@b.com" || profile.Name != "A" {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}

func TestGRPCServiceMapsNotFoundToSentinel(t *testing.T) {
	client := &stubAuthClient{err: status.Error(codes.NotFound, "user not found")}
	svc := NewGRPCService(client)

	_, err := svc.GetUserProfile(t.Context(), "user-1")
	if !errors.Is(err, domain.ErrUserProfileNotFound) {
		t.Fatalf("expected ErrUserProfileNotFound, got %v", err)
	}
}

func TestGRPCServiceWrapsOtherErrors(t *testing.T) {
	client := &stubAuthClient{err: status.Error(codes.Unavailable, "down")}
	svc := NewGRPCService(client)

	_, err := svc.GetUserProfile(t.Context(), "user-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, domain.ErrUserProfileNotFound) {
		t.Fatalf("did not expect not-found sentinel, got %v", err)
	}
}

type stubAuthClient struct {
	authpb.AuthServiceClient
	response *authpb.UserProfile
	err      error
}

func (s *stubAuthClient) GetUserProfile(context.Context, *authpb.GetUserProfileRequest, ...grpc.CallOption) (*authpb.UserProfile, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.response, nil
}
