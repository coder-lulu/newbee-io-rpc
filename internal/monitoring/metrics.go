package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector Prometheus指标收集器
type MetricsCollector struct {
	// CI变更历史指标
	ChangeHistoryTotal       *prometheus.CounterVec   // 变更操作总数
	ChangeHistoryDuration    *prometheus.HistogramVec // 变更操作耗时
	ChangeHistoryErrors      *prometheus.CounterVec   // 变更错误数
	ChangeHistoryRollbacks   prometheus.Counter       // 回滚操作数
	ChangeHistoryApprovals   *prometheus.CounterVec   // 审批操作数(批准/拒绝)
	ChangeHistoryComparisons prometheus.Counter       // 变更对比操作数

	// CI生命周期指标
	LifecycleStateTotal      *prometheus.CounterVec   // 状态转换总数
	LifecycleStateDuration   *prometheus.HistogramVec // 状态持续时间
	LifecycleStateErrors     *prometheus.CounterVec   // 状态转换错误数
	LifecycleStateCurrent    *prometheus.GaugeVec     // 当前各状态的CI数量
	LifecycleStateTimeouts   *prometheus.CounterVec   // 超时状态数
	LifecycleStateRetries    *prometheus.CounterVec   // 重试操作数
	LifecycleStateCancels    *prometheus.CounterVec   // 取消操作数

	// 性能指标
	DatabaseQueryDuration *prometheus.HistogramVec // 数据库查询耗时
	CacheHitRatio         *prometheus.GaugeVec     // 缓存命中率
	ActiveOperations      *prometheus.GaugeVec     // 当前活跃操作数

	// 系统指标
	ComponentHealth *prometheus.GaugeVec // 组件健康状态
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		// ================================================
		// CI变更历史指标
		// ================================================
		ChangeHistoryTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "io_change_history_operations_total",
				Help: "CI变更操作总数 (operation_type: create/update/delete, status: success/failure)",
			},
			[]string{"tenant_id", "operation_type", "status"},
		),

		ChangeHistoryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "io_change_history_operation_duration_seconds",
				Help:    "CI变更操作耗时分布 (秒)",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
			},
			[]string{"tenant_id", "operation_type"},
		),

		ChangeHistoryErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "io_change_history_errors_total",
				Help: "CI变更错误总数 (error_type: validation/permission/database/rollback)",
			},
			[]string{"tenant_id", "operation_type", "error_type"},
		),

		ChangeHistoryRollbacks: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "io_change_history_rollbacks_total",
				Help: "CI变更回滚操作总数",
			},
		),

		ChangeHistoryApprovals: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "io_change_history_approvals_total",
				Help: "CI变更审批操作总数 (result: approved/rejected)",
			},
			[]string{"tenant_id", "result"},
		),

		ChangeHistoryComparisons: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "io_change_history_comparisons_total",
				Help: "CI变更对比操作总数",
			},
		),

		// ================================================
		// CI生命周期状态指标
		// ================================================
		LifecycleStateTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "io_lifecycle_state_transitions_total",
				Help: "CI生命周期状态转换总数 (from_state -> to_state)",
			},
			[]string{"tenant_id", "from_state", "to_state", "status"},
		),

		LifecycleStateDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "io_lifecycle_state_duration_seconds",
				Help:    "CI在各状态的停留时间分布 (秒)",
				Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600, 7200, 14400},
			},
			[]string{"tenant_id", "state_type"},
		),

		LifecycleStateErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "io_lifecycle_state_errors_total",
				Help: "CI生命周期状态转换错误总数 (error_type: invalid_transition/validation/execution)",
			},
			[]string{"tenant_id", "from_state", "to_state", "error_type"},
		),

		LifecycleStateCurrent: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "io_lifecycle_state_current_count",
				Help: "当前各状态的CI实例数量",
			},
			[]string{"tenant_id", "state_type"},
		),

		LifecycleStateTimeouts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "io_lifecycle_state_timeouts_total",
				Help: "CI生命周期状态超时总数",
			},
			[]string{"tenant_id", "state_type"},
		),

		LifecycleStateRetries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "io_lifecycle_state_retries_total",
				Help: "CI生命周期状态重试总数",
			},
			[]string{"tenant_id", "state_type"},
		),

		LifecycleStateCancels: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "io_lifecycle_state_cancels_total",
				Help: "CI生命周期状态取消总数",
			},
			[]string{"tenant_id", "state_type"},
		),

		// ================================================
		// 性能指标
		// ================================================
		DatabaseQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "io_database_query_duration_seconds",
				Help:    "数据库查询耗时分布 (秒)",
				Buckets: []float64{0.001, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1},
			},
			[]string{"tenant_id", "operation", "table"},
		),

		CacheHitRatio: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "io_cache_hit_ratio",
				Help: "缓存命中率 (0-1)",
			},
			[]string{"tenant_id", "cache_type"},
		),

		ActiveOperations: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "io_active_operations_count",
				Help: "当前活跃操作数量",
			},
			[]string{"tenant_id", "operation_type"},
		),

		// ================================================
		// 系统指标
		// ================================================
		ComponentHealth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "io_component_health_status",
				Help: "组件健康状态 (1=healthy, 0=unhealthy)",
			},
			[]string{"component"},
		),
	}
}

