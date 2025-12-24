package discoverytemplate

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetDiscoveryTemplateByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDiscoveryTemplateByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDiscoveryTemplateByIdLogic {
	return &GetDiscoveryTemplateByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDiscoveryTemplateByIdLogic) GetDiscoveryTemplateById(in *io.IDReq) (*io.DiscoveryTemplateInfo, error) {
	result, err := l.svcCtx.DB.DiscoveryTemplate.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &io.DiscoveryTemplateInfo{
		Id:          &result.ID,
		CreatedAt:    utils.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:    utils.GetPointer(result.UpdatedAt.UnixMilli()),
		Status:	utils.GetPointer(uint32(result.Status)),
		TemplateName:	&result.TemplateName,
		TemplateCode:	&result.TemplateCode,
		Description:	&result.Description,
		Version:	&result.Version,
		TemplateType:	&result.TemplateType,
		DiscoveryConfig:	&result.DiscoveryConfig,
		FieldMappingTemplates:	&result.FieldMappingTemplates,
		ValidationRules:	&result.ValidationRules,
		IsPublic:	&result.IsPublic,
		IsSystem:	&result.IsSystem,
		UsageCount:	&result.UsageCount,
		Tags:	&result.Tags,
		Metadata:	&result.Metadata,
	}, nil
}

