package discoverypool

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDiscoveryPoolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateDiscoveryPoolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDiscoveryPoolLogic {
	return &CreateDiscoveryPoolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateDiscoveryPoolLogic) CreateDiscoveryPool(in *io.DiscoveryPoolInfo) (*io.BaseIDResp, error) {
	// TODO: Phase 2.4 - Agent Verification (待proto添加agent_id字段后实现)
	// 一旦DiscoveryPoolInfo添加了agent_id字段，在此处添加Agent验证逻辑：
	// if in.AgentId != nil && *in.AgentId != "" {
	// 	// 验证Agent存在且在线
	// 	agentResp, err := l.svcCtx.OpsRpc.GetAgentList(l.ctx, &ops.AgentListReq{
	// 		Page:     1,
	// 		PageSize: 1,
	// 		// 需要在AgentListReq中添加agent_id过滤字段
	// 	})
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to verify agent: %w", err)
	// 	}
	// 	if agentResp.Total == 0 {
	// 		return nil, fmt.Errorf("agent not found: %s", *in.AgentId)
	// 	}
	// 	agent := agentResp.Data[0]
	// 	if agent.AgentStatus != nil && *agent.AgentStatus != "online" {
	// 		return nil, fmt.Errorf("agent is not online: %s (status: %s)",
	// 			*in.AgentId, *agent.AgentStatus)
	// 	}
	// }

    query := l.svcCtx.DB.DiscoveryPool.Create().
			SetNotNilName(in.Name).
			SetNotNilDescription(in.Description).
			SetNotNilDiscoveryType(in.DiscoveryType).
			SetNotNilPoolStatus(in.PoolStatus).
			SetNotNilDiscoveryConfig(in.DiscoveryConfig).
			SetNotNilSchedule(in.Schedule).
			SetNotNilFieldMapping(in.FieldMapping).
			SetNotNilTotalRuns(in.TotalRuns).
			SetNotNilSuccessRuns(in.SuccessRuns).
			SetNotNilFailedRuns(in.FailedRuns).
			SetNotNilLastRunAt(utils.GetTimeMilliPointer(in.LastRunAt)).
			SetNotNilLastSuccessAt(utils.GetTimeMilliPointer(in.LastSuccessAt)).
			SetNotNilLastError(in.LastError).
			SetNotNilApprovalStatus(in.ApprovalStatus).
			SetNotNilApprovedBy(in.ApprovedBy).
			SetNotNilApprovedAt(utils.GetTimeMilliPointer(in.ApprovedAt)).
			SetNotNilRejectionReason(in.RejectionReason).
			SetNotNilMetadata(in.Metadata)

	if in.Status != nil {
		query.SetNotNilStatus(utils.GetPointer(uint8(*in.Status)))
	}
	if in.BatchSize != nil {
		query.SetNotNilBatchSize(utils.GetPointer(int(*in.BatchSize)))
	}
	if in.ConcurrentLimit != nil {
		query.SetNotNilConcurrentLimit(utils.GetPointer(int(*in.ConcurrentLimit)))
	}
	if in.MaxRetry != nil {
		query.SetNotNilMaxRetry(utils.GetPointer(int(*in.MaxRetry)))
	}
	if in.RetryInterval != nil {
		query.SetNotNilRetryInterval(utils.GetPointer(int(*in.RetryInterval)))
	}

	result, err := query.Save(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseIDResp{Id: result.ID, Msg: errormsg.CreateSuccess }, nil
}
