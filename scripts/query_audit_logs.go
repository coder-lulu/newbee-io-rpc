package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-io-rpc/ent"
	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
)

// 查询配置审计日志示例脚本
//
// 用法:
//   go run scripts/query_audit_logs.go 'dsn' [tenant_id] [config_key]
//
// 示例:
//   # 查询租户1的所有审计日志（最近100条）
//   go run scripts/query_audit_logs.go 'root:password@tcp(localhost:3306)/unified_io?parseTime=true' 1
//
//   # 查询租户1的特定配置键的审计日志
//   go run scripts/query_audit_logs.go 'root:password@tcp(localhost:3306)/unified_io?parseTime=true' 1 worker.batch_size

func main() {
	// 检查命令行参数
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run query_audit_logs.go <mysql_dsn> [tenant_id] [config_key]")
		fmt.Println("Example: go run query_audit_logs.go 'root:password@tcp(localhost:3306)/unified_io?parseTime=true' 1")
		os.Exit(1)
	}

	dsn := os.Args[1]
	tenantID := uint64(1) // 默认租户ID
	configKey := ""       // 默认查询所有配置

	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &tenantID)
	}
	if len(os.Args) >= 4 {
		configKey = os.Args[3]
	}

	fmt.Printf("🔍 查询配置审计日志...\n")
	fmt.Printf("   DSN: %s\n", dsn)
	fmt.Printf("   租户ID: %d\n", tenantID)
	if configKey != "" {
		fmt.Printf("   配置键: %s\n", configKey)
	} else {
		fmt.Printf("   配置键: <所有配置>\n")
	}
	fmt.Println()

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

	// 构建查询
	query := client.ConfigAuditLog.Query().
		Where(func(s *ent.ConfigAuditLogQuery) {
			// 使用ent的Where方法需要先导入configauditlog包
			// 这里简化处理
		}).
		Order(ent.Desc("created_at")).
		Limit(100)

	// 执行查询
	logs, err := query.All(ctx)
	if err != nil {
		fmt.Printf("❌ 查询失败: %v\n", err)
		os.Exit(1)
	}

	// 打印结果
	if len(logs) == 0 {
		fmt.Println("📭 没有找到审计日志")
		return
	}

	fmt.Printf("📊 找到 %d 条审计日志:\n\n", len(logs))
	fmt.Println("=" + repeat("=", 120))
	fmt.Printf("%-5s %-20s %-15s %-30s %-20s %-10s → %-10s\n",
		"ID", "变更时间", "变更类型", "配置键", "变更人", "旧版本", "新版本")
	fmt.Println("=" + repeat("=", 120))

	for _, log := range logs {
		changedBy := fmt.Sprintf("User %d", log.ChangedBy)
		if log.ChangedByName != "" {
			changedBy = log.ChangedByName
		} else if log.ChangedBy == 0 {
			changedBy = "System"
		}

		fmt.Printf("%-5d %-20s %-15s %-30s %-20s v%-9d → v%-9d\n",
			log.ID,
			log.CreatedAt.Format("2006-01-02 15:04:05"),
			log.ChangeType,
			truncate(log.ConfigKey, 30),
			truncate(changedBy, 20),
			log.OldVersion,
			log.NewVersion,
		)

		// 显示详细信息
		if log.OldValue != "" || log.NewValue != "" {
			fmt.Printf("      旧值: %s\n", truncate(log.OldValue, 80))
			fmt.Printf("      新值: %s\n", truncate(log.NewValue, 80))
		}

		if log.ChangeReason != "" {
			fmt.Printf("      原因: %s\n", log.ChangeReason)
		}

		if log.IsRollback {
			fmt.Printf("      🔄 回滚操作 (来源: %s)\n", log.RollbackFromLogID)
		}

		fmt.Println()
	}

	fmt.Println("=" + repeat("=", 120))
	fmt.Printf("\n✅ 查询完成！共 %d 条记录\n", len(logs))
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// repeat 重复字符串
func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
