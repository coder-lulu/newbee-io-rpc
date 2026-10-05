package workermetrics

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/ent/workermetrics"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetWorkerMetricsListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetWorkerMetricsListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWorkerMetricsListLogic {
	return &GetWorkerMetricsListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetWorkerMetricsListLogic) GetWorkerMetricsList(in *io.WorkerMetricsListReq) (*io.WorkerMetricsListResp, error) {
	var predicates []predicate.WorkerMetrics
	if in.CreatedAt != nil {
		predicates = append(predicates, workermetrics.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, workermetrics.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.WorkerId != nil {
		predicates = append(predicates, workermetrics.WorkerIDContains(*in.WorkerId))
	}
	if in.WorkerName != nil {
		predicates = append(predicates, workermetrics.WorkerNameContains(*in.WorkerName))
	}
	if in.WorkerStatus != nil {
		predicates = append(predicates, workermetrics.WorkerStatusContains(*in.WorkerStatus))
	}
	if in.CurrentTasks != nil {
		predicates = append(predicates, workermetrics.CurrentTasksEQ(int(*in.CurrentTasks)))
	}
	if in.TotalTasks != nil {
		predicates = append(predicates, workermetrics.TotalTasksEQ(int(*in.TotalTasks)))
	}
	if in.SuccessTasks != nil {
		predicates = append(predicates, workermetrics.SuccessTasksEQ(int(*in.SuccessTasks)))
	}
	if in.FailedTasks != nil {
		predicates = append(predicates, workermetrics.FailedTasksEQ(int(*in.FailedTasks)))
	}
	if in.CpuUsage != nil {
		predicates = append(predicates, workermetrics.CPUUsageEQ(*in.CpuUsage))
	}
	if in.MemoryUsage != nil {
		predicates = append(predicates, workermetrics.MemoryUsageEQ(*in.MemoryUsage))
	}
	if in.LastHeartbeat != nil {
		predicates = append(predicates, workermetrics.LastHeartbeatGTE(time.UnixMilli(*in.LastHeartbeat)))
	}
	if in.Metadata != nil {
		predicates = append(predicates, workermetrics.MetadataContains(*in.Metadata))
	}
	result, err := l.svcCtx.DB.WorkerMetrics.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.WorkerMetricsListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &io.WorkerMetricsInfo{
			Id:            &v.ID,
			CreatedAt:     utils.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:     utils.GetPointer(v.UpdatedAt.UnixMilli()),
			WorkerId:      &v.WorkerID,
			WorkerName:    &v.WorkerName,
			WorkerStatus:  &v.WorkerStatus,
			CurrentTasks:  utils.GetPointer(int64(v.CurrentTasks)),
			TotalTasks:    utils.GetPointer(int64(v.TotalTasks)),
			SuccessTasks:  utils.GetPointer(int64(v.SuccessTasks)),
			FailedTasks:   utils.GetPointer(int64(v.FailedTasks)),
			CpuUsage:      &v.CPUUsage,
			MemoryUsage:   &v.MemoryUsage,
			LastHeartbeat: utils.TimeToUnixMilli(v.LastHeartbeat),
			Metadata:      &v.Metadata,
		})
	}

	return resp, nil
}
