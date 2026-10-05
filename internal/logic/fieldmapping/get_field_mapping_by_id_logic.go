package fieldmapping

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetFieldMappingByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFieldMappingByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFieldMappingByIdLogic {
	return &GetFieldMappingByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetFieldMappingByIdLogic) GetFieldMappingById(in *io.IDReq) (*io.FieldMappingInfo, error) {
	result, err := l.svcCtx.DB.FieldMapping.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &io.FieldMappingInfo{
		Id:                  &result.ID,
		CreatedAt:           utils.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:           utils.GetPointer(result.UpdatedAt.UnixMilli()),
		Status:              utils.GetPointer(uint32(result.Status)),
		MappingName:         &result.MappingName,
		Description:         &result.Description,
		MappingType:         &result.MappingType,
		IsActive:            &result.IsActive,
		SourceField:         &result.SourceField,
		SourceFieldPath:     &result.SourceFieldPath,
		SourceDataType:      &result.SourceDataType,
		SourceFormat:        &result.SourceFormat,
		TargetField:         &result.TargetField,
		TargetFieldPath:     &result.TargetFieldPath,
		TargetDataType:      &result.TargetDataType,
		TargetFormat:        &result.TargetFormat,
		TransformType:       &result.TransformType,
		TransformConfig:     &result.TransformConfig,
		DefaultValue:        &result.DefaultValue,
		AllowNull:           &result.AllowNull,
		IsRequired:          &result.IsRequired,
		ValidationRules:     &result.ValidationRules,
		ValidationRegex:     &result.ValidationRegex,
		LookupTable:         &result.LookupTable,
		LookupCaseSensitive: &result.LookupCaseSensitive,
		ConditionRules:      &result.ConditionRules,
		Priority:            utils.GetPointer(int64(result.Priority)),
		SortOrder:           utils.GetPointer(int64(result.SortOrder)),
		DiscoveryPoolId:     result.DiscoveryPoolID,
		InputTaskId:         result.InputTaskID,
		OutputTaskId:        result.OutputTaskID,
		UsageCount:          &result.UsageCount,
		SuccessCount:        &result.SuccessCount,
		FailedCount:         &result.FailedCount,
		LastUsedAt:          utils.TimeToUnixMilli(result.LastUsedAt),
		LastError:           &result.LastError,
		LastErrorAt:         utils.TimeToUnixMilli(result.LastErrorAt),
		Metadata:            &result.Metadata,
	}, nil
}
