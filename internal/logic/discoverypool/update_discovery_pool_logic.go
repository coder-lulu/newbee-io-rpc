package discoverypool

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateDiscoveryPoolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateDiscoveryPoolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDiscoveryPoolLogic {
	return &UpdateDiscoveryPoolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateDiscoveryPoolLogic) UpdateDiscoveryPool(in *io.DiscoveryPoolInfo) (*io.BaseResp, error) {
	query:= l.svcCtx.DB.DiscoveryPool.UpdateOneID(*in.Id).
			SetNotNilName(in.Name).
			SetNotNilDescription(in.Description).
			SetNotNilDiscoveryType(in.DiscoveryType).
			SetNotNilPoolStatus(in.PoolStatus).
			SetNotNilDiscoveryConfig(in.DiscoveryConfig).
			SetNotNilSchedule(in.Schedule).
			SetNotNilFieldMapping(in.FieldMapping).
			SetNotNilTotalRuns(in.TotalRuns).
			SetNotNilSuccessRuns(in.SuccessRuns).
			SetNotNilFailedRuns(in.FailedRuns).
			SetNotNilLastRunAt(utils.GetTimeMilliPointer(in.LastRunAt)).
			SetNotNilLastSuccessAt(utils.GetTimeMilliPointer(in.LastSuccessAt)).
			SetNotNilLastError(in.LastError).
			SetNotNilApprovalStatus(in.ApprovalStatus).
			SetNotNilApprovedBy(in.ApprovedBy).
			SetNotNilApprovedAt(utils.GetTimeMilliPointer(in.ApprovedAt)).
			SetNotNilRejectionReason(in.RejectionReason).
			SetNotNilMetadata(in.Metadata)

	if in.Status != nil {
		query.SetNotNilStatus(utils.GetPointer(uint8(*in.Status)))
	}
	if in.BatchSize != nil {
		query.SetNotNilBatchSize(utils.GetPointer(int(*in.BatchSize)))
	}
	if in.ConcurrentLimit != nil {
		query.SetNotNilConcurrentLimit(utils.GetPointer(int(*in.ConcurrentLimit)))
	}
	if in.MaxRetry != nil {
		query.SetNotNilMaxRetry(utils.GetPointer(int(*in.MaxRetry)))
	}
	if in.RetryInterval != nil {
		query.SetNotNilRetryInterval(utils.GetPointer(int(*in.RetryInterval)))
	}

	 err := query.Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.UpdateSuccess }, nil
}
