package discoverypool

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetDiscoveryPoolByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDiscoveryPoolByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDiscoveryPoolByIdLogic {
	return &GetDiscoveryPoolByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDiscoveryPoolByIdLogic) GetDiscoveryPoolById(in *io.IDReq) (*io.DiscoveryPoolInfo, error) {
	result, err := l.svcCtx.DB.DiscoveryPool.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &io.DiscoveryPoolInfo{
		Id:              &result.ID,
		CreatedAt:       utils.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:       utils.GetPointer(result.UpdatedAt.UnixMilli()),
		Status:          utils.GetPointer(uint32(result.Status)),
		Name:            &result.Name,
		Description:     &result.Description,
		DiscoveryType:   &result.DiscoveryType,
		PoolStatus:      &result.PoolStatus,
		DiscoveryConfig: &result.DiscoveryConfig,
		Schedule:        &result.Schedule,
		BatchSize:       utils.GetPointer(int64(result.BatchSize)),
		ConcurrentLimit: utils.GetPointer(int64(result.ConcurrentLimit)),
		MaxRetry:        utils.GetPointer(int64(result.MaxRetry)),
		RetryInterval:   utils.GetPointer(int64(result.RetryInterval)),
		FieldMapping:    &result.FieldMapping,
		TotalRuns:       &result.TotalRuns,
		SuccessRuns:     &result.SuccessRuns,
		FailedRuns:      &result.FailedRuns,
		LastRunAt:       utils.TimeToUnixMilli(result.LastRunAt),
		LastSuccessAt:   utils.TimeToUnixMilli(result.LastSuccessAt),
		LastError:       &result.LastError,
		ApprovalStatus:  &result.ApprovalStatus,
		ApprovedBy:      result.ApprovedBy,
		ApprovedAt:      utils.TimeToUnixMilli(result.ApprovedAt),
		RejectionReason: &result.RejectionReason,
		Metadata:        &result.Metadata,
	}, nil
}
