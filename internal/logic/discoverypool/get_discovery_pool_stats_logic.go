package discoverypool

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/ent/discoverypool"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDiscoveryPoolStatsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDiscoveryPoolStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDiscoveryPoolStatsLogic {
	return &GetDiscoveryPoolStatsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDiscoveryPoolStatsLogic) GetDiscoveryPoolStats(in *io.Empty) (*io.DiscoveryPoolStatsResp, error) {
	totalPools, err := l.svcCtx.DB.DiscoveryPool.Query().Count(l.ctx)
	if err != nil {
		return nil, err
	}

	activePools, err := l.svcCtx.DB.DiscoveryPool.Query().
		Where(discoverypool.PoolStatusEQ("active")).
		Count(l.ctx)
	if err != nil {
		return nil, err
	}

	runningPools, err := l.svcCtx.DB.DiscoveryPool.Query().
		Where(discoverypool.PoolStatusEQ("running")).
		Count(l.ctx)
	if err != nil {
		return nil, err
	}

	pools, err := l.svcCtx.DB.DiscoveryPool.Query().All(l.ctx)
	if err != nil {
		return nil, err
	}

	var totalDiscoveries, successfulDiscoveries, failedDiscoveries uint64
	for _, pool := range pools {
		totalDiscoveries += uint64(pool.TotalRuns)
		successfulDiscoveries += uint64(pool.SuccessRuns)
		failedDiscoveries += uint64(pool.FailedRuns)
	}

	return &io.DiscoveryPoolStatsResp{
		TotalPools:            uint64(totalPools),
		ActivePools:           uint64(activePools),
		RunningPools:          uint64(runningPools),
		TotalDiscoveries:      totalDiscoveries,
		SuccessfulDiscoveries: successfulDiscoveries,
		FailedDiscoveries:     failedDiscoveries,
	}, nil
}
