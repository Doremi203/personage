package userprofile

import (
	"context"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func NewStub() *stubService {
	return &stubService{}
}

type stubService struct{}

func (s *stubService) GetUserProfile(ctx context.Context, userID domain.UserID) (domain.UserProfile, error) {
	return domain.UserProfile{
		Email: "test@personage.local",
		Name:  "Test User",
	}, nil
}
