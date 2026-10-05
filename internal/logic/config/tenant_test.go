package config

import (
	"context"
	"github.com/coder-lulu/newbee-common/v2/middleware/keys"
	"github.com/coder-lulu/newbee-io-rpc/types/io"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
)

func TestRequireTenant(t *testing.T) {
	for _, ctx := range []context.Context{context.Background(), context.WithValue(context.Background(), keys.TenantIDKey, "0"), metadata.NewIncomingContext(context.Background(), metadata.Pairs(string(keys.TenantIDKey), "invalid"))} {
		if _, err := requireTenant(ctx); status.Code(err) != codes.Unauthenticated {
			t.Fatal("missing tenant accepted")
		}
	}
	for _, ctx := range []context.Context{context.WithValue(context.Background(), keys.TenantIDKey, "23"), metadata.NewIncomingContext(context.Background(), metadata.Pairs(string(keys.TenantIDKey), "23"))} {
		id, err := requireTenant(ctx)
		if err != nil || id != 23 {
			t.Fatalf("tenant not preserved: %d %v", id, err)
		}
	}
}
func TestConfigReadsRejectMissingTenantBeforeDatabase(t *testing.T) {
	ctx := context.Background()
	if _, err := NewListConfigLogic(ctx, nil).ListConfig(&io.ListConfigReq{}); status.Code(err) != codes.Unauthenticated {
		t.Fatal(err)
	}
	if _, err := NewListAuditLogLogic(ctx, nil).ListAuditLog(&io.ListAuditLogReq{}); status.Code(err) != codes.Unauthenticated {
		t.Fatal(err)
	}
	key := "demo.test"
	if _, err := NewGetConfigHistoryLogic(ctx, nil).GetConfigHistory(&io.GetConfigHistoryReq{ConfigKey: &key}); status.Code(err) != codes.Unauthenticated {
		t.Fatal(err)
	}
}
