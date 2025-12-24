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

type CreateWorkerMetricsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateWorkerMetricsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateWorkerMetricsLogic {
	return &CreateWorkerMetricsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateWorkerMetricsLogic) CreateWorkerMetrics(in *io.WorkerMetricsInfo) (*io.BaseIDResp, error) {
    query := l.svcCtx.DB.WorkerMetrics.Create().
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

	result, err := query.Save(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseIDResp{Id: result.ID, Msg: errormsg.CreateSuccess }, nil
}
