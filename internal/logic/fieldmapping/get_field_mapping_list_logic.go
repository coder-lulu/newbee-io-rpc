package fieldmapping

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/fieldmapping"
	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
    "github.com/zeromicro/go-zero/core/logx"
)

type GetFieldMappingListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFieldMappingListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFieldMappingListLogic {
	return &GetFieldMappingListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetFieldMappingListLogic) GetFieldMappingList(in *io.FieldMappingListReq) (*io.FieldMappingListResp, error) {
	var predicates []predicate.FieldMapping
	if in.CreatedAt != nil {
		predicates = append(predicates, fieldmapping.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, fieldmapping.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.Status != nil {
		predicates = append(predicates, fieldmapping.StatusEQ(uint8(*in.Status)))
	}
	if in.MappingName != nil {
		predicates = append(predicates, fieldmapping.MappingNameContains(*in.MappingName))
	}
	if in.Description != nil {
		predicates = append(predicates, fieldmapping.DescriptionContains(*in.Description))
	}
	if in.MappingType != nil {
		predicates = append(predicates, fieldmapping.MappingTypeContains(*in.MappingType))
	}
	if in.IsActive != nil {
		predicates = append(predicates, fieldmapping.IsActiveEQ(*in.IsActive))
	}
	if in.SourceField != nil {
		predicates = append(predicates, fieldmapping.SourceFieldContains(*in.SourceField))
	}
	if in.SourceFieldPath != nil {
		predicates = append(predicates, fieldmapping.SourceFieldPathContains(*in.SourceFieldPath))
	}
	if in.SourceDataType != nil {
		predicates = append(predicates, fieldmapping.SourceDataTypeContains(*in.SourceDataType))
	}
	if in.SourceFormat != nil {
		predicates = append(predicates, fieldmapping.SourceFormatContains(*in.SourceFormat))
	}
	if in.TargetField != nil {
		predicates = append(predicates, fieldmapping.TargetFieldContains(*in.TargetField))
	}
	if in.TargetFieldPath != nil {
		predicates = append(predicates, fieldmapping.TargetFieldPathContains(*in.TargetFieldPath))
	}
	if in.TargetDataType != nil {
		predicates = append(predicates, fieldmapping.TargetDataTypeContains(*in.TargetDataType))
	}
	if in.TargetFormat != nil {
		predicates = append(predicates, fieldmapping.TargetFormatContains(*in.TargetFormat))
	}
	if in.TransformType != nil {
		predicates = append(predicates, fieldmapping.TransformTypeContains(*in.TransformType))
	}
	if in.TransformConfig != nil {
		predicates = append(predicates, fieldmapping.TransformConfigContains(*in.TransformConfig))
	}
	if in.DefaultValue != nil {
		predicates = append(predicates, fieldmapping.DefaultValueContains(*in.DefaultValue))
	}
	if in.AllowNull != nil {
		predicates = append(predicates, fieldmapping.AllowNullEQ(*in.AllowNull))
	}
	if in.IsRequired != nil {
		predicates = append(predicates, fieldmapping.IsRequiredEQ(*in.IsRequired))
	}
	if in.ValidationRules != nil {
		predicates = append(predicates, fieldmapping.ValidationRulesContains(*in.ValidationRules))
	}
	if in.ValidationRegex != nil {
		predicates = append(predicates, fieldmapping.ValidationRegexContains(*in.ValidationRegex))
	}
	if in.LookupTable != nil {
		predicates = append(predicates, fieldmapping.LookupTableContains(*in.LookupTable))
	}
	if in.LookupCaseSensitive != nil {
		predicates = append(predicates, fieldmapping.LookupCaseSensitiveEQ(*in.LookupCaseSensitive))
	}
	if in.ConditionRules != nil {
		predicates = append(predicates, fieldmapping.ConditionRulesContains(*in.ConditionRules))
	}
	if in.Priority != nil {
		predicates = append(predicates, fieldmapping.PriorityEQ(int(*in.Priority)))
	}
	if in.SortOrder != nil {
		predicates = append(predicates, fieldmapping.SortOrderEQ(int(*in.SortOrder)))
	}
	if in.DiscoveryPoolId != nil {
		predicates = append(predicates, fieldmapping.DiscoveryPoolIDEQ(*in.DiscoveryPoolId))
	}
	if in.InputTaskId != nil {
		predicates = append(predicates, fieldmapping.InputTaskIDEQ(*in.InputTaskId))
	}
	if in.OutputTaskId != nil {
		predicates = append(predicates, fieldmapping.OutputTaskIDEQ(*in.OutputTaskId))
	}
	if in.UsageCount != nil {
		predicates = append(predicates, fieldmapping.UsageCountEQ(*in.UsageCount))
	}
	if in.SuccessCount != nil {
		predicates = append(predicates, fieldmapping.SuccessCountEQ(*in.SuccessCount))
	}
	if in.FailedCount != nil {
		predicates = append(predicates, fieldmapping.FailedCountEQ(*in.FailedCount))
	}
	if in.LastUsedAt != nil {
		predicates = append(predicates, fieldmapping.LastUsedAtGTE(time.UnixMilli(*in.LastUsedAt)))
	}
	if in.LastError != nil {
		predicates = append(predicates, fieldmapping.LastErrorContains(*in.LastError))
	}
	if in.LastErrorAt != nil {
		predicates = append(predicates, fieldmapping.LastErrorAtGTE(time.UnixMilli(*in.LastErrorAt)))
	}
	if in.Metadata != nil {
		predicates = append(predicates, fieldmapping.MetadataContains(*in.Metadata))
	}
	result, err := l.svcCtx.DB.FieldMapping.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.FieldMappingListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &io.FieldMappingInfo{
			Id:          &v.ID,
			CreatedAt:   utils.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:   utils.GetPointer(v.UpdatedAt.UnixMilli()),
			Status:	utils.GetPointer(uint32(v.Status)),
			MappingName:	&v.MappingName,
			Description:	&v.Description,
			MappingType:	&v.MappingType,
			IsActive:	&v.IsActive,
			SourceField:	&v.SourceField,
			SourceFieldPath:	&v.SourceFieldPath,
			SourceDataType:	&v.SourceDataType,
			SourceFormat:	&v.SourceFormat,
			TargetField:	&v.TargetField,
			TargetFieldPath:	&v.TargetFieldPath,
			TargetDataType:	&v.TargetDataType,
			TargetFormat:	&v.TargetFormat,
			TransformType:	&v.TransformType,
			TransformConfig:	&v.TransformConfig,
			DefaultValue:	&v.DefaultValue,
			AllowNull:	&v.AllowNull,
			IsRequired:	&v.IsRequired,
			ValidationRules:	&v.ValidationRules,
			ValidationRegex:	&v.ValidationRegex,
			LookupTable:	&v.LookupTable,
			LookupCaseSensitive:	&v.LookupCaseSensitive,
			ConditionRules:	&v.ConditionRules,
			Priority:	utils.GetPointer(int64(v.Priority)),
			SortOrder:	utils.GetPointer(int64(v.SortOrder)),
			DiscoveryPoolId:	v.DiscoveryPoolID,
			InputTaskId:	v.InputTaskID,
			OutputTaskId:	v.OutputTaskID,
			UsageCount:	&v.UsageCount,
			SuccessCount:	&v.SuccessCount,
			FailedCount:	&v.FailedCount,
			LastUsedAt:	utils.GetUnixMilliPointer(v.LastUsedAt.UnixMilli()),
			LastError:	&v.LastError,
			LastErrorAt:	utils.GetUnixMilliPointer(v.LastErrorAt.UnixMilli()),
			Metadata:	&v.Metadata,
		})
	}

	return resp, nil
}
