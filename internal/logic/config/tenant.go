package config

import (
	"context"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func requireTenant(ctx context.Context) (uint64, error) {
	id := hooks.GetCurrentTenantID(ctx)
	if id == 0 {
		return 0, status.Error(codes.Unauthenticated, "tenant context required")
	}
	return id, nil
}
