package crontask

import (
	"context"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/suyuan32/simple-admin-common/msg/errormsg"
	"github.com/zeromicro/go-zero/core/logx"
)

type TriggerCronTaskNowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTriggerCronTaskNowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TriggerCronTaskNowLogic {
	return &TriggerCronTaskNowLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// TriggerCronTaskNow manually triggers a CronTask execution immediately
// 立即手动触发一个 CronTask 的执行（创建一个 InputTask 实例）
func (l *TriggerCronTaskNowLogic) TriggerCronTaskNow(in *io.IDReq) (*io.BaseIDResp, error) {
	// 1. 验证参数
	if in.Id == 0 {
		return nil, fmt.Errorf("id is required")
	}

	logx.Infow("Manually triggering CronTask",
		logx.Field("id", in.Id))

	// 2. 查询 CronTask
	cronTask, err := l.svcCtx.DB.CronTask.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	// 3. 创建 InputTask 实例（手动触发）
	executionTime := time.Now()
	inputTask, err := l.svcCtx.DB.InputTask.Create().
		SetTaskName(fmt.Sprintf("%s - Manual Trigger - %s", cronTask.TaskName, executionTime.Format("2006-01-02 15:04:05"))).
		SetTaskType("cron_manual"). // 标记为手动触发的cron任务
		SetInputSource(cronTask.InputSource).
		SetSourceConfig(cronTask.SourceConfig).
		SetTaskStatus("pending").
		SetCronTaskID(cronTask.ID).
		SetExecutionTime(executionTime).
		SetStatus(1).
		Save(l.ctx)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	logx.Infow("InputTask created for manual CronTask trigger",
		logx.Field("cron_task_id", cronTask.ID),
		logx.Field("input_task_id", inputTask.ID),
		logx.Field("execution_time", executionTime.Format(time.RFC3339)))

	// 4. 更新 CronTask 的执行统计（execution_count）
	_, err = l.svcCtx.DB.CronTask.UpdateOneID(cronTask.ID).
		SetExecutionCount(cronTask.ExecutionCount + 1).
		SetLastRunTime(executionTime).
		Save(l.ctx)

	if err != nil {
		logx.Errorw("Failed to update CronTask execution count",
			logx.Field("cron_task_id", cronTask.ID),
			logx.Field("error", err))
		// 不返回错误，InputTask已创建成功
	}

	return &io.BaseIDResp{
		Id:  inputTask.ID,
		Msg: errormsg.CreateSuccess,
	}, nil
}
