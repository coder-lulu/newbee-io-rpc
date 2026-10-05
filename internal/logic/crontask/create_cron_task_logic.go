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

type CreateCronTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateCronTaskLogic {
	return &CreateCronTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateCronTaskLogic) CreateCronTask(in *io.CronTaskInfo) (*io.BaseIDResp, error) {
	// 1. 验证必填字段
	if in.TaskName == nil || *in.TaskName == "" {
		return nil, fmt.Errorf("task_name is required")
	}
	if in.CronExpression == nil || *in.CronExpression == "" {
		return nil, fmt.Errorf("cron_expression is required")
	}
	if in.InputSource == nil || *in.InputSource == "" {
		return nil, fmt.Errorf("input_source is required")
	}

	// 2. 验证和解析 Cron 表达式
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(*in.CronExpression)
	if err != nil {
		logx.Errorw("Invalid cron expression",
			logx.Field("cron_expression", *in.CronExpression),
			logx.Field("error", err))
		return nil, fmt.Errorf("invalid cron expression '%s': %w", *in.CronExpression, err)
	}

	// 3. 计算下次执行时间（基于当前时间）
	now := time.Now().UTC()
	nextRunTime := schedule.Next(now)

	logx.Infow("Creating CronTask",
		logx.Field("task_name", *in.TaskName),
		logx.Field("cron_expression", *in.CronExpression),
		logx.Field("input_source", *in.InputSource),
		logx.Field("next_run_time", nextRunTime.Format(time.RFC3339)))

	// 4. 构建创建查询
	query := l.svcCtx.DB.CronTask.Create().
		SetNotNilTaskName(in.TaskName).
		SetNotNilCronExpression(in.CronExpression).
		SetNotNilInputSource(in.InputSource).
		SetNotNilSourceConfig(in.SourceConfig).
		SetNotNilDescription(in.Description).
		SetNextRunTime(nextRunTime) // 自动计算的下次执行时间

	// 5. 设置enabled状态（默认为true）
	if in.Enabled != nil {
		query.SetEnabled(*in.Enabled)
	} else {
		query.SetEnabled(true) // 默认启用
	}

	// 6. 设置可选字段
	if in.Status != nil {
		query.SetNotNilStatus(pointy.GetPointer(uint8(*in.Status)))
	}
	if in.LastRunTime != nil {
		query.SetNotNilLastRunTime(pointy.GetTimeMilliPointer(in.LastRunTime))
	}
	// 初始化执行统计字段为0（如果未提供）
	if in.ExecutionCount != nil {
		query.SetNotNilExecutionCount(pointy.GetPointer(int(*in.ExecutionCount)))
	} else {
		query.SetExecutionCount(0)
	}
	if in.SuccessCount != nil {
		query.SetNotNilSuccessCount(pointy.GetPointer(int(*in.SuccessCount)))
	} else {
		query.SetSuccessCount(0)
	}
	if in.FailureCount != nil {
		query.SetNotNilFailureCount(pointy.GetPointer(int(*in.FailureCount)))
	} else {
		query.SetFailureCount(0)
	}

	// 7. 保存到数据库
	result, err := query.Save(l.ctx)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	logx.Infow("CronTask created successfully",
		logx.Field("id", result.ID),
		logx.Field("task_name", result.TaskName),
		logx.Field("enabled", result.Enabled),
		logx.Field("next_run_time", result.NextRunTime.Format(time.RFC3339)))

	// 8. 如果任务启用且CronScheduler可用，添加到调度器
	if result.Enabled && l.svcCtx.CronScheduler != nil {
		if err := l.svcCtx.CronScheduler.AddTask(result); err != nil {
			logx.Errorw("Failed to add task to CronScheduler",
				logx.Field("task_id", result.ID),
				logx.Field("error", err))
			// 不返回错误，任务已保存到数据库，可以稍后手动添加到调度器
		} else {
			logx.Infow("Task added to CronScheduler successfully",
				logx.Field("task_id", result.ID))
		}
	}

	return &io.BaseIDResp{Id: result.ID, Msg: errormsg.CreateSuccess}, nil
}
