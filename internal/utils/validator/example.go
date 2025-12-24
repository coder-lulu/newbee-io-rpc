package validator

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/types/cmdb"
	"github.com/zeromicro/go-zero/core/logx"
)

// 示例：在创建属性的Logic中使用验证器
type CreateAttributeLogicExample struct {
	ctx    context.Context
	logger logx.Logger
}

func (l *CreateAttributeLogicExample) CreateAttribute(in *cmdb.AttributeInfo) (*cmdb.BaseIDResp, error) {
	// 使用验证器验证请求参数
	if err := Validate(in); err != nil {
		l.logger.Errorf("参数验证失败: %v", err)
		return nil, err
	}

	// 验证通过后继续处理业务逻辑
	// ...

	return &cmdb.BaseIDResp{
		Id:  123,
		Msg: "创建成功",
	}, nil
}

// 示例：在Go-Zero的API服务中集成验证器中间件
/*
func main() {
	// ... 其他初始化代码

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 注册中间件
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 假设我们有一个请求包含JSON数据
			if r.Method == http.MethodPost || r.Method == http.MethodPut {
				// 具体的验证逻辑根据实际路由和请求在handler中处理
				// 这里只是一个通用的概念示例
			}
			next(w, r)
		}
	})

	// ... 注册路由和启动服务
}
*/

// 示例：在Go-Zero的RPC服务中集成验证器拦截器
/*
func main() {
	// ... 其他初始化代码

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		// 注册gRPC服务
		pb.RegisterServiceServer(grpcServer, server.NewServiceServer(ctx))
	})

	// 添加验证器拦截器
	s.AddUnaryInterceptors(UnaryServerInterceptor())

	// ... 启动服务
}
*/
