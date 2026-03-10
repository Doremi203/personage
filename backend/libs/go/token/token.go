package token

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"google.golang.org/grpc/metadata"

	"github.com/google/uuid"
)

var (
	ErrTokenNotFound = errors.Error("token not found")
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

func FromGRPCCtx(
	ctx context.Context,
) (Token, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Token{}, errors.Error("failed to extract grpc metadata")
	}

	tokens := md.Get("user-token")
	if len(tokens) == 0 {
		return Token{}, errors.Error("failed to find user-token in grpc metadata")
	}

	userID, err := uuid.Parse(tokens[0])
	if err != nil {
		return Token{}, errors.WrapFail(err, "failed to parse user-token")
	}

	return Token{
		UserID: userID,
	}, nil
}
