package webapp

import (
	"context"
	"runtime/debug"

	"gitlab.com/amoguscorp/personage/backend/libs/go/errors"
	"gitlab.com/amoguscorp/personage/backend/libs/go/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewUnaryPanicInterceptor(logger log.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error(errors.Errorf("panic in %v with %v and %v",
					errors.Token("handler", info.FullMethod),
					errors.Token("error", r),
					errors.Token("stack", debug.Stack()),
				))
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}
