package outputtask

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/outputtask"
	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetOutputTaskListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOutputTaskListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOutputTaskListLogic {
	return &GetOutputTaskListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetOutputTaskListLogic) GetOutputTaskList(in *io.OutputTaskListReq) (*io.OutputTaskListResp, error) {
	var predicates []predicate.OutputTask
	if in.CreatedAt != nil {
		predicates = append(predicates, outputtask.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, outputtask.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.Status != nil {
		predicates = append(predicates, outputtask.StatusEQ(uint8(*in.Status)))
	}
	if in.TaskName != nil {
		predicates = append(predicates, outputtask.TaskNameContains(*in.TaskName))
	}
	if in.TaskType != nil {
		predicates = append(predicates, outputtask.TaskTypeContains(*in.TaskType))
	}
	if in.OutputTarget != nil {
		predicates = append(predicates, outputtask.OutputTargetContains(*in.OutputTarget))
	}
	if in.TargetConfig != nil {
		predicates = append(predicates, outputtask.TargetConfigContains(*in.TargetConfig))
	}
	if in.TaskStatus != nil {
		predicates = append(predicates, outputtask.TaskStatusContains(*in.TaskStatus))
	}
	if in.DataTargetId != nil {
		predicates = append(predicates, outputtask.DataTargetIDEQ(*in.DataTargetId))
	}
	if in.ScheduledAt != nil {
		predicates = append(predicates, outputtask.ScheduledAtGTE(time.UnixMilli(*in.ScheduledAt)))
	}
	if in.StartedAt != nil {
		predicates = append(predicates, outputtask.StartedAtGTE(time.UnixMilli(*in.StartedAt)))
	}
	if in.CompletedAt != nil {
		predicates = append(predicates, outputtask.CompletedAtGTE(time.UnixMilli(*in.CompletedAt)))
	}
	if in.TotalRecords != nil {
		predicates = append(predicates, outputtask.TotalRecordsEQ(*in.TotalRecords))
	}
	if in.ProcessedRecords != nil {
		predicates = append(predicates, outputtask.ProcessedRecordsEQ(*in.ProcessedRecords))
	}
	if in.SuccessRecords != nil {
		predicates = append(predicates, outputtask.SuccessRecordsEQ(*in.SuccessRecords))
	}
	if in.FailedRecords != nil {
		predicates = append(predicates, outputtask.FailedRecordsEQ(*in.FailedRecords))
	}
	if in.ErrorMessage != nil {
		predicates = append(predicates, outputtask.ErrorMessageContains(*in.ErrorMessage))
	}
	if in.Metadata != nil {
		predicates = append(predicates, outputtask.MetadataContains(*in.Metadata))
	}
	result, err := l.svcCtx.DB.OutputTask.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.OutputTaskListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &io.OutputTaskInfo{
			Id:               &v.ID,
			CreatedAt:        utils.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:        utils.GetPointer(v.UpdatedAt.UnixMilli()),
			Status:           utils.GetPointer(uint32(v.Status)),
			TaskName:         &v.TaskName,
			TaskType:         &v.TaskType,
			OutputTarget:     &v.OutputTarget,
			TargetConfig:     &v.TargetConfig,
			TaskStatus:       &v.TaskStatus,
			DataTargetId:     v.DataTargetID,
			ScheduledAt:      utils.TimeToUnixMilli(v.ScheduledAt),
			StartedAt:        utils.TimeToUnixMilli(v.StartedAt),
			CompletedAt:      utils.TimeToUnixMilli(v.CompletedAt),
			TotalRecords:     &v.TotalRecords,
			ProcessedRecords: &v.ProcessedRecords,
			SuccessRecords:   &v.SuccessRecords,
			FailedRecords:    &v.FailedRecords,
			ErrorMessage:     &v.ErrorMessage,
			Metadata:         &v.Metadata,
		})
	}

	return resp, nil
}
