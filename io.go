package main

import (
	"flag"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/internal/config"
	"github.com/coder-lulu/newbee-io-rpc/internal/server"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/prometheus"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/io.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	ctx := svc.NewServiceContext(c)

	// The service context owns the metrics/health HTTP listener; keep RPC metrics
	// enabled without starting a second Prometheus listener on the same address.
	prometheus.Enable()
	rpcConf := c.RpcServerConf
	rpcConf.Prometheus.Host = ""
	s := zrpc.MustNewServer(rpcConf, func(grpcServer *grpc.Server) {
		io.RegisterIoServer(grpcServer, server.NewIoServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
