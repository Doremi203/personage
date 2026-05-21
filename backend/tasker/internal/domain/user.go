package domain

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
)

var ErrUserProfileNotFound = errors.Error("user profile not found")

type UserProfile struct {
	Email string
	Name  string
}

//go:generate mockgen -source=user.go -destination=mock/user_mock.go -typed

type UserProfileService interface {
	GetUserProfile(ctx context.Context, userID UserID) (UserProfile, error)
}
