package discoverypool

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/discoverypool"
	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
    "github.com/zeromicro/go-zero/core/logx"
)

type GetDiscoveryPoolListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDiscoveryPoolListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDiscoveryPoolListLogic {
	return &GetDiscoveryPoolListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDiscoveryPoolListLogic) GetDiscoveryPoolList(in *io.DiscoveryPoolListReq) (*io.DiscoveryPoolListResp, error) {
	var predicates []predicate.DiscoveryPool
	if in.CreatedAt != nil {
		predicates = append(predicates, discoverypool.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, discoverypool.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.Status != nil {
		predicates = append(predicates, discoverypool.StatusEQ(uint8(*in.Status)))
	}
	if in.Name != nil {
		predicates = append(predicates, discoverypool.NameContains(*in.Name))
	}
	if in.Description != nil {
		predicates = append(predicates, discoverypool.DescriptionContains(*in.Description))
	}
	if in.DiscoveryType != nil {
		predicates = append(predicates, discoverypool.DiscoveryTypeContains(*in.DiscoveryType))
	}
	if in.PoolStatus != nil {
		predicates = append(predicates, discoverypool.PoolStatusContains(*in.PoolStatus))
	}
	if in.DiscoveryConfig != nil {
		predicates = append(predicates, discoverypool.DiscoveryConfigContains(*in.DiscoveryConfig))
	}
	if in.Schedule != nil {
		predicates = append(predicates, discoverypool.ScheduleContains(*in.Schedule))
	}
	if in.BatchSize != nil {
		predicates = append(predicates, discoverypool.BatchSizeEQ(int(*in.BatchSize)))
	}
	if in.ConcurrentLimit != nil {
		predicates = append(predicates, discoverypool.ConcurrentLimitEQ(int(*in.ConcurrentLimit)))
	}
	if in.MaxRetry != nil {
		predicates = append(predicates, discoverypool.MaxRetryEQ(int(*in.MaxRetry)))
	}
	if in.RetryInterval != nil {
		predicates = append(predicates, discoverypool.RetryIntervalEQ(int(*in.RetryInterval)))
	}
	if in.FieldMapping != nil {
		predicates = append(predicates, discoverypool.FieldMappingContains(*in.FieldMapping))
	}
	if in.TotalRuns != nil {
		predicates = append(predicates, discoverypool.TotalRunsEQ(*in.TotalRuns))
	}
	if in.SuccessRuns != nil {
		predicates = append(predicates, discoverypool.SuccessRunsEQ(*in.SuccessRuns))
	}
	if in.FailedRuns != nil {
		predicates = append(predicates, discoverypool.FailedRunsEQ(*in.FailedRuns))
	}
	if in.LastRunAt != nil {
		predicates = append(predicates, discoverypool.LastRunAtGTE(time.UnixMilli(*in.LastRunAt)))
	}
	if in.LastSuccessAt != nil {
		predicates = append(predicates, discoverypool.LastSuccessAtGTE(time.UnixMilli(*in.LastSuccessAt)))
	}
	if in.LastError != nil {
		predicates = append(predicates, discoverypool.LastErrorContains(*in.LastError))
	}
	if in.ApprovalStatus != nil {
		predicates = append(predicates, discoverypool.ApprovalStatusContains(*in.ApprovalStatus))
	}
	if in.ApprovedBy != nil {
		predicates = append(predicates, discoverypool.ApprovedByEQ(*in.ApprovedBy))
	}
	if in.ApprovedAt != nil {
		predicates = append(predicates, discoverypool.ApprovedAtGTE(time.UnixMilli(*in.ApprovedAt)))
	}
	if in.RejectionReason != nil {
		predicates = append(predicates, discoverypool.RejectionReasonContains(*in.RejectionReason))
	}
	if in.Metadata != nil {
		predicates = append(predicates, discoverypool.MetadataContains(*in.Metadata))
	}
	result, err := l.svcCtx.DB.DiscoveryPool.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.DiscoveryPoolListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &io.DiscoveryPoolInfo{
			Id:          &v.ID,
			CreatedAt:   utils.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:   utils.GetPointer(v.UpdatedAt.UnixMilli()),
			Status:	utils.GetPointer(uint32(v.Status)),
			Name:	&v.Name,
			Description:	&v.Description,
			DiscoveryType:	&v.DiscoveryType,
			PoolStatus:	&v.PoolStatus,
			DiscoveryConfig:	&v.DiscoveryConfig,
			Schedule:	&v.Schedule,
			BatchSize:	utils.GetPointer(int64(v.BatchSize)),
			ConcurrentLimit:	utils.GetPointer(int64(v.ConcurrentLimit)),
			MaxRetry:	utils.GetPointer(int64(v.MaxRetry)),
			RetryInterval:	utils.GetPointer(int64(v.RetryInterval)),
			FieldMapping:	&v.FieldMapping,
			TotalRuns:	&v.TotalRuns,
			SuccessRuns:	&v.SuccessRuns,
			FailedRuns:	&v.FailedRuns,
			LastRunAt:	utils.GetUnixMilliPointer(v.LastRunAt.UnixMilli()),
			LastSuccessAt:	utils.GetUnixMilliPointer(v.LastSuccessAt.UnixMilli()),
			LastError:	&v.LastError,
			ApprovalStatus:	&v.ApprovalStatus,
			ApprovedBy:	v.ApprovedBy,
			ApprovedAt:	utils.GetUnixMilliPointer(v.ApprovedAt.UnixMilli()),
			RejectionReason:	&v.RejectionReason,
			Metadata:	&v.Metadata,
		})
	}

	return resp, nil
}
