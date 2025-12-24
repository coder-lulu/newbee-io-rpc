package base

import (
	"context"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/coder-lulu/newbee-common/v2/msg/errormsg"
	"github.com/coder-lulu/newbee-common/v2/msg/logmsg"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-common/v2/errors"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type InitDatabaseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewInitDatabaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InitDatabaseLogic {
	return &InitDatabaseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *InitDatabaseLogic) InitDatabase(in *io.Empty) (*io.BaseResp, error) {
	if err := l.svcCtx.DB.Schema.Create(l.ctx, schema.WithForeignKeys(false)); err != nil {
		logx.Errorw(logmsg.DatabaseError, logx.Field("detail", err.Error()))
		return nil, errors.InternalWithCause("Database initialization failed", err)
	}

	errHandler := func(err error) (*io.BaseResp, error) {
		logx.Errorw(logmsg.DatabaseError, logx.Field("detail", err.Error()))
		return nil, errors.InternalWithCause("Database initialization failed", err)
	}

	err := l.InsertInitData()
	if err != nil {
		return errHandler(err)
	}

	return &io.BaseResp{Msg: errormsg.Success}, nil
}

func (l *InitDatabaseLogic) InsertInitData() error {
	// 🔥 使用默认租户ID=1进行初始化，而不是系统级上下文(tenant_id=0)
	// 这与Core服务保持一致，为默认租户创建初始数据
	tenantID := uint64(1)
	ctxWithTenant := hooks.SetTenantIDToContext(context.Background(), tenantID)

	err := l.insertDiscoveryProviderData(ctxWithTenant, tenantID)
	if err != nil {
		return err
	}

	logx.Infow("Unified IO database initialized successfully",
		logx.Field("tenant_id", tenantID),
		logx.Field("provider_count", 15))
	return nil
}
