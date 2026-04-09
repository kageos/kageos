package server

import (
	"context"
	"fmt"

	v1 "github.com/ai-agent-os/ai-agent-os/core/app-runtime/api/v1"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// RegisterNATS 注册所有 NATS 订阅（类似 Gin 的 router：subject -> handler）
// subject 真值统一放在 pkg/subjects，这里只做路由装配。
func RegisterNATS(ctx context.Context, conn *nats.Conn, subs *[]*nats.Subscription,
	appH *v1.AppHandler,
	serviceTreeH *v1.ServiceTreeHandler,
	workspaceH *v1.WorkspaceHandler,
	requestH *v1.RequestHandler,
) error {
	var err error
	var sub *nats.Subscription

	// ---------- App ----------
	sub, err = conn.QueueSubscribe(subjects.RuntimeAppCreateCommandSubject, subjects.RuntimeAppCreateQueueGroup, appH.HandleAppCreate)
	if err != nil {
		return fmt.Errorf("subscribe app create: %w", err)
	}
	*subs = append(*subs, sub)

	sub, err = conn.QueueSubscribe(subjects.RuntimeAppUpdateCommandSubject, subjects.RuntimeAppUpdateQueueGroup, appH.HandleAppUpdate)
	if err != nil {
		return fmt.Errorf("subscribe app update: %w", err)
	}
	*subs = append(*subs, sub)

	sub, err = conn.QueueSubscribe(subjects.RuntimeAppDeleteCommandSubject, subjects.RuntimeAppDeleteQueueGroup, appH.HandleAppDelete)
	if err != nil {
		return fmt.Errorf("subscribe app delete: %w", err)
	}
	*subs = append(*subs, sub)

	// ---------- ServiceTree ----------
	sub, err = conn.QueueSubscribe(subjects.RuntimeServiceTreeCreateCommandSubject, subjects.RuntimeServiceTreeCreateQueueGroup, serviceTreeH.HandleServiceTreeCreate)
	if err != nil {
		return fmt.Errorf("subscribe service tree create: %w", err)
	}
	*subs = append(*subs, sub)

	sub, err = conn.QueueSubscribe(subjects.RuntimeServiceTreeDeleteCommandSubject, subjects.RuntimeServiceTreeDeleteQueueGroup, serviceTreeH.HandleServiceTreeDelete)
	if err != nil {
		return fmt.Errorf("subscribe service tree delete: %w", err)
	}
	*subs = append(*subs, sub)

	sub, err = conn.QueueSubscribe(subjects.RuntimeDirectoryTreeBatchCreateCommandSubject, subjects.RuntimeDirectoryTreeBatchCreateQueueGroup, serviceTreeH.HandleBatchCreateDirectoryTree)
	if err != nil {
		return fmt.Errorf("subscribe batch create directory tree: %w", err)
	}
	*subs = append(*subs, sub)

	sub, err = conn.QueueSubscribe(subjects.RuntimeFileBatchWriteCommandSubject, subjects.RuntimeFileBatchWriteQueueGroup, serviceTreeH.HandleBatchWriteFiles)
	if err != nil {
		return fmt.Errorf("subscribe batch write files: %w", err)
	}
	*subs = append(*subs, sub)

	// ---------- Workspace (read/replace/delete file) ----------
	sub, err = conn.QueueSubscribe(subjects.RuntimeDirectoryFilesReadQuerySubject, subjects.RuntimeDirectoryFilesReadQueueGroup, workspaceH.HandleReadDirectoryFiles)
	if err != nil {
		return fmt.Errorf("subscribe read directory files: %w", err)
	}
	*subs = append(*subs, sub)

	sub, err = conn.QueueSubscribe(subjects.RuntimeFileReplaceBatchCommandSubject, subjects.RuntimeFileReplaceBatchQueueGroup, workspaceH.HandleReplaceInFileBatch)
	if err != nil {
		return fmt.Errorf("subscribe replace in file batch: %w", err)
	}
	*subs = append(*subs, sub)

	sub, err = conn.QueueSubscribe(subjects.RuntimeFileDeleteCommandSubject, subjects.RuntimeFileDeleteQueueGroup, workspaceH.HandleDeleteFile)
	if err != nil {
		return fmt.Errorf("subscribe delete file: %w", err)
	}
	*subs = append(*subs, sub)

	sub, err = conn.QueueSubscribe(subjects.RuntimeAppLogReadQuerySubject, subjects.RuntimeAppLogReadQueueGroup, workspaceH.HandleReadAppLog)
	if err != nil {
		return fmt.Errorf("subscribe read app log: %w", err)
	}
	*subs = append(*subs, sub)

	// ---------- Request (app-server -> app) ----------
	sub, err = conn.QueueSubscribe(subjects.RuntimeAppInvokeCommandSubjectPattern, subjects.RuntimeAppInvokeQueueGroup, requestH.HandleAppServerInvokeRequest)
	if err != nil {
		return fmt.Errorf("subscribe app-server invoke request: %w", err)
	}
	*subs = append(*subs, sub)

	logger.Infof(ctx, "[NATS Router] Registered %d subscriptions", len(*subs))
	return nil
}
