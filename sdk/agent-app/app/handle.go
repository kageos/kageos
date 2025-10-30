package app

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/nats-io/nats.go"
)

// handleMessageAsync 异步处理接收到的消息
func (a *App) handleMessageAsync(msg *nats.Msg) {
	// 立即启动 goroutine 处理，避免阻塞 NATS 订阅
	go a.handleMessage(msg)
}

// handleMessage 处理接收到的消息
func (a *App) handleMessage(msg *nats.Msg) {
	ctx := context.Background()

	// 检查是否已经请求关闭
	a.shutdownMu.RLock()
	if a.shutdownRequested {
		a.shutdownMu.RUnlock()
		logger.Warnf(ctx, "Shutdown requested, rejecting new request")
		return
	}
	a.shutdownMu.RUnlock()

	var req dto.RequestAppReq
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		a.sendErrResponse(&dto.RequestAppResp{Error: err.Error(), TraceId: msg.Header.Get("trace_id")})
		logger.Errorf(context.Background(), err.Error())
		return
	}

	// 增加运行中函数计数
	a.incrementRunningCount()

	defer a.decrementRunningCount()
	logger.Infof(ctx, "handleMessage req:%+v", req)
	resp, err := a.handle(&req)
	if err != nil {
		a.sendErrResponse(resp)
		logger.Errorf(context.Background(), err.Error())
		return
	}
	logger.Infof(ctx, "handleMessage req:%+v", req)
	a.sendResponse(resp)
}

func (a *App) handle(req *dto.RequestAppReq) (resp *dto.RequestAppResp, err error) {
	// 添加 panic 恢复机制
	defer func() {
		if r := recover(); r != nil {
			// 获取完整的堆栈信息
			stack := debug.Stack()

			// 将 panic 转换为 error，包含堆栈信息
			var panicMsg string
			if panicErr, ok := r.(error); ok {
				panicMsg = panicErr.Error()
			} else {
				panicMsg = fmt.Sprintf("%v", r)
			}

			// 创建包含堆栈信息的错误
			err = fmt.Errorf("panic occurred: %s\nStack trace:\n%s", panicMsg, string(stack))

			resp.Error = err.Error()
			resp.TraceId = req.TraceId
			// 记录详细的 panic 信息到日志
			logger.Errorf(context.Background(), "Handler panic recovered: %s\nStack trace:\n%s", panicMsg, string(stack))
			return
		}
	}()

	// 解析请求
	//var req dto.RequestAppReq
	//if err := json.Unmarshal(msg.Data, &req); err != nil {
	//	return nil, err
	//}
	ctx := context.Background()
	newContext, err := NewContext(ctx, req)
	if err != nil {
		return &dto.RequestAppResp{Result: nil, Error: err.Error(), TraceId: newContext.msg.TraceId}, err
	}

	// TODO: 这里调用具体的业务逻辑处理
	// result := handleBusinessLogic(req.Method, req.Body, req.UrlQuery)

	logger.Infof(ctx, "Handle req:%+v", req)
	router, err := a.getRouter(newContext.msg.Router, newContext.msg.Method)
	if err != nil {
		logger.Errorf(ctx, err.Error())
		// 发送响应（带上 trace_id）
		return &dto.RequestAppResp{Result: nil, Error: err.Error(), TraceId: newContext.msg.TraceId}, err
	}
	handleFunc := router.HandleFunc

	var res response.RunFunctionResp
	err = handleFunc(newContext, &res)
	if err != nil {
		logger.Errorf(ctx, "handleFunc err:%s", err.Error())
		return &dto.RequestAppResp{Result: nil, Error: err.Error(), TraceId: newContext.msg.TraceId}, err
	}
	logger.Infof(ctx, "handleFunc req:%+v", req)

	// 退出命令
	if newContext.msg.Method == "exit" {
		a.Close()
	}
	return &dto.RequestAppResp{Result: res.Data(), TraceId: newContext.msg.TraceId}, nil
}
