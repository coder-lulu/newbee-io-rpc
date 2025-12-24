package validator

import (
	"context"
	"reflect"

	"github.com/zeromicro/go-zero/core/errorx"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type validatable interface {
	Validate() error
}

// UnaryServerInterceptor 返回一个gRPC拦截器，用于验证请求参数
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := validateRequest(req); err != nil {
			logx.WithContext(ctx).Errorf("请求参数验证失败: %v", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor 返回一个gRPC流拦截器，用于验证流请求
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := &recvWrapper{
			ServerStream: ss,
		}

		return handler(srv, wrapper)
	}
}

// 请求验证
func validateRequest(req interface{}) error {
	if req == nil {
		return nil
	}

	// 检查请求是否实现了validatable接口
	if v, ok := req.(validatable); ok {
		if err := v.Validate(); err != nil {
			return errorx.NewInvalidArgumentError(err.Error())
		}
		return nil
	}

	// 如果没有实现接口，使用反射检查是否有ValidateAll方法
	rv := reflect.ValueOf(req)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		// 使用自动验证
		return Validate(req)
	}

	return nil
}

// recvWrapper 包装grpc.ServerStream，用于验证每个接收到的消息
type recvWrapper struct {
	grpc.ServerStream
}

// RecvMsg 重写接收消息方法，添加验证
func (s *recvWrapper) RecvMsg(m interface{}) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}

	if err := validateRequest(m); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	return nil
}
