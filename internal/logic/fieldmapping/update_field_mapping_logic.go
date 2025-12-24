package fieldmapping

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateFieldMappingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateFieldMappingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateFieldMappingLogic {
	return &UpdateFieldMappingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateFieldMappingLogic) UpdateFieldMapping(in *io.FieldMappingInfo) (*io.BaseResp, error) {
	query:= l.svcCtx.DB.FieldMapping.UpdateOneID(*in.Id).
			SetNotNilMappingName(in.MappingName).
			SetNotNilDescription(in.Description).
			SetNotNilMappingType(in.MappingType).
			SetNotNilIsActive(in.IsActive).
			SetNotNilSourceField(in.SourceField).
			SetNotNilSourceFieldPath(in.SourceFieldPath).
			SetNotNilSourceDataType(in.SourceDataType).
			SetNotNilSourceFormat(in.SourceFormat).
			SetNotNilTargetField(in.TargetField).
			SetNotNilTargetFieldPath(in.TargetFieldPath).
			SetNotNilTargetDataType(in.TargetDataType).
			SetNotNilTargetFormat(in.TargetFormat).
			SetNotNilTransformType(in.TransformType).
			SetNotNilTransformConfig(in.TransformConfig).
			SetNotNilDefaultValue(in.DefaultValue).
			SetNotNilAllowNull(in.AllowNull).
			SetNotNilIsRequired(in.IsRequired).
			SetNotNilValidationRules(in.ValidationRules).
			SetNotNilValidationRegex(in.ValidationRegex).
			SetNotNilLookupTable(in.LookupTable).
			SetNotNilLookupCaseSensitive(in.LookupCaseSensitive).
			SetNotNilConditionRules(in.ConditionRules).
			SetNotNilDiscoveryPoolID(in.DiscoveryPoolId).
			SetNotNilInputTaskID(in.InputTaskId).
			SetNotNilOutputTaskID(in.OutputTaskId).
			SetNotNilUsageCount(in.UsageCount).
			SetNotNilSuccessCount(in.SuccessCount).
			SetNotNilFailedCount(in.FailedCount).
			SetNotNilLastUsedAt(utils.GetTimeMilliPointer(in.LastUsedAt)).
			SetNotNilLastError(in.LastError).
			SetNotNilLastErrorAt(utils.GetTimeMilliPointer(in.LastErrorAt)).
			SetNotNilMetadata(in.Metadata)

	if in.Status != nil {
		query.SetNotNilStatus(utils.GetPointer(uint8(*in.Status)))
	}
	if in.Priority != nil {
		query.SetNotNilPriority(utils.GetPointer(int(*in.Priority)))
	}
	if in.SortOrder != nil {
		query.SetNotNilSortOrder(utils.GetPointer(int(*in.SortOrder)))
	}

	 err := query.Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.UpdateSuccess }, nil
}
