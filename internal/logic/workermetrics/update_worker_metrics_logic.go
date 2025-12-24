package workermetrics

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateWorkerMetricsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateWorkerMetricsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateWorkerMetricsLogic {
	return &UpdateWorkerMetricsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateWorkerMetricsLogic) UpdateWorkerMetrics(in *io.WorkerMetricsInfo) (*io.BaseResp, error) {
	query:= l.svcCtx.DB.WorkerMetrics.UpdateOneID(*in.Id).
			SetNotNilWorkerID(in.WorkerId).
			SetNotNilWorkerName(in.WorkerName).
			SetNotNilWorkerStatus(in.WorkerStatus).
			SetNotNilCPUUsage(in.CpuUsage).
			SetNotNilMemoryUsage(in.MemoryUsage).
			SetNotNilLastHeartbeat(utils.GetTimeMilliPointer(in.LastHeartbeat)).
			SetNotNilMetadata(in.Metadata)

	if in.CurrentTasks != nil {
		query.SetNotNilCurrentTasks(utils.GetPointer(int(*in.CurrentTasks)))
	}
	if in.TotalTasks != nil {
		query.SetNotNilTotalTasks(utils.GetPointer(int(*in.TotalTasks)))
	}
	if in.SuccessTasks != nil {
		query.SetNotNilSuccessTasks(utils.GetPointer(int(*in.SuccessTasks)))
	}
	if in.FailedTasks != nil {
		query.SetNotNilFailedTasks(utils.GetPointer(int(*in.FailedTasks)))
	}

	 err := query.Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.UpdateSuccess }, nil
}
