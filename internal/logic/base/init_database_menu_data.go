package base

import (
	"context"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-core/rpc/types/core"
	"github.com/zeromicro/go-zero/core/logx"
	"go.openly.dev/pointy"
)

// insertIOLifecycleMenuData 插入IO生命周期和变更历史菜单数据到Core服务菜单表
func (l *InitDatabaseLogic) insertIOLifecycleMenuData(ctx context.Context) error {
	if l.svcCtx.CoreRpc == nil {
		logx.Info("Core RPC client is not configured, skipping menu insertion")
		return nil
	}

	// 为Core RPC调用创建包含租户信息的上下文
	// 使用默认租户ID 1，这是系统初始化时使用的标准租户
	tenantCtx := hooks.SetTenantIDToContext(ctx, uint64(1))

	// 检查菜单是否已经存在，防止重复插入
	existingMenus := make(map[uint64]bool)
	menuListResp, err := l.svcCtx.CoreRpc.GetMenuList(tenantCtx, &core.PageInfoReq{
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		logx.Errorw("Failed to check existing IO lifecycle menus, but continuing with insertion", logx.Field("error", err.Error()))
		logx.Info("Attempting to insert IO lifecycle menus despite Core service connection issues")
	} else {
		ioMenuCount := 0
		if menuListResp.Data != nil {
			for _, menu := range menuListResp.Data {
				if menu.ServiceName != nil && *menu.ServiceName == "unified-io" {
					if menu.Id != nil {
						existingMenus[*menu.Id] = true
					}
					ioMenuCount++
				}
			}
		}
		logx.Infof("Found %d existing unified-io menus, will perform incremental insertion", ioMenuCount)
	}

	// 定义IO生命周期和变更历史菜单数据
	ioLifecycleMenus := []*core.MenuInfo{
		// ================================================
		// 1. CI变更历史管理 (ID: 350)
		// ================================================
		{
			Id:          pointy.Uint64(350),
			Level:       pointy.Uint32(2),
			MenuType:    pointy.Uint32(1), // 菜单类型
			ParentId:    pointy.Uint64(201), // 挂载在"输入输出"模块下
			Path:        pointy.String("change-history"),
			Name:        pointy.String("CI变更历史"),
			Component:   pointy.String("io/change-history/list"),
			Sort:        pointy.Uint32(11),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("CI变更历史"),
				Icon:     pointy.String("lucide:history"),
				HideMenu: pointy.Bool(false),
			},
		},
		// CI变更历史 - 隐藏子页面：详情页
		{
			Id:          pointy.Uint64(360),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(1),
			ParentId:    pointy.Uint64(350),
			Path:        pointy.String("change-history/:id"),
			Name:        pointy.String("变更详情"),
			Component:   pointy.String("io/change-history/detail"),
			Sort:        pointy.Uint32(1),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("变更详情"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		// CI变更历史 - 隐藏子页面：时间线
		{
			Id:          pointy.Uint64(361),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(1),
			ParentId:    pointy.Uint64(350),
			Path:        pointy.String("change-history/timeline/:ciId"),
			Name:        pointy.String("变更时间线"),
			Component:   pointy.String("io/change-history/timeline"),
			Sort:        pointy.Uint32(2),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("变更时间线"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		// ================================================
		// 2. CI生命周期状态管理 (ID: 370)
		// ================================================
		{
			Id:          pointy.Uint64(370),
			Level:       pointy.Uint32(2),
			MenuType:    pointy.Uint32(1), // 菜单类型
			ParentId:    pointy.Uint64(201), // 挂载在"输入输出"模块下
			Path:        pointy.String("lifecycle-state"),
			Name:        pointy.String("CI生命周期"),
			Component:   pointy.String("io/lifecycle-state/list"),
			Sort:        pointy.Uint32(12),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("CI生命周期"),
				Icon:     pointy.String("lucide:git-branch"),
				HideMenu: pointy.Bool(false),
			},
		},
		// CI生命周期 - 隐藏子页面：详情页
		{
			Id:          pointy.Uint64(380),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(1),
			ParentId:    pointy.Uint64(370),
			Path:        pointy.String("lifecycle-state/:id"),
			Name:        pointy.String("生命周期详情"),
			Component:   pointy.String("io/lifecycle-state/detail"),
			Sort:        pointy.Uint32(1),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("生命周期详情"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		// CI生命周期 - 隐藏子页面：时间线
		{
			Id:          pointy.Uint64(381),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(1),
			ParentId:    pointy.Uint64(370),
			Path:        pointy.String("lifecycle-state/timeline/:ciId"),
			Name:        pointy.String("生命周期时间线"),
			Component:   pointy.String("io/lifecycle-state/timeline"),
			Sort:        pointy.Uint32(2),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("生命周期时间线"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
	}

	// 定义权限按钮（MenuType = 2）
	buttonPermissions := []*core.MenuInfo{
		// ================================================
		// CI变更历史权限按钮 (父ID: 350)
		// ================================================
		{
			Id:          pointy.Uint64(351),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2), // 按钮类型
			ParentId:    pointy.Uint64(350),
			Path:        pointy.String("/io/change_history/list"),
			Name:        pointy.String("查询变更历史"),
			Permission:  pointy.String("io:change_history:list"),
			Sort:        pointy.Uint32(1),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("查询变更历史"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(352),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(350),
			Path:        pointy.String("/io/change_history/info"),
			Name:        pointy.String("查看变更详情"),
			Permission:  pointy.String("io:change_history:info"),
			Sort:        pointy.Uint32(2),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("查看变更详情"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(353),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(350),
			Path:        pointy.String("/io/change_history/compare"),
			Name:        pointy.String("对比变更数据"),
			Permission:  pointy.String("io:change_history:compare"),
			Sort:        pointy.Uint32(3),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("对比变更数据"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(354),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(350),
			Path:        pointy.String("/io/change_history/timeline"),
			Name:        pointy.String("查看变更时间线"),
			Permission:  pointy.String("io:change_history:timeline"),
			Sort:        pointy.Uint32(4),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("查看时间线"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(355),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(350),
			Path:        pointy.String("/io/change_history/rollback"),
			Name:        pointy.String("回滚变更"),
			Permission:  pointy.String("io:change_history:rollback"),
			Sort:        pointy.Uint32(5),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("回滚变更"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(356),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(350),
			Path:        pointy.String("/io/change_history/approve"),
			Name:        pointy.String("审批变更"),
			Permission:  pointy.String("io:change_history:approve"),
			Sort:        pointy.Uint32(6),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("审批变更"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(357),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(350),
			Path:        pointy.String("/io/change_history/export"),
			Name:        pointy.String("导出变更记录"),
			Permission:  pointy.String("io:change_history:export"),
			Sort:        pointy.Uint32(7),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("导出记录"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		// ================================================
		// CI生命周期状态权限按钮 (父ID: 370)
		// ================================================
		{
			Id:          pointy.Uint64(371),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(370),
			Path:        pointy.String("/io/lifecycle_state/list"),
			Name:        pointy.String("查询生命周期状态"),
			Permission:  pointy.String("io:lifecycle_state:list"),
			Sort:        pointy.Uint32(1),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("查询状态"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(372),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(370),
			Path:        pointy.String("/io/lifecycle_state/info"),
			Name:        pointy.String("查看状态详情"),
			Permission:  pointy.String("io:lifecycle_state:info"),
			Sort:        pointy.Uint32(2),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("查看详情"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(373),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(370),
			Path:        pointy.String("/io/lifecycle_state/timeline"),
			Name:        pointy.String("查看状态时间线"),
			Permission:  pointy.String("io:lifecycle_state:timeline"),
			Sort:        pointy.Uint32(3),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("查看时间线"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(374),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(370),
			Path:        pointy.String("/io/lifecycle_state/transition"),
			Name:        pointy.String("执行状态转换"),
			Permission:  pointy.String("io:lifecycle_state:transition"),
			Sort:        pointy.Uint32(4),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("状态转换"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(375),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(370),
			Path:        pointy.String("/io/lifecycle_state/retry"),
			Name:        pointy.String("重试失败状态"),
			Permission:  pointy.String("io:lifecycle_state:retry"),
			Sort:        pointy.Uint32(5),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("重试状态"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(376),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(370),
			Path:        pointy.String("/io/lifecycle_state/cancel"),
			Name:        pointy.String("取消执行中状态"),
			Permission:  pointy.String("io:lifecycle_state:cancel"),
			Sort:        pointy.Uint32(6),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("取消状态"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(377),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(370),
			Path:        pointy.String("/io/lifecycle_state/check_timeout"),
			Name:        pointy.String("检查超时状态"),
			Permission:  pointy.String("io:lifecycle_state:check_timeout"),
			Sort:        pointy.Uint32(7),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("检查超时"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
		{
			Id:          pointy.Uint64(378),
			Level:       pointy.Uint32(3),
			MenuType:    pointy.Uint32(2),
			ParentId:    pointy.Uint64(370),
			Path:        pointy.String("/io/lifecycle_state/export"),
			Name:        pointy.String("导出状态记录"),
			Permission:  pointy.String("io:lifecycle_state:export"),
			Sort:        pointy.Uint32(8),
			ServiceName: pointy.String("unified-io"),
			TenantId:    pointy.Uint64(1),
			Meta: &core.Meta{
				Title:    pointy.String("导出记录"),
				Icon:     pointy.String(""),
				HideMenu: pointy.Bool(true),
			},
		},
	}

	// 合并所有菜单
	allMenus := append(ioLifecycleMenus, buttonPermissions...)

	successCount := 0
	skippedCount := 0

	logx.Info("Starting IO lifecycle and change history menu insertion...")

	// 插入所有菜单
	for _, menu := range allMenus {
		// 检查菜单是否已存在
		if existingMenus[*menu.Id] {
			logx.Infof("Menu '%s' (ID: %d) already exists, skipping", *menu.Name, *menu.Id)
			skippedCount++
			continue
		}

		// 菜单不存在，执行插入
		createResp, err := l.svcCtx.CoreRpc.CreateMenu(tenantCtx, menu)
		if err != nil {
			logx.Errorw("Failed to create menu due to Core service issues",
				logx.Field("menu", menu.Name),
				logx.Field("expected_id", menu.Id),
				logx.Field("menu_type", menu.MenuType),
				logx.Field("error", err.Error()))
			continue
		}

		if createResp != nil && createResp.Id != 0 {
			if createResp.Id == *menu.Id {
				logx.Infof("Created menu '%s' with expected ID: %d", *menu.Name, createResp.Id)
			} else {
				logx.Infof("Created menu '%s' with auto-generated ID: %d (expected: %d)", *menu.Name, createResp.Id, *menu.Id)
			}
			successCount++
			existingMenus[createResp.Id] = true
		}
	}

	totalMenus := len(allMenus)
	logx.Infow("IO lifecycle and change history menu insertion completed",
		logx.Field("total_items", totalMenus),
		logx.Field("page_count", len(ioLifecycleMenus)),
		logx.Field("button_count", len(buttonPermissions)),
		logx.Field("created_count", successCount),
		logx.Field("skipped_existing", skippedCount),
		logx.Field("failed_count", totalMenus-successCount-skippedCount))

	return nil
}
