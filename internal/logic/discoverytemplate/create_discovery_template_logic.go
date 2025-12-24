package discoverytemplate

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDiscoveryTemplateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateDiscoveryTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDiscoveryTemplateLogic {
	return &CreateDiscoveryTemplateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateDiscoveryTemplateLogic) CreateDiscoveryTemplate(in *io.DiscoveryTemplateInfo) (*io.BaseIDResp, error) {
    query := l.svcCtx.DB.DiscoveryTemplate.Create().
			SetNotNilTemplateName(in.TemplateName).
			SetNotNilTemplateCode(in.TemplateCode).
			SetNotNilDescription(in.Description).
			SetNotNilVersion(in.Version).
			SetNotNilTemplateType(in.TemplateType).
			SetNotNilDiscoveryConfig(in.DiscoveryConfig).
			SetNotNilFieldMappingTemplates(in.FieldMappingTemplates).
			SetNotNilValidationRules(in.ValidationRules).
			SetNotNilIsPublic(in.IsPublic).
			SetNotNilIsSystem(in.IsSystem).
			SetNotNilUsageCount(in.UsageCount).
			SetNotNilTags(in.Tags).
			SetNotNilMetadata(in.Metadata)

	if in.Status != nil {
		query.SetNotNilStatus(utils.GetPointer(uint8(*in.Status)))
	}

	result, err := query.Save(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseIDResp{Id: result.ID, Msg: errormsg.CreateSuccess }, nil
}
