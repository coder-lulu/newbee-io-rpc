package core

import (
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
)

// NewDataValidator 创建数据校验器
func NewDataValidator(svcCtx *svc.ServiceContext, config *Configuration) DataValidator {
	// TODO: 返回具体实现
	// return &dataValidatorImpl{
	// 	svcCtx: svcCtx,
	// 	config: config,
	// }
	return nil
}

// NewPermissionChecker 创建权限检查器
func NewPermissionChecker(svcCtx *svc.ServiceContext, config *Configuration) PermissionChecker {
	// TODO: 返回具体实现
	// return &permissionCheckerImpl{
	// 	svcCtx: svcCtx,
	// 	config: config,
	// }
	return nil
}

// NewChangeRecorder 创建变更记录器
func NewChangeRecorder(svcCtx *svc.ServiceContext, config *Configuration) ChangeRecorder {
	// TODO: 返回具体实现
	// return &changeRecorderImpl{
	// 	svcCtx: svcCtx,
	// 	config: config,
	// }
	return nil
}

// NewLifecycleManager 创建生命周期管理器
func NewLifecycleManager(svcCtx *svc.ServiceContext, config *Configuration) LifecycleManager {
	// TODO: 返回具体实现
	// return &lifecycleManagerImpl{
	// 	svcCtx: svcCtx,
	// 	config: config,
	// }
	return nil
}

// NewApprovalManager 创建审批管理器
func NewApprovalManager(svcCtx *svc.ServiceContext, config *Configuration) ApprovalManager {
	// TODO: 返回具体实现
	// return &approvalManagerImpl{
	// 	svcCtx: svcCtx,
	// 	config: config,
	// }
	return nil
}

// NewDataPersister 创建数据持久化器
func NewDataPersister(svcCtx *svc.ServiceContext, config *Configuration) DataPersister {
	// TODO: 返回具体实现
	// return &dataPersisterImpl{
	// 	svcCtx: svcCtx,
	// 	config: config,
	// }
	return nil
}
