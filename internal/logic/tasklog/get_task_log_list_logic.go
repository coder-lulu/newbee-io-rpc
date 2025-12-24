package tasklog

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/tasklog"
	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
    "github.com/zeromicro/go-zero/core/logx"
)

type GetTaskLogListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTaskLogListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskLogListLogic {
	return &GetTaskLogListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetTaskLogListLogic) GetTaskLogList(in *io.TaskLogListReq) (*io.TaskLogListResp, error) {
	var predicates []predicate.TaskLog
	if in.CreatedAt != nil {
		predicates = append(predicates, tasklog.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, tasklog.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.TaskType != nil {
		predicates = append(predicates, tasklog.TaskTypeContains(*in.TaskType))
	}
	if in.TaskId != nil {
		predicates = append(predicates, tasklog.TaskIDEQ(*in.TaskId))
	}
	if in.LogLevel != nil {
		predicates = append(predicates, tasklog.LogLevelContains(*in.LogLevel))
	}
	if in.LogMessage != nil {
		predicates = append(predicates, tasklog.LogMessageContains(*in.LogMessage))
	}
	if in.LogDetail != nil {
		predicates = append(predicates, tasklog.LogDetailContains(*in.LogDetail))
	}
	if in.LoggedAt != nil {
		predicates = append(predicates, tasklog.LoggedAtGTE(time.UnixMilli(*in.LoggedAt)))
	}
	result, err := l.svcCtx.DB.TaskLog.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.TaskLogListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &io.TaskLogInfo{
			Id:          &v.ID,
			CreatedAt:   utils.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:   utils.GetPointer(v.UpdatedAt.UnixMilli()),
			TaskType:	&v.TaskType,
			TaskId:	&v.TaskID,
			LogLevel:	&v.LogLevel,
			LogMessage:	&v.LogMessage,
			LogDetail:	&v.LogDetail,
			LoggedAt:	utils.GetPointer(v.LoggedAt.UnixMilli()),
		})
	}

	return resp, nil
}
