package tasklog

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetTaskLogByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTaskLogByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskLogByIdLogic {
	return &GetTaskLogByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetTaskLogByIdLogic) GetTaskLogById(in *io.IDReq) (*io.TaskLogInfo, error) {
	result, err := l.svcCtx.DB.TaskLog.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &io.TaskLogInfo{
		Id:          &result.ID,
		CreatedAt:    utils.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:    utils.GetPointer(result.UpdatedAt.UnixMilli()),
		TaskType:	&result.TaskType,
		TaskId:	&result.TaskID,
		LogLevel:	&result.LogLevel,
		LogMessage:	&result.LogMessage,
		LogDetail:	&result.LogDetail,
		LoggedAt:	utils.GetPointer(result.LoggedAt.UnixMilli()),
	}, nil
}

