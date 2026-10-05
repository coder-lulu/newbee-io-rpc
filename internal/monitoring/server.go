package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zeromicro/go-zero/core/logx"
)

// PrometheusServer Prometheus HTTP服务器
type PrometheusServer struct {
	server  *http.Server
	host    string
	port    int
	path    string
	metrics *MetricsCollector
}

// NewPrometheusServer 创建Prometheus服务器
func NewPrometheusServer(host string, port int, path string, metrics *MetricsCollector) *PrometheusServer {
	return &PrometheusServer{
		host:    host,
		port:    port,
		path:    path,
		metrics: metrics,
	}
}

// Start 启动Prometheus服务器
func (s *PrometheusServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	mux := http.NewServeMux()
	mux.Handle(s.path, promhttp.Handler())

	// 添加健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logx.Infow("Starting Prometheus metrics server",
		logx.Field("address", addr),
		logx.Field("path", s.path))

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logx.Errorw("Prometheus server error",
				logx.Field("error", err))
		}
	}()

	return nil
}

// Stop 停止Prometheus服务器
func (s *PrometheusServer) Stop() error {
	if s.server == nil {
		return nil
	}

	logx.Info("Stopping Prometheus metrics server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown Prometheus server: %w", err)
	}

	logx.Info("Prometheus metrics server stopped")
	return nil
}

// Metrics 返回metrics收集器
func (s *PrometheusServer) Metrics() *MetricsCollector {
	return s.metrics
}
