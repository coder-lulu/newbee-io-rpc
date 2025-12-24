package workermetrics

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetWorkerMetricsByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetWorkerMetricsByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWorkerMetricsByIdLogic {
	return &GetWorkerMetricsByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetWorkerMetricsByIdLogic) GetWorkerMetricsById(in *io.IDReq) (*io.WorkerMetricsInfo, error) {
	result, err := l.svcCtx.DB.WorkerMetrics.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &io.WorkerMetricsInfo{
		Id:          &result.ID,
		CreatedAt:    utils.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:    utils.GetPointer(result.UpdatedAt.UnixMilli()),
		WorkerId:	&result.WorkerID,
		WorkerName:	&result.WorkerName,
		WorkerStatus:	&result.WorkerStatus,
		CurrentTasks:	utils.GetPointer(int64(result.CurrentTasks)),
		TotalTasks:	utils.GetPointer(int64(result.TotalTasks)),
		SuccessTasks:	utils.GetPointer(int64(result.SuccessTasks)),
		FailedTasks:	utils.GetPointer(int64(result.FailedTasks)),
		CpuUsage:	&result.CPUUsage,
		MemoryUsage:	&result.MemoryUsage,
		LastHeartbeat:	utils.GetUnixMilliPointer(result.LastHeartbeat.UnixMilli()),
		Metadata:	&result.Metadata,
	}, nil
}

