package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
)

func TestNewStateMachine(t *testing.T) {
	logger := logx.WithContext(nil)
	sm := NewStateMachine(nil, logger)

	assert.NotNil(t, sm)
	assert.NotNil(t, sm.logger)
}

func TestStateMachine_ValidateTransition_PendingToRunning(t *testing.T) {
	sm := &StateMachine{}

	err := sm.ValidateTransition(TaskStatusPending, TaskStatusRunning)
	assert.NoError(t, err, "Transition from pending to running should be valid")
}

func TestStateMachine_ValidateTransition_PendingToCancelled(t *testing.T) {
	sm := &StateMachine{}

	err := sm.ValidateTransition(TaskStatusPending, TaskStatusCancelled)
	assert.NoError(t, err, "Transition from pending to cancelled should be valid")
}

func TestStateMachine_ValidateTransition_PendingToCompleted_Invalid(t *testing.T) {
	sm := &StateMachine{}

	err := sm.ValidateTransition(TaskStatusPending, TaskStatusCompleted)
	assert.Error(t, err, "Transition from pending to completed should be invalid")
	assert.Contains(t, err.Error(), "invalid state transition")
}

func TestStateMachine_ValidateTransition_RunningToCompleted(t *testing.T) {
	sm := &StateMachine{}

	err := sm.ValidateTransition(TaskStatusRunning, TaskStatusCompleted)
	assert.NoError(t, err, "Transition from running to completed should be valid")
}

func TestStateMachine_ValidateTransition_RunningToFailed(t *testing.T) {
	sm := &StateMachine{}

	err := sm.ValidateTransition(TaskStatusRunning, TaskStatusFailed)
	assert.NoError(t, err, "Transition from running to failed should be valid")
}

func TestStateMachine_ValidateTransition_RunningToCancelled(t *testing.T) {
	sm := &StateMachine{}

	err := sm.ValidateTransition(TaskStatusRunning, TaskStatusCancelled)
	assert.NoError(t, err, "Transition from running to cancelled should be valid")
}

func TestStateMachine_ValidateTransition_RunningToPending_Invalid(t *testing.T) {
	sm := &StateMachine{}

	err := sm.ValidateTransition(TaskStatusRunning, TaskStatusPending)
	assert.Error(t, err, "Transition from running to pending should be invalid")
}

func TestStateMachine_ValidateTransition_CompletedTransitions_Invalid(t *testing.T) {
	sm := &StateMachine{}

	// Completed是终态，不能转换到任何其他状态
	testCases := []TaskStatus{
		TaskStatusPending,
		TaskStatusRunning,
		TaskStatusFailed,
		TaskStatusCancelled,
	}

	for _, targetStatus := range testCases {
		err := sm.ValidateTransition(TaskStatusCompleted, targetStatus)
		assert.Error(t, err, "Completed is terminal state, should not transition to %s", targetStatus)
	}
}

func TestStateMachine_ValidateTransition_FailedToPending_Retry(t *testing.T) {
	sm := &StateMachine{}

	// Failed可以转换到Pending（用于重试）
	err := sm.ValidateTransition(TaskStatusFailed, TaskStatusPending)
	assert.NoError(t, err, "Transition from failed to pending should be valid (retry)")
}

func TestStateMachine_ValidateTransition_FailedToOthers_Invalid(t *testing.T) {
	sm := &StateMachine{}

	// Failed只能转换到Pending（重试），不能转换到其他状态
	invalidTransitions := []TaskStatus{
		TaskStatusRunning,
		TaskStatusCompleted,
		TaskStatusCancelled,
	}

	for _, targetStatus := range invalidTransitions {
		err := sm.ValidateTransition(TaskStatusFailed, targetStatus)
		assert.Error(t, err, "Transition from failed to %s should be invalid", targetStatus)
	}
}

func TestStateMachine_ValidateTransition_CancelledTransitions_Invalid(t *testing.T) {
	sm := &StateMachine{}

	// Cancelled是终态，不能转换到任何其他状态
	testCases := []TaskStatus{
		TaskStatusPending,
		TaskStatusRunning,
		TaskStatusCompleted,
		TaskStatusFailed,
	}

	for _, targetStatus := range testCases {
		err := sm.ValidateTransition(TaskStatusCancelled, targetStatus)
		assert.Error(t, err, "Cancelled is terminal state, should not transition to %s", targetStatus)
	}
}

func TestStateMachine_ValidateTransition_UnknownSourceState(t *testing.T) {
	sm := &StateMachine{}

	err := sm.ValidateTransition(TaskStatus("unknown"), TaskStatusRunning)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown source state")
}

