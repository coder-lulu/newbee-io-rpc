package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestLoadConfigWithPrometheus(t *testing.T) {
	const yaml = `
Name: io.rpc.test
ListenOn: 127.0.0.1:9105
DatabaseConf:
  Host: 127.0.0.1
  Port: 5432
  Type: postgres
RedisConf:
  Host: 127.0.0.1:6379
Prometheus:
  Host: 127.0.0.1
  Port: 4005
  Path: /io-metrics
`
	var c Config
	if err := conf.LoadFromYamlBytes([]byte(yaml), &c); err != nil {
		t.Fatalf("load RPC configuration: %v", err)
	}
	if c.Prometheus.Host != "127.0.0.1" || c.Prometheus.Port != 4005 || c.Prometheus.Path != "/io-metrics" {
		t.Fatalf("unexpected Prometheus configuration: %+v", c.Prometheus)
	}
}
