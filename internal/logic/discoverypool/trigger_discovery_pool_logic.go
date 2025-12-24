package discoverypool

import (
	"context"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/internal/engine"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type TriggerDiscoveryPoolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTriggerDiscoveryPoolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TriggerDiscoveryPoolLogic {
	return &TriggerDiscoveryPoolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TriggerDiscoveryPoolLogic) TriggerDiscoveryPool(in *io.IDReq) (*io.BaseResp, error) {
	pool, err := l.svcCtx.DB.DiscoveryPool.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "DiscoveryPool not found"}, err
	}

	if pool.PoolStatus != "active" {
		return &io.BaseResp{Msg: "Pool is not active"}, nil
	}

	discoveryEngine := engine.NewDiscoveryEngine(l.svcCtx.DB)

	result, err := discoveryEngine.ExecuteDiscovery(l.ctx, pool)
	if err != nil {
		logx.Errorf("Discovery execution error: %v", err)
		discoveryEngine.UpdatePoolStatistics(l.ctx, pool.ID, false, err.Error())
		return &io.BaseResp{Msg: "Discovery execution failed"}, err
	}

	if !result.Success {
		logx.Errorf("Discovery failed: %s", result.ErrorMessage)
		discoveryEngine.UpdatePoolStatistics(l.ctx, pool.ID, false, result.ErrorMessage)
		return &io.BaseResp{Msg: result.ErrorMessage}, nil
	}

	if err := discoveryEngine.UpdatePoolStatistics(l.ctx, pool.ID, true, ""); err != nil {
		logx.Errorf("Failed to update pool statistics: %v", err)
	}

	msg := fmt.Sprintf("Discovery completed successfully. Task ID: %d, Records: %d", result.TaskID, result.TotalRecords)
	return &io.BaseResp{Msg: msg}, nil
}
