package token

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"strings"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const InterceptAllMethodsOption = "all-methods"

func NewUnaryTokenInterceptor(
	provider Verifier,
	logger log.Logger,
	methods ...string,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		if !slices.Contains(methods, InterceptAllMethodsOption) && !slices.Contains(methods, info.FullMethod) {
			return handler(ctx, req)
		}

		tokenRaw, err := fromGRPCCtx(ctx)
		if err != nil {
			logger.Warn(errors.WrapFail(err, "extract user token from context"))
			return nil, status.Error(codes.Unauthenticated, "valid user token is not provided")
		}

		ok, err := provider.VerifyToken(ctx, tokenRaw)
		if err != nil {
			return nil, errors.WrapFail(err, "verify token")
		}
		if !ok {
			logger.Warn(errors.Error("token is invalid"))
			return nil, status.Error(codes.Unauthenticated, "token is invalid")
		}

		claims, err := parseClaimsInsecure(tokenRaw)
		if err != nil {
			return Token{}, errors.WrapFail(err, "parse claims")
		}

		userIdClaim, ok := claims["user_id"]
		if !ok {
			return Token{}, status.Error(codes.Unauthenticated, "user id is not found in token")
		}

		userIdStr, ok := userIdClaim.(string)
		if !ok {
			return Token{}, status.Error(codes.Unauthenticated, "user id invalid format")
		}

		userID, err := uuid.Parse(userIdStr)
		if err != nil {
			return Token{}, status.Error(codes.Unauthenticated, "user id invalid format")
		}

		return handler(ContextWithToken(ctx, Token{
			UserID: userID,
		}), req)
	}
}

func parseClaimsInsecure(rawToken string) (map[string]any, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return nil, errors.Errorf(
			"invalid JWT: expected 3 parts, got %s",
			errors.Token("got", len(parts)),
		)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.WrapFail(err, "base64-decode payload")
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errors.WrapFail(err, "unmarshal claims")
	}

	return claims, nil
}