func TestStateMachine_ValidateTransition_AllValidTransitions(t *testing.T) {
	sm := &StateMachine{}

	// 定义所有合法的状态转换
	validTransitions := map[TaskStatus][]TaskStatus{
		TaskStatusPending: {
			TaskStatusRunning,
			TaskStatusCancelled,
		},
		TaskStatusRunning: {
			TaskStatusCompleted,
			TaskStatusFailed,
			TaskStatusCancelled,
		},
		TaskStatusCompleted: {}, // 终态
		TaskStatusFailed: {
			TaskStatusPending, // 允许重试
		},
		TaskStatusCancelled: {}, // 终态
	}

	// 测试所有合法转换
	for from, toStates := range validTransitions {
		for _, to := range toStates {
			err := sm.ValidateTransition(from, to)
			assert.NoError(t, err, "Transition %s -> %s should be valid", from, to)
		}
	}
}

func TestStateMachine_ValidateTransition_StateTransitionMatrix(t *testing.T) {
	sm := &StateMachine{}

	// 定义期望的转换结果矩阵
	// true = 合法转换, false = 非法转换
	expectedMatrix := map[TaskStatus]map[TaskStatus]bool{
		TaskStatusPending: {
			TaskStatusPending:   false,
			TaskStatusRunning:   true,
			TaskStatusCompleted: false,
			TaskStatusFailed:    false,
			TaskStatusCancelled: true,
		},
		TaskStatusRunning: {
			TaskStatusPending:   false,
			TaskStatusRunning:   false,
			TaskStatusCompleted: true,
			TaskStatusFailed:    true,
			TaskStatusCancelled: true,
		},
		TaskStatusCompleted: {
			TaskStatusPending:   false,
			TaskStatusRunning:   false,
			TaskStatusCompleted: false,
			TaskStatusFailed:    false,
			TaskStatusCancelled: false,
		},
		TaskStatusFailed: {
			TaskStatusPending:   true,
			TaskStatusRunning:   false,
			TaskStatusCompleted: false,
			TaskStatusFailed:    false,
			TaskStatusCancelled: false,
		},
		TaskStatusCancelled: {
			TaskStatusPending:   false,
			TaskStatusRunning:   false,
			TaskStatusCompleted: false,
			TaskStatusFailed:    false,
			TaskStatusCancelled: false,
		},
	}

	// 遍历所有状态组合
	for from, transitions := range expectedMatrix {
		for to, shouldBeValid := range transitions {
			err := sm.ValidateTransition(from, to)

			if shouldBeValid {
				assert.NoError(t, err, "Transition %s -> %s should be valid", from, to)
			} else {
				assert.Error(t, err, "Transition %s -> %s should be invalid", from, to)
			}
		}
	}
}

func TestStateMachine_TaskStatusConstants(t *testing.T) {
	// 验证状态常量的值
	assert.Equal(t, TaskStatus("pending"), TaskStatusPending)
	assert.Equal(t, TaskStatus("running"), TaskStatusRunning)
	assert.Equal(t, TaskStatus("completed"), TaskStatusCompleted)
	assert.Equal(t, TaskStatus("failed"), TaskStatusFailed)
	assert.Equal(t, TaskStatus("cancelled"), TaskStatusCancelled)
}

func TestStateMachine_TaskTypeConstants(t *testing.T) {
	// 验证任务类型常量的值
	assert.Equal(t, TaskType("input"), TaskTypeInput)
	assert.Equal(t, TaskType("output"), TaskTypeOutput)
}

// 测试状态转换的边界情况
func TestStateMachine_ValidateTransition_EdgeCases(t *testing.T) {
	sm := &StateMachine{}

	testCases := []struct {
		name        string
		from        TaskStatus
		to          TaskStatus
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Same state transition (pending -> pending)",
			from:        TaskStatusPending,
			to:          TaskStatusPending,
			shouldError: true,
			errorMsg:    "invalid state transition",
		},
		{
			name:        "Same state transition (running -> running)",
			from:        TaskStatusRunning,
			to:          TaskStatusRunning,
			shouldError: true,
			errorMsg:    "invalid state transition",
		},
		{
			name:        "Empty source state",
			from:        TaskStatus(""),
			to:          TaskStatusRunning,
			shouldError: true,
			errorMsg:    "unknown source state",
		},
		{
			name:        "Empty target state",
			from:        TaskStatusPending,
			to:          TaskStatus(""),
			shouldError: true,
			errorMsg:    "invalid state transition",
		},
		{
			name:        "Valid retry transition",
			from:        TaskStatusFailed,
			to:          TaskStatusPending,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := sm.ValidateTransition(tc.from, tc.to)

			if tc.shouldError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
