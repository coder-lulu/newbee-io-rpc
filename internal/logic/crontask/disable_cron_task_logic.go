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

type DisableCronTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisableCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableCronTaskLogic {
	return &DisableCronTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DisableCronTask disables an enabled CronTask
// 禁用一个已启用的 CronTask，并将其从调度器移除
func (l *DisableCronTaskLogic) DisableCronTask(in *io.IDReq) (*io.BaseResp, error) {
	// 1. 验证参数
	if in.Id == 0 {
		return nil, fmt.Errorf("id is required")
	}

	logx.Infow("Disabling CronTask",
		logx.Field("id", in.Id))

	// 2. 查询任务
	task, err := l.svcCtx.DB.CronTask.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	// 3. 检查任务状态
	if !task.Enabled {
		logx.Infow("CronTask is already disabled",
			logx.Field("id", in.Id))
		return &io.BaseResp{Msg: "Task is already disabled"}, nil
	}

	// 4. 如果CronScheduler可用，先从调度器移除任务
	if l.svcCtx.CronScheduler != nil {
		if err := l.svcCtx.CronScheduler.RemoveTask(in.Id); err != nil {
			logx.Errorw("Failed to remove task from CronScheduler",
				logx.Field("task_id", in.Id),
				logx.Field("error", err))
			// 继续执行数据库更新，不因调度器错误而中断
		} else {
			logx.Infow("Task removed from CronScheduler successfully",
				logx.Field("task_id", in.Id))
		}
	}

	// 5. 更新enabled状态为false
	err = l.svcCtx.DB.CronTask.UpdateOneID(in.Id).
		SetEnabled(false).
		Exec(l.ctx)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	logx.Infow("CronTask disabled successfully",
		logx.Field("id", in.Id),
		logx.Field("task_name", task.TaskName))

	return &io.BaseResp{Msg: errormsg.UpdateSuccess}, nil
}
