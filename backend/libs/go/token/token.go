package token

import (
	"context"
	"strings"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type Token struct {
	UserID uuid.UUID
}

func (t Token) GetUserID() uuid.UUID {
	return t.UserID
}

type tokenKey struct{}

func ContextWithToken(ctx context.Context, token Token) context.Context {
	return context.WithValue(ctx, tokenKey{}, token)
}

func FromContext(ctx context.Context) (Token, bool) {
	tx, ok := ctx.Value(tokenKey{}).(Token)
	return tx, ok
}

func fromGRPCCtx(
	ctx context.Context,
) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.Error("failed to extract grpc metadata")
	}

	authValues := md.Get("user-token")
	if len(authValues) == 0 {
		return "", errors.Error("failed to find user-token in grpc metadata")
	}

	rawToken := authValues[0]
	if strings.HasPrefix(rawToken, "Bearer ") {
		rawToken = strings.TrimPrefix(rawToken, "Bearer ")
	}

	return rawToken, nil
}