// RecordChangeOperation 记录变更操作
func (m *MetricsCollector) RecordChangeOperation(tenantID, operationType, status string, duration float64) {
	m.ChangeHistoryTotal.WithLabelValues(tenantID, operationType, status).Inc()
	m.ChangeHistoryDuration.WithLabelValues(tenantID, operationType).Observe(duration)
}

// RecordChangeError 记录变更错误
func (m *MetricsCollector) RecordChangeError(tenantID, operationType, errorType string) {
	m.ChangeHistoryErrors.WithLabelValues(tenantID, operationType, errorType).Inc()
}

// RecordRollback 记录回滚操作
func (m *MetricsCollector) RecordRollback() {
	m.ChangeHistoryRollbacks.Inc()
}

// RecordApproval 记录审批操作
func (m *MetricsCollector) RecordApproval(tenantID, result string) {
	m.ChangeHistoryApprovals.WithLabelValues(tenantID, result).Inc()
}

// RecordComparison 记录变更对比操作
func (m *MetricsCollector) RecordComparison() {
	m.ChangeHistoryComparisons.Inc()
}

// RecordStateTransition 记录状态转换
func (m *MetricsCollector) RecordStateTransition(tenantID, fromState, toState, status string) {
	m.LifecycleStateTotal.WithLabelValues(tenantID, fromState, toState, status).Inc()
}

// RecordStateDuration 记录状态持续时间
func (m *MetricsCollector) RecordStateDuration(tenantID, stateType string, duration float64) {
	m.LifecycleStateDuration.WithLabelValues(tenantID, stateType).Observe(duration)
}

// RecordStateError 记录状态转换错误
func (m *MetricsCollector) RecordStateError(tenantID, fromState, toState, errorType string) {
	m.LifecycleStateErrors.WithLabelValues(tenantID, fromState, toState, errorType).Inc()
}

// UpdateCurrentStateCounts 更新当前状态计数
func (m *MetricsCollector) UpdateCurrentStateCounts(tenantID, stateType string, count float64) {
	m.LifecycleStateCurrent.WithLabelValues(tenantID, stateType).Set(count)
}

// RecordStateTimeout 记录状态超时
func (m *MetricsCollector) RecordStateTimeout(tenantID, stateType string) {
	m.LifecycleStateTimeouts.WithLabelValues(tenantID, stateType).Inc()
}

// RecordStateRetry 记录状态重试
func (m *MetricsCollector) RecordStateRetry(tenantID, stateType string) {
	m.LifecycleStateRetries.WithLabelValues(tenantID, stateType).Inc()
}

// RecordStateCancel 记录状态取消
func (m *MetricsCollector) RecordStateCancel(tenantID, stateType string) {
	m.LifecycleStateCancels.WithLabelValues(tenantID, stateType).Inc()
}

// RecordDatabaseQuery 记录数据库查询
func (m *MetricsCollector) RecordDatabaseQuery(tenantID, operation, table string, duration float64) {
	m.DatabaseQueryDuration.WithLabelValues(tenantID, operation, table).Observe(duration)
}

// UpdateCacheHitRatio 更新缓存命中率
func (m *MetricsCollector) UpdateCacheHitRatio(tenantID, cacheType string, ratio float64) {
	m.CacheHitRatio.WithLabelValues(tenantID, cacheType).Set(ratio)
}

// SetActiveOperations 设置活跃操作数
func (m *MetricsCollector) SetActiveOperations(tenantID, operationType string, count float64) {
	m.ActiveOperations.WithLabelValues(tenantID, operationType).Set(count)
}

// SetComponentHealth 设置组件健康状态
func (m *MetricsCollector) SetComponentHealth(component string, healthy bool) {
	status := 0.0
	if healthy {
		status = 1.0
	}
	m.ComponentHealth.WithLabelValues(component).Set(status)
}
