package inputtask

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/inputtask"
	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
    "github.com/zeromicro/go-zero/core/logx"
)

type GetInputTaskListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetInputTaskListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInputTaskListLogic {
	return &GetInputTaskListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetInputTaskListLogic) GetInputTaskList(in *io.InputTaskListReq) (*io.InputTaskListResp, error) {
	var predicates []predicate.InputTask
	if in.CreatedAt != nil {
		predicates = append(predicates, inputtask.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, inputtask.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.Status != nil {
		predicates = append(predicates, inputtask.StatusEQ(uint8(*in.Status)))
	}
	if in.TaskName != nil {
		predicates = append(predicates, inputtask.TaskNameContains(*in.TaskName))
	}
	if in.TaskType != nil {
		predicates = append(predicates, inputtask.TaskTypeContains(*in.TaskType))
	}
	if in.InputSource != nil {
		predicates = append(predicates, inputtask.InputSourceContains(*in.InputSource))
	}
	if in.SourceConfig != nil {
		predicates = append(predicates, inputtask.SourceConfigContains(*in.SourceConfig))
	}
	if in.TaskStatus != nil {
		predicates = append(predicates, inputtask.TaskStatusContains(*in.TaskStatus))
	}
	if in.DiscoveryPoolId != nil {
		predicates = append(predicates, inputtask.DiscoveryPoolIDEQ(*in.DiscoveryPoolId))
	}
	if in.ScheduledAt != nil {
		predicates = append(predicates, inputtask.ScheduledAtGTE(time.UnixMilli(*in.ScheduledAt)))
	}
	if in.StartedAt != nil {
		predicates = append(predicates, inputtask.StartedAtGTE(time.UnixMilli(*in.StartedAt)))
	}
	if in.CompletedAt != nil {
		predicates = append(predicates, inputtask.CompletedAtGTE(time.UnixMilli(*in.CompletedAt)))
	}
	if in.TotalRecords != nil {
		predicates = append(predicates, inputtask.TotalRecordsEQ(*in.TotalRecords))
	}
	if in.ProcessedRecords != nil {
		predicates = append(predicates, inputtask.ProcessedRecordsEQ(*in.ProcessedRecords))
	}
	if in.SuccessRecords != nil {
		predicates = append(predicates, inputtask.SuccessRecordsEQ(*in.SuccessRecords))
	}
	if in.FailedRecords != nil {
		predicates = append(predicates, inputtask.FailedRecordsEQ(*in.FailedRecords))
	}
	if in.ErrorMessage != nil {
		predicates = append(predicates, inputtask.ErrorMessageContains(*in.ErrorMessage))
	}
	if in.Metadata != nil {
		predicates = append(predicates, inputtask.MetadataContains(*in.Metadata))
	}
	result, err := l.svcCtx.DB.InputTask.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.InputTaskListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &io.InputTaskInfo{
			Id:          &v.ID,
			CreatedAt:   utils.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:   utils.GetPointer(v.UpdatedAt.UnixMilli()),
			Status:	utils.GetPointer(uint32(v.Status)),
			TaskName:	&v.TaskName,
			TaskType:	&v.TaskType,
			InputSource:	&v.InputSource,
			SourceConfig:	&v.SourceConfig,
			TaskStatus:	&v.TaskStatus,
			DiscoveryPoolId:	v.DiscoveryPoolID,
			ScheduledAt:	utils.GetUnixMilliPointer(v.ScheduledAt.UnixMilli()),
			StartedAt:	utils.GetUnixMilliPointer(v.StartedAt.UnixMilli()),
			CompletedAt:	utils.GetUnixMilliPointer(v.CompletedAt.UnixMilli()),
			TotalRecords:	&v.TotalRecords,
			ProcessedRecords:	&v.ProcessedRecords,
			SuccessRecords:	&v.SuccessRecords,
			FailedRecords:	&v.FailedRecords,
			ErrorMessage:	&v.ErrorMessage,
			Metadata:	&v.Metadata,
		})
	}

	return resp, nil
}
