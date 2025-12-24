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

type UpdateDiscoveryTemplateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateDiscoveryTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDiscoveryTemplateLogic {
	return &UpdateDiscoveryTemplateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateDiscoveryTemplateLogic) UpdateDiscoveryTemplate(in *io.DiscoveryTemplateInfo) (*io.BaseResp, error) {
	query:= l.svcCtx.DB.DiscoveryTemplate.UpdateOneID(*in.Id).
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

	 err := query.Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.UpdateSuccess }, nil
}
