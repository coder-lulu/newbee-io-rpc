package service

import (
	"context"
	"github.com/coder-lulu/newbee-common/v2/middleware/keys"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"google.golang.org/grpc/metadata"
	"testing"
	"time"
)

func TestAuditContextPreservesTenantAfterCancellation(t *testing.T) {
	incoming := metadata.NewIncomingContext(context.Background(), metadata.Pairs(string(keys.TenantIDKey), "23"))
	ctx, cancel := context.WithCancel(incoming)
	cancel()
	audit, stop := newAuditContext(ctx)
	defer stop()
	if audit.Err() != nil || hooks.GetCurrentTenantID(audit) != 23 {
		t.Fatal("audit lost tenant or inherited caller cancellation")
	}
	deadline, ok := audit.Deadline()
	if !ok || time.Until(deadline) > 10*time.Second {
		t.Fatal("audit context must be bounded")
	}
	if hooks.IsSystemContext(audit) {
		t.Fatal("audit must not use system context")
	}
}
