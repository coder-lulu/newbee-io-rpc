package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/configitem"
	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
)

// ConfigItem 配置项定义
type ConfigItem struct {
	Key          string
	Value        string
	ValueType    string
	Category     string
	ServiceName  string
	Description  string
	DefaultValue string
	IsReadonly   bool
	Scope        string
	ConfigGroup  string
}

// DefaultConfigs 默认配置列表
var DefaultConfigs = []ConfigItem{
	// ==================== TaskWorker配置 ====================
	{
		Key:          "worker.pull_interval",
		Value:        "10s",
		ValueType:    "string",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "TaskWorker拉取任务的时间间隔",
		DefaultValue: "10s",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "worker",
	},
	{
		Key:          "worker.batch_size",
		Value:        "10",
		ValueType:    "int",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "TaskWorker每次拉取的任务数量",
		DefaultValue: "10",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "worker",
	},
	{
		Key:          "worker.max_concurrent",
		Value:        "5",
		ValueType:    "int",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "TaskWorker最大并发处理任务数",
		DefaultValue: "5",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "worker",
	},
	{
		Key:          "worker.task_timeout",
		Value:        "5m",
		ValueType:    "string",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "单个任务执行超时时间",
		DefaultValue: "5m",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "worker",
	},
	{
		Key:          "worker.stale_threshold",
		Value:        "1h",
		ValueType:    "string",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "任务过期阈值（超过此时间的pending任务视为stale）",
		DefaultValue: "1h",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "worker",
	},
	{
		Key:          "worker.stale_check_interval",
		Value:        "10m",
		ValueType:    "string",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "Stale任务检查间隔",
		DefaultValue: "10m",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "worker",
	},

	// ==================== CronScheduler配置 ====================
	{
		Key:          "cron.default_timeout",
		Value:        "30m",
		ValueType:    "string",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "Cron任务默认超时时间",
		DefaultValue: "30m",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "cron",
	},
	{
		Key:          "cron.max_retry",
		Value:        "3",
		ValueType:    "int",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "Cron任务失败后最大重试次数",
		DefaultValue: "3",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "cron",
	},

	// ==================== 分布式锁配置 ====================
	{
		Key:          "lock.default_expiry",
		Value:        "30s",
		ValueType:    "string",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "分布式锁默认过期时间",
		DefaultValue: "30s",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "lock",
	},
	{
		Key:          "lock.retry_times",
		Value:        "3",
		ValueType:    "int",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "获取锁失败后重试次数",
		DefaultValue: "3",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "lock",
	},
	{
		Key:          "lock.retry_delay",
		Value:        "500ms",
		ValueType:    "string",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "获取锁失败后重试延迟",
		DefaultValue: "500ms",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "lock",
	},

	// ==================== 系统级配置 ====================
	{
		Key:          "system.maintenance_mode",
		Value:        "false",
		ValueType:    "bool",
		Category:     "system",
		ServiceName:  "unified-io",
		Description:  "系统维护模式（启用后拒绝新任务）",
		DefaultValue: "false",
		IsReadonly:   false,
		Scope:        "global",
		ConfigGroup:  "system",
	},
	{
		Key:          "system.log_level",
		Value:        "info",
		ValueType:    "string",
		Category:     "system",
		ServiceName:  "unified-io",
		Description:  "日志级别（debug/info/warn/error）",
		DefaultValue: "info",
		IsReadonly:   false,
		Scope:        "global",
		ConfigGroup:  "system",
	},

	// ==================== Provider配置 ====================
	{
		Key:          "provider.mysql.timeout",
		Value:        "30s",
		ValueType:    "string",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "MySQL Provider执行超时时间",
		DefaultValue: "30s",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "provider",
	},
	{
		Key:          "provider.api.timeout",
		Value:        "60s",
		ValueType:    "string",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "API Provider HTTP请求超时时间",
		DefaultValue: "60s",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "provider",
	},
	{
		Key:          "provider.api.max_retries",
		Value:        "3",
		ValueType:    "int",
		Category:     "service",
		ServiceName:  "unified-io",
		Description:  "API Provider请求失败最大重试次数",
		DefaultValue: "3",
		IsReadonly:   false,
		Scope:        "service",
		ConfigGroup:  "provider",
	},
}

func main() {
	// 检查命令行参数
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run init_configs.go <mysql_dsn> [tenant_id]")
		fmt.Println("Example: go run init_configs.go 'root:password@tcp(localhost:3306)/unified_io?parseTime=true' 1")
		os.Exit(1)
	}

	dsn := os.Args[1]
	tenantID := uint64(1) // 默认租户ID
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &tenantID)
	}

	fmt.Printf("🚀 开始初始化配置中心...\n")
	fmt.Printf("   DSN: %s\n", dsn)
	fmt.Printf("   租户ID: %d\n\n", tenantID)

	// 创建数据库客户端
	client, err := ent.Open("mysql", dsn)
	if err != nil {
		logx.Errorf("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer client.Close()

	// 注册租户Hook（使用SystemContext绕过）
	client.Use(hooks.TenantMutationHook())
	client.Intercept(hooks.TenantQueryInterceptor())

	ctx := hooks.NewSystemContext(context.Background())

	// 初始化配置
	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, cfg := range DefaultConfigs {
		// 检查配置是否已存在
		exists, err := client.ConfigItem.Query().
			Where(
				configitem.TenantIDEQ(tenantID),
				configitem.ConfigKeyEQ(cfg.Key),
			).
			Exist(ctx)

		if err != nil {
			fmt.Printf("❌ 检查配置失败: %s - %v\n", cfg.Key, err)
			errorCount++
			continue
		}

		if exists {
			fmt.Printf("⏭️  配置已存在，跳过: %s\n", cfg.Key)
			skipCount++
			continue
		}

		// 创建配置
		_, err = client.ConfigItem.Create().
			SetTenantID(tenantID).
			SetConfigKey(cfg.Key).
			SetConfigValue(cfg.Value).
			SetValueType(cfg.ValueType).
			SetCategory(cfg.Category).
			SetServiceName(cfg.ServiceName).
			SetDescription(cfg.Description).
			SetDefaultValue(cfg.DefaultValue).
			SetIsReadonly(cfg.IsReadonly).
			SetScope(cfg.Scope).
			SetConfigGroup(cfg.ConfigGroup).
			SetVersion(1).
			SetStatus(1).
			Save(ctx)

		if err != nil {
			fmt.Printf("❌ 创建配置失败: %s - %v\n", cfg.Key, err)
			errorCount++
			continue
		}

		fmt.Printf("✅ 创建配置成功: %s = %s\n", cfg.Key, cfg.Value)
		successCount++
		time.Sleep(10 * time.Millisecond) // 避免过快
	}

	// 打印统计结果
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("📊 配置初始化完成！\n")
	fmt.Printf("   ✅ 成功: %d\n", successCount)
	fmt.Printf("   ⏭️  跳过: %d\n", skipCount)
	fmt.Printf("   ❌ 失败: %d\n", errorCount)
	fmt.Printf("   📦 总计: %d\n", len(DefaultConfigs))
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	if errorCount > 0 {
		os.Exit(1)
	}
}
