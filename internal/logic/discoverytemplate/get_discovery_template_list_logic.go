package discoverytemplate

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/discoverytemplate"
	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
    "github.com/zeromicro/go-zero/core/logx"
)

type GetDiscoveryTemplateListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDiscoveryTemplateListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDiscoveryTemplateListLogic {
	return &GetDiscoveryTemplateListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDiscoveryTemplateListLogic) GetDiscoveryTemplateList(in *io.DiscoveryTemplateListReq) (*io.DiscoveryTemplateListResp, error) {
	var predicates []predicate.DiscoveryTemplate
	if in.CreatedAt != nil {
		predicates = append(predicates, discoverytemplate.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, discoverytemplate.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.Status != nil {
		predicates = append(predicates, discoverytemplate.StatusEQ(uint8(*in.Status)))
	}
	if in.TemplateName != nil {
		predicates = append(predicates, discoverytemplate.TemplateNameContains(*in.TemplateName))
	}
	if in.TemplateCode != nil {
		predicates = append(predicates, discoverytemplate.TemplateCodeContains(*in.TemplateCode))
	}
	if in.Description != nil {
		predicates = append(predicates, discoverytemplate.DescriptionContains(*in.Description))
	}
	if in.Version != nil {
		predicates = append(predicates, discoverytemplate.VersionContains(*in.Version))
	}
	if in.TemplateType != nil {
		predicates = append(predicates, discoverytemplate.TemplateTypeContains(*in.TemplateType))
	}
	if in.DiscoveryConfig != nil {
		predicates = append(predicates, discoverytemplate.DiscoveryConfigContains(*in.DiscoveryConfig))
	}
	if in.FieldMappingTemplates != nil {
		predicates = append(predicates, discoverytemplate.FieldMappingTemplatesContains(*in.FieldMappingTemplates))
	}
	if in.ValidationRules != nil {
		predicates = append(predicates, discoverytemplate.ValidationRulesContains(*in.ValidationRules))
	}
	if in.IsPublic != nil {
		predicates = append(predicates, discoverytemplate.IsPublicEQ(*in.IsPublic))
	}
	if in.IsSystem != nil {
		predicates = append(predicates, discoverytemplate.IsSystemEQ(*in.IsSystem))
	}
	if in.UsageCount != nil {
		predicates = append(predicates, discoverytemplate.UsageCountEQ(*in.UsageCount))
	}
	if in.Tags != nil {
		predicates = append(predicates, discoverytemplate.TagsContains(*in.Tags))
	}
	if in.Metadata != nil {
		predicates = append(predicates, discoverytemplate.MetadataContains(*in.Metadata))
	}
	result, err := l.svcCtx.DB.DiscoveryTemplate.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.DiscoveryTemplateListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &io.DiscoveryTemplateInfo{
			Id:          &v.ID,
			CreatedAt:   utils.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:   utils.GetPointer(v.UpdatedAt.UnixMilli()),
			Status:	utils.GetPointer(uint32(v.Status)),
			TemplateName:	&v.TemplateName,
			TemplateCode:	&v.TemplateCode,
			Description:	&v.Description,
			Version:	&v.Version,
			TemplateType:	&v.TemplateType,
			DiscoveryConfig:	&v.DiscoveryConfig,
			FieldMappingTemplates:	&v.FieldMappingTemplates,
			ValidationRules:	&v.ValidationRules,
			IsPublic:	&v.IsPublic,
			IsSystem:	&v.IsSystem,
			UsageCount:	&v.UsageCount,
			Tags:	&v.Tags,
			Metadata:	&v.Metadata,
		})
	}

	return resp, nil
}
