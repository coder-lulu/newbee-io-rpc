package base

import (
	"context"
	"fmt"
	"github.com/coder-lulu/newbee-common/v2/enum/common"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-common/v2/utils/pointy"
	"github.com/coder-lulu/newbee-core/rpc/coreclient"
	"github.com/coder-lulu/newbee-core/rpc/types/core"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"google.golang.org/grpc"
	"testing"
)

type bootstrapCore struct {
	coreclient.Core
	menus  []*core.MenuInfo
	grants []uint64
	apis   map[string]bool
	fail   bool
}

func (c *bootstrapCore) GetRoleList(ctx context.Context, in *core.RoleListReq, _ ...grpc.CallOption) (*core.RoleListResp, error) {
	if c.fail {
		return nil, fmt.Errorf("Core unavailable")
	}
	if hooks.GetCurrentTenantID(ctx) != 1 {
		return nil, fmt.Errorf("missing bootstrap tenant")
	}
	return &core.RoleListResp{Data: []*core.RoleInfo{{Id: pointy.GetPointer(uint64(9)), Code: pointy.GetPointer("superadmin")}}}, nil
}
func (c *bootstrapCore) GetMenuList(context.Context, *core.PageInfoReq, ...grpc.CallOption) (*core.MenuInfoList, error) {
	return &core.MenuInfoList{Data: c.menus}, nil
}
func (c *bootstrapCore) CreateMenu(_ context.Context, in *core.MenuInfo, _ ...grpc.CallOption) (*core.BaseIDResp, error) {
	if in.GetParentId() != common.DefaultParentId {
		found := false
		for _, menu := range c.menus {
			if menu.GetId() == in.GetParentId() {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("parent menu does not exist: %d", in.GetParentId())
		}
	}
	for _, menu := range c.menus {
		if menu.GetPath() == in.GetPath() {
			return nil, fmt.Errorf("duplicate menu path: %s", in.GetPath())
		}
		if menu.GetName() == in.GetName() && menu.GetMenuType() == in.GetMenuType() {
			return nil, fmt.Errorf("duplicate menu name: %s", in.GetName())
		}
	}
	id := uint64(1001 + len(c.menus))
	in.Id = &id
	c.menus = append(c.menus, in)
	return &core.BaseIDResp{Id: id}, nil
}
func (c *bootstrapCore) GetMenuAuthority(context.Context, *core.IDReq, ...grpc.CallOption) (*core.RoleMenuAuthorityResp, error) {
	return &core.RoleMenuAuthorityResp{MenuIds: c.grants}, nil
}
func (c *bootstrapCore) CreateOrUpdateMenuAuthority(_ context.Context, in *core.RoleMenuAuthorityReq, _ ...grpc.CallOption) (*core.BaseResp, error) {
	if in.RoleId != 9 {
		return nil, fmt.Errorf("unexpected role")
	}
	c.grants = in.MenuIds
	return &core.BaseResp{}, nil
}
func (c *bootstrapCore) CreateApi(_ context.Context, in *core.ApiInfo, _ ...grpc.CallOption) (*core.BaseIDResp, error) {
	c.apis[in.GetMethod()+" "+in.GetPath()] = true
	return &core.BaseIDResp{Id: 1}, nil
}

func TestCoreCatalogInitialization(t *testing.T) {
	client := &bootstrapCore{grants: []uint64{42}, apis: make(map[string]bool), menus: []*core.MenuInfo{{Id: pointy.GetPointer(uint64(77)), ParentId: pointy.GetPointer(common.DefaultParentId), Path: pointy.GetPointer("config"), Name: pointy.GetPointer("配置中心"), MenuType: pointy.GetPointer(uint32(1))},
		{Id: pointy.GetPointer(uint64(88)), ParentId: pointy.GetPointer(common.DefaultParentId), Path: pointy.GetPointer("/io")},
		{Id: pointy.GetPointer(uint64(89)), ParentId: pointy.GetPointer(uint64(88)), Path: pointy.GetPointer("custom-existing-path"), Component: pointy.GetPointer("io/input-task/list")},
	}}
	logic := NewInitDatabaseLogic(context.Background(), &svc.ServiceContext{CoreRpc: client})
	for i := 0; i < 2; i++ {
		if err := logic.insertCoreData(); err != nil {
			t.Fatal(err)
		}
	}
	if len(client.menus) != 14 || len(client.grants) != 13+1 || client.grants[0] != 42 || len(client.apis) != len(apiEndpoints) {
		t.Fatalf("unexpected catalog: menus=%d grants=%v apis=%d", len(client.menus), client.grants, len(client.apis))
	}
	client.fail = true
	if err := logic.insertCoreData(); err == nil {
		t.Fatal("Core failure must propagate")
	}
	logic = NewInitDatabaseLogic(context.Background(), &svc.ServiceContext{})
	if err := logic.insertCoreData(); err == nil {
		t.Fatal("missing Core configuration must fail")
	}
}
