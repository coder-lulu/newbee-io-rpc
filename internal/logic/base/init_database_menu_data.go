package base

import (
	"fmt"
	"github.com/coder-lulu/newbee-common/v2/enum/common"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-common/v2/utils/pointy"
	"github.com/coder-lulu/newbee-core/rpc/types/core"
)

// Initial installation registers catalogs in the default tenant only. Existing
// menus and role grants are preserved; no ordinary role receives new access.
func (l *InitDatabaseLogic) insertCoreData() error {
	if l.svcCtx.CoreRpc == nil {
		return fmt.Errorf("Core RPC is required")
	}
	ctx := hooks.SetTenantIDToContext(l.ctx, 1)
	client := l.svcCtx.CoreRpc
	roles, err := client.GetRoleList(ctx, &core.RoleListReq{Page: 1, PageSize: 2, Code: pointy.GetPointer("superadmin")})
	if err != nil {
		return fmt.Errorf("read superadmin role: %w", err)
	}
	if len(roles.GetData()) != 1 || roles.Data[0].GetId() == 0 {
		return fmt.Errorf("default tenant superadmin role is missing or ambiguous")
	}
	roleID := roles.Data[0].GetId()
	var existing []*core.MenuInfo
	for page := uint64(1); ; page++ {
		resp, err := client.GetMenuList(ctx, &core.PageInfoReq{Page: page, PageSize: 100})
		if err != nil {
			return fmt.Errorf("read menus: %w", err)
		}
		existing = append(existing, resp.GetData()...)
		if len(resp.GetData()) < 100 {
			break
		}
	}
	ensureMenu := func(path, title, component string, parent uint64, sort uint32, menuType uint32) (uint64, error) {
		for _, menu := range existing {
			if menu.GetParentId() == parent && (menu.GetPath() == path || (component != "LAYOUT" && menu.GetComponent() == component)) {
				return menu.GetId(), nil
			}
		}
		resp, err := client.CreateMenu(ctx, &core.MenuInfo{
			Path: pointy.GetPointer(path), Name: pointy.GetPointer("IO · " + title),
			Component: pointy.GetPointer(component), ParentId: &parent,
			Sort: &sort, MenuType: &menuType, ServiceName: pointy.GetPointer("unified-io"),
			Meta: &core.Meta{Title: pointy.GetPointer(title), Icon: pointy.GetPointer("lucide:server")},
		})
		if err != nil {
			return 0, fmt.Errorf("create menu %s: %w", path, err)
		}
		if resp.GetId() == 0 {
			return 0, fmt.Errorf("create menu %s returned no ID", path)
		}
		return resp.GetId(), nil
	}
	rootID, err := ensureMenu("/io", "输入输出", "LAYOUT", common.DefaultParentId, 5, 0)
	if err != nil {
		return err
	}
	menuIDs := []uint64{rootID}
	for index, item := range []struct{ path, title, component string }{
		{"input-task", "输入任务", "io/input-task/list"},
		{"output-task", "输出任务", "io/output-task/list"},
		{"data-target", "数据目标", "io/data-target/index"},
		{"field-mapping", "字段映射", "io/field-mapping/list"},
		{"discovery-provider", "发现服务", "io/discovery-provider/index"},
		{"discovery-template", "发现模板", "io/discovery-template/index"},
		{"discovery-pool", "发现池", "io/discovery-pool/list"},
		{"change-history", "CI变更历史", "io/change-history/list"},
		{"lifecycle-state", "CI生命周期", "io/lifecycle-state/list"},
		{"config", "配置中心", "io/config/list"},
		{"monitor", "IO监控", "io/monitor/index"},
		{"logs", "任务与映射日志", "io/logs/index"},
	} {
		id, err := ensureMenu("/io/"+item.path, item.title, item.component, rootID, uint32(index+1), 1)
		if err != nil {
			return err
		}
		menuIDs = append(menuIDs, id)
	}
	grants, err := client.GetMenuAuthority(ctx, &core.IDReq{Id: roleID})
	if err != nil {
		return fmt.Errorf("read superadmin menus: %w", err)
	}
	merged := append([]uint64(nil), grants.GetMenuIds()...)
	known := make(map[uint64]bool, len(merged))
	for _, id := range merged {
		known[id] = true
	}
	for _, id := range menuIDs {
		if !known[id] {
			merged = append(merged, id)
			known[id] = true
		}
	}
	if len(merged) != len(grants.GetMenuIds()) {
		if _, err := client.CreateOrUpdateMenuAuthority(ctx, &core.RoleMenuAuthorityReq{RoleId: roleID, MenuIds: merged}); err != nil {
			return fmt.Errorf("grant superadmin menus: %w", err)
		}
	}
	// Core CreateApi is idempotent by (tenant, path, method). The catalog
	// supplies no blanket Casbin grants; superadmin uses Core's existing policy.
	for _, api := range apiEndpoints {
		if _, err := client.CreateApi(ctx, &core.ApiInfo{
			Path: &api.path, Method: &api.method,
			ServiceName: pointy.GetPointer("unified-io"), ApiGroup: pointy.GetPointer("unified-io"),
			Description: pointy.GetPointer(api.method + " " + api.path),
		}); err != nil {
			return fmt.Errorf("register API %s %s: %w", api.method, api.path, err)
		}
	}
	return nil
}
