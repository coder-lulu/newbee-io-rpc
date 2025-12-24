package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/internal/provider"
	"github.com/zeromicro/go-zero/core/logx"
)

type DiscoveryEngine struct {
	db       *ent.Client
	registry *provider.ProviderRegistry
}

func NewDiscoveryEngine(db *ent.Client) *DiscoveryEngine {
	return &DiscoveryEngine{
		db:       db,
		registry: provider.GetRegistry(),
	}
}

type DiscoveryExecutionResult struct {
	Success        bool
	TotalRecords   int64
	SuccessRecords int64
	FailedRecords  int64
	ErrorMessage   string
	TaskID         uint64
}

func (e *DiscoveryEngine) ExecuteDiscovery(ctx context.Context, pool *ent.DiscoveryPool) (*DiscoveryExecutionResult, error) {
	logx.Infof("Starting discovery execution for pool: %d (%s)", pool.ID, pool.Name)

	var discoveryConfig map[string]interface{}
	if err := json.Unmarshal([]byte(pool.DiscoveryConfig), &discoveryConfig); err != nil {
		return &DiscoveryExecutionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Invalid discovery config: %v", err),
		}, nil
	}

	providerID, ok := discoveryConfig["provider_id"].(string)
	if !ok {
		return &DiscoveryExecutionResult{
			Success:      false,
			ErrorMessage: "provider_id not found in discovery config",
		}, nil
	}

	p, err := e.registry.Get(providerID)
	if err != nil {
		return &DiscoveryExecutionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Provider not found: %v", err),
		}, nil
	}

	config, ok := discoveryConfig["config"].(map[string]interface{})
	if !ok {
		return &DiscoveryExecutionResult{
			Success:      false,
			ErrorMessage: "config not found in discovery config",
		}, nil
	}

	if err := p.ValidateConfig(config); err != nil {
		return &DiscoveryExecutionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Config validation failed: %v", err),
		}, nil
	}

	task, err := e.createInputTask(ctx, pool)
	if err != nil {
		return &DiscoveryExecutionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to create input task: %v", err),
		}, nil
	}

	result := &DiscoveryExecutionResult{
		TaskID: task.ID,
	}

	if err := e.updateTaskStatus(ctx, task.ID, "running"); err != nil {
		logx.Errorf("Failed to update task status to running: %v", err)
	}

	discoveryResult, err := p.Discover(config)
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Discovery failed: %v", err)
		e.updateTaskStatus(ctx, task.ID, "failed")
		e.updateTaskError(ctx, task.ID, result.ErrorMessage)
		return result, nil
	}

	if !discoveryResult.Success {
		result.Success = false
		result.ErrorMessage = discoveryResult.ErrorMessage
		e.updateTaskStatus(ctx, task.ID, "failed")
		e.updateTaskError(ctx, task.ID, result.ErrorMessage)
		return result, nil
	}

	result.Success = true
	result.TotalRecords = discoveryResult.TotalRecords
	result.SuccessRecords = discoveryResult.TotalRecords
	result.FailedRecords = 0

	if err := e.saveDiscoveryResults(ctx, task.ID, pool.ID, discoveryResult.Records); err != nil {
		logx.Errorf("Failed to save discovery results: %v", err)
	}

	if err := e.updateTaskCompletion(ctx, task.ID, result); err != nil {
		logx.Errorf("Failed to update task completion: %v", err)
	}

	logx.Infof("Discovery completed for pool %d: %d records discovered", pool.ID, result.TotalRecords)

	return result, nil
}

func (e *DiscoveryEngine) saveDiscoveryResults(ctx context.Context, taskID, poolID uint64, records []map[string]interface{}) error {
	resultsJSON, err := json.Marshal(map[string]interface{}{
		"task_id":   taskID,
		"pool_id":   poolID,
		"records":   records,
		"count":     len(records),
		"timestamp": time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	task, err := e.db.InputTask.Get(ctx, taskID)
	if err != nil {
		return err
	}

	return e.db.InputTask.UpdateOne(task).
		SetMetadata(string(resultsJSON)).
		Exec(ctx)
}

func (e *DiscoveryEngine) createInputTask(ctx context.Context, pool *ent.DiscoveryPool) (*ent.InputTask, error) {
	now := time.Now()

	task, err := e.db.InputTask.Create().
		SetTaskName(fmt.Sprintf("Discovery - %s", pool.Name)).
		SetTaskType("triggered").
		SetInputSource("discovery_pool").
		SetTaskStatus("pending").
		SetDiscoveryPoolID(pool.ID).
		SetScheduledAt(now).
		SetStatus(1).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	return task, nil
}

func (e *DiscoveryEngine) updateTaskStatus(ctx context.Context, taskID uint64, status string) error {
	task, err := e.db.InputTask.Get(ctx, taskID)
	if err != nil {
		return err
	}

	update := e.db.InputTask.UpdateOne(task).SetTaskStatus(status)

	if status == "running" {
		now := time.Now()
		update = update.SetStartedAt(now)
	}

	return update.Exec(ctx)
}

func (e *DiscoveryEngine) updateTaskError(ctx context.Context, taskID uint64, errorMsg string) error {
	task, err := e.db.InputTask.Get(ctx, taskID)
	if err != nil {
		return err
	}

	now := time.Now()
	return e.db.InputTask.UpdateOne(task).
		SetErrorMessage(errorMsg).
		SetCompletedAt(now).
		Exec(ctx)
}

func (e *DiscoveryEngine) updateTaskCompletion(ctx context.Context, taskID uint64, result *DiscoveryExecutionResult) error {
	task, err := e.db.InputTask.Get(ctx, taskID)
	if err != nil {
		return err
	}

	now := time.Now()
	return e.db.InputTask.UpdateOne(task).
		SetTaskStatus("completed").
		SetCompletedAt(now).
		SetTotalRecords(result.TotalRecords).
		SetProcessedRecords(result.TotalRecords).
		SetSuccessRecords(result.SuccessRecords).
		SetFailedRecords(result.FailedRecords).
		Exec(ctx)
}

func (e *DiscoveryEngine) UpdatePoolStatistics(ctx context.Context, poolID uint64, success bool, errorMsg string) error {
	pool, err := e.db.DiscoveryPool.Get(ctx, poolID)
	if err != nil {
		return err
	}

	now := time.Now()
	update := e.db.DiscoveryPool.UpdateOne(pool).
		SetLastRunAt(now).
		SetTotalRuns(pool.TotalRuns + 1)

	if success {
		update = update.
			SetSuccessRuns(pool.SuccessRuns + 1).
			SetLastSuccessAt(now).
			SetLastError("")
	} else {
		update = update.
			SetFailedRuns(pool.FailedRuns + 1).
			SetLastError(errorMsg)
	}

	return update.Exec(ctx)
}
