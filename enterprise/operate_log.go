package enterprise

import (
	dto "github.com/ai-agent-os/ai-agent-os/dto/enterprise"
)

// OperateLogger 操作日志记录器接口
// 用于记录用户在平台上的所有操作行为（新增、更新、删除等）
//
// 设计说明：
//   - 社区版：使用 UnImplOperateLogger 空实现，不记录任何日志
//   - 企业版：使用 enterprise_impl 中的具体实现，记录详细的操作日志
//
// 使用场景：
//   - 在 CallbackApp 中调用，记录所有回调操作
//   - 支持操作审计、合规检查、问题追溯等企业级需求
//
// 实现位置：
//   - 开源实现：UnImplOperateLogger（空实现，社区版使用）
//   - 企业实现：enterprise_impl/operatelog（闭源，企业版使用）
type OperateLogger interface {
	Init // 继承初始化接口，企业实现需要初始化数据库等资源

	// CreateOperateLogger 创建操作日志记录
	// 记录用户的操作行为，包括操作类型、资源信息、数据变更等
	//
	// 参数：
	//   - req: 操作日志请求，包含用户ID、操作类型、资源信息、数据变更等
	//
	// 返回：
	//   - resp: 操作日志响应，包含日志ID等信息
	//   - error: 如果记录失败返回错误
	//
	// 注意：
	//   - 社区版实现直接返回空响应，不执行任何操作
	//   - 企业版实现会将日志持久化到数据库
	CreateOperateLogger(req *dto.CreateOperateLoggerReq) (*dto.CreateOperateLoggerResp, error)
}

// 全局变量：存储当前实现
var operateLoggerImpl OperateLogger = &UnImplOperateLogger{}

// RegisterOperateLogger 注册操作日志记录器实现
// 企业版在 init() 中调用此函数注册真实实现
func RegisterOperateLogger(impl OperateLogger) {
	operateLoggerImpl = impl
}

// GetOperateLogger 获取当前操作日志记录器
// 业务代码通过此函数获取实现（社区版或企业版）
func GetOperateLogger() OperateLogger {
	return operateLoggerImpl
}

// InitOperateLogger 初始化操作日志记录器
// 用于在系统启动时初始化操作日志功能
//
// 参数：
//   - opt: 初始化选项，包含数据库连接等依赖
//
// 返回：
//   - error: 如果初始化失败返回错误
//
// 说明：
//   - 自动使用已注册的实现（社区版或企业版）
//   - 企业版需要在 init() 中调用 RegisterOperateLogger() 注册
func InitOperateLogger(opt *InitOptions) error {
	return operateLoggerImpl.Init(opt)
}

// UnImplOperateLogger 未实现的操作日志记录器（社区版）
// 这是开源版本使用的实现，企业实现会替换为完整实现
//
// 设计目的：
//   - 保持接口一致性，社区版和企业版使用相同的接口
//   - 企业实现会替换为完整实现，记录完整的操作日志
//   - 社区版和企业版都记录完整的操作日志（存储方式相同）
//
// 策略说明：
//   - 社区版：记录完整的操作日志（与企业版一样存储，无保留时间限制）
//   - 企业版：记录完整的操作日志（与企业版一样存储，无保留时间限制）
//   - 查看权限：只有企业版可以查看操作日志（通过 operate_log 查询接口的企业版鉴权中间件控制）
//
// 商业价值：
//   - 升级后能看到完整的历史数据，提升升级体验
//   - "升级后立即查看完整的历史操作记录"是一个很好的卖点
//   - 通过查看权限控制来区分社区版和企业版，而不是通过记录策略
//
// 使用场景：
//   - 开源项目默认使用此实现（空实现，不记录）
//   - 企业版用户购买许可证后，替换为企业实现（完整记录）
type UnImplOperateLogger struct {
	// 空结构体，不需要任何字段
	// 注意：企业实现会替换为完整实现，记录完整的操作日志
}

// Init 初始化方法（空实现）
// 社区版不需要初始化任何资源，直接返回成功
// 注意：企业实现会初始化数据库等资源，用于记录完整的操作日志
func (u *UnImplOperateLogger) Init(opt *InitOptions) error {
	return nil
}

// CreateOperateLogger 创建操作日志
// 社区版实现：空实现，不记录日志
// 企业版实现：完整记录操作日志（与企业版一样存储，无保留时间限制）
//
// 策略：
//   - 社区版：开源版本使用空实现，不记录日志
//   - 企业版：企业实现会替换为完整实现，记录完整的操作日志
//   - 查看权限：只有企业版可以查看操作日志（通过 operate_log 查询接口的企业版鉴权中间件控制）
//
// 注意：
//   - 这个方法会被调用，社区版（开源版本）使用空实现，不记录日志
//   - 企业版实现会在这里记录完整的操作日志到数据库
//   - 企业实现可以选择在社区版时也记录日志（完整记录），升级后能看到完整的历史数据
func (u *UnImplOperateLogger) CreateOperateLogger(req *dto.CreateOperateLoggerReq) (*dto.CreateOperateLoggerResp, error) {
	// 社区版（开源版本）默认实现：不记录日志
	// 企业实现会替换为完整实现，记录完整的操作日志（与企业版一样存储）
	return &dto.CreateOperateLoggerResp{}, nil
}
