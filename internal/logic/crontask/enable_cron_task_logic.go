package crontask

import (
	"context"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/suyuan32/simple-admin-common/msg/errormsg"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnableCronTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEnableCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableCronTaskLogic {
	return &EnableCronTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// EnableCronTask enables a disabled CronTask
// 启用一个已禁用的 CronTask，并将其添加到调度器
func (l *EnableCronTaskLogic) EnableCronTask(in *io.IDReq) (*io.BaseResp, error) {
	// 1. 验证参数
	if in.Id == 0 {
		return nil, fmt.Errorf("id is required")
	}

	logx.Infow("Enabling CronTask",
		logx.Field("id", in.Id))

	// 2. 查询任务
	task, err := l.svcCtx.DB.CronTask.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	// 3. 检查任务状态
	if task.Enabled {
		logx.Infow("CronTask is already enabled",
			logx.Field("id", in.Id))
		return &io.BaseResp{Msg: "Task is already enabled"}, nil
	}

	// 4. 更新enabled状态为true
	err = l.svcCtx.DB.CronTask.UpdateOneID(in.Id).
		SetEnabled(true).
		Exec(l.ctx)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	logx.Infow("CronTask enabled successfully",
		logx.Field("id", in.Id),
		logx.Field("task_name", task.TaskName))

	// 5. 如果CronScheduler可用，添加任务到调度器
	if l.svcCtx.CronScheduler != nil {
		// 重新查询任务（包含更新后的enabled状态）
		task, err = l.svcCtx.DB.CronTask.Get(l.ctx, in.Id)
		if err != nil {
			logx.Errorw("Failed to query task after enabling",
				logx.Field("id", in.Id),
				logx.Field("error", err))
		} else {
			if err := l.svcCtx.CronScheduler.AddTask(task); err != nil {
				logx.Errorw("Failed to add task to CronScheduler",
					logx.Field("task_id", task.ID),
					logx.Field("error", err))
				return nil, fmt.Errorf("task enabled in database but failed to add to scheduler: %w", err)
			} else {
				logx.Infow("Task added to CronScheduler successfully",
					logx.Field("task_id", task.ID))
			}
		}
	}

	return &io.BaseResp{Msg: errormsg.UpdateSuccess}, nil
}
