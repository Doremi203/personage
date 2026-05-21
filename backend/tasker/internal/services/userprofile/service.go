package userprofile

import (
	"context"

	authpb "github.com/Doremi203/personage/backend/libs/go/auth/gen/api/auth"
	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewGRPCService(client authpb.AuthServiceClient) *grpcService {
	return &grpcService{client: client}
}

type grpcService struct {
	client authpb.AuthServiceClient
}

func (s *grpcService) GetUserProfile(ctx context.Context, userID domain.UserID) (domain.UserProfile, error) {
	resp, err := s.client.GetUserProfile(ctx, &authpb.GetUserProfileRequest{
		UserId: userID.String(),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return domain.UserProfile{}, domain.ErrUserProfileNotFound
		}
		return domain.UserProfile{}, errors.WrapFailf(
			err,
			"call auth service GetUserProfile %s",
			errors.Token("user_id", userID.String()),
		)
	}

	return domain.UserProfile{
		Email: resp.GetEmail(),
		Name:  resp.GetName(),
	}, nil
}
