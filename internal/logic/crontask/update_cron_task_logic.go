package crontask

import (
	"context"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/robfig/cron/v3"
	"github.com/suyuan32/simple-admin-common/msg/errormsg"
	"github.com/suyuan32/simple-admin-common/utils/pointy"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCronTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCronTaskLogic {
	return &UpdateCronTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateCronTaskLogic) UpdateCronTask(in *io.CronTaskInfo) (*io.BaseResp, error) {
	// 1. 验证必填字段
	if in.Id == nil {
		return nil, fmt.Errorf("id is required")
	}

	// 2. 如果更新了Cron表达式，需要验证并重新计算next_run_time
	var nextRunTime *time.Time
	if in.CronExpression != nil && *in.CronExpression != "" {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(*in.CronExpression)
		if err != nil {
			logx.Errorw("Invalid cron expression in update",
				logx.Field("id", *in.Id),
				logx.Field("cron_expression", *in.CronExpression),
				logx.Field("error", err))
			return nil, fmt.Errorf("invalid cron expression '%s': %w", *in.CronExpression, err)
		}

		// 重新计算下次执行时间
		now := time.Now().UTC()
		next := schedule.Next(now)
		nextRunTime = &next

		logx.Infow("Recalculated next_run_time for updated cron expression",
			logx.Field("id", *in.Id),
			logx.Field("cron_expression", *in.CronExpression),
			logx.Field("next_run_time", next.Format(time.RFC3339)))
	}

	// 3. 构建更新查询
	query := l.svcCtx.DB.CronTask.UpdateOneID(*in.Id).
		SetNotNilTaskName(in.TaskName).
		SetNotNilCronExpression(in.CronExpression).
		SetNotNilInputSource(in.InputSource).
		SetNotNilSourceConfig(in.SourceConfig).
		SetNotNilEnabled(in.Enabled).
		SetNotNilLastRunTime(pointy.GetTimeMilliPointer(in.LastRunTime)).
		SetNotNilDescription(in.Description)

	// 如果计算了新的next_run_time，设置它
	if nextRunTime != nil {
		query.SetNextRunTime(*nextRunTime)
	} else if in.NextRunTime != nil {
		// 否则使用提供的next_run_time（如果有）
		query.SetNotNilNextRunTime(pointy.GetTimeMilliPointer(in.NextRunTime))
	}

	// 4. 设置可选字段
	if in.Status != nil {
		query.SetNotNilStatus(pointy.GetPointer(uint8(*in.Status)))
	}
	if in.ExecutionCount != nil {
		query.SetNotNilExecutionCount(pointy.GetPointer(int(*in.ExecutionCount)))
	}
	if in.SuccessCount != nil {
		query.SetNotNilSuccessCount(pointy.GetPointer(int(*in.SuccessCount)))
	}
	if in.FailureCount != nil {
		query.SetNotNilFailureCount(pointy.GetPointer(int(*in.FailureCount)))
	}

	// 5. 执行更新
	err := query.Exec(l.ctx)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	logx.Infow("CronTask updated successfully",
		logx.Field("id", *in.Id))

	// 6. 如果CronScheduler可用，更新调度器中的任务
	// TODO: 集成CronScheduler
	// if l.svcCtx.CronScheduler != nil {
	//     // 重新查询完整的任务信息
	//     task, err := l.svcCtx.DB.CronTask.Get(l.ctx, *in.Id)
	//     if err != nil {
	//         logx.Errorw("Failed to query updated task",
	//             logx.Field("id", *in.Id),
	//             logx.Field("error", err))
	//     } else if task.Enabled {
	//         // 更新调度器（先删除后添加）
	//         if err := l.svcCtx.CronScheduler.UpdateTask(task); err != nil {
	//             logx.Errorw("Failed to update task in CronScheduler",
	//                 logx.Field("task_id", task.ID),
	//                 logx.Field("error", err))
	//         }
	//     } else {
	//         // 如果任务被禁用，从调度器移除
	//         l.svcCtx.CronScheduler.RemoveTask(*in.Id)
	//     }
	// }

	return &io.BaseResp{Msg: errormsg.UpdateSuccess}, nil
}
