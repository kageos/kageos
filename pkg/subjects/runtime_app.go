package subjects

import "fmt"

const (
	// runtime / app 调用链
	RuntimeAppCreateCommandSubject                = "runtime.v1.cmd.app.create"
	RuntimeAppUpdateCommandSubject                = "runtime.v1.cmd.app.update"
	RuntimeAppDeleteCommandSubject                = "runtime.v1.cmd.app.delete"
	RuntimeServiceTreeDeleteCommandSubject        = "runtime.v1.cmd.service-tree.delete"
	RuntimeServiceTreeUpdateCommandSubject        = "runtime.v1.cmd.service-tree.update"
	RuntimeDirectoryFilesReadQuerySubject         = "runtime.v1.query.directory-files.read"
	RuntimeFileReplaceBatchCommandSubject         = "runtime.v1.cmd.file.replace-batch"
	RuntimeFileDeleteCommandSubject               = "runtime.v1.cmd.file.delete"
	RuntimeAppLogReadQuerySubject                 = "runtime.v1.query.app-log.read"
	RuntimeDirectoryTreeBatchCreateCommandSubject = "runtime.v1.cmd.directory-tree.batch-create"
	RuntimeFileBatchWriteCommandSubject           = "runtime.v1.cmd.file.batch-write"
	RuntimeDirectoryTreeReplaceCommandSubject     = "runtime.v1.cmd.directory-tree.replace"
	RuntimeNamespaceCreateCommandSubject          = "runtime.v1.cmd.namespace.create"
	RuntimeAppInvokeCommandSubjectPattern         = "runtime.v1.cmd.app.invoke.*.*.*"
	RuntimeLifecycleEventSubjectPattern           = "runtime.v1.event.lifecycle.*.*.*"

	AppControlSubjectPattern              = "app.v1.cmd.control.*.*.*"
	AppDiscoveryRequestSubject            = "app.v1.cmd.discovery.request"
	AppServerAppInvokeReplySubjectPattern = "app-server.v1.reply.app.invoke.*.*.*"

	RuntimeAppCreateQueueGroup                = "app-runtime-create-workers"
	RuntimeAppUpdateQueueGroup                = "app-runtime-update-workers"
	RuntimeAppDeleteQueueGroup                = "app-runtime-delete-workers"
	RuntimeServiceTreeDeleteQueueGroup        = "app-runtime-delete-service-tree-workers"
	RuntimeDirectoryTreeBatchCreateQueueGroup = "app-runtime-batch-create-directory-tree-workers"
	RuntimeFileBatchWriteQueueGroup           = "app-runtime-batch-write-files-workers"
	RuntimeDirectoryTreeReplaceQueueGroup     = "app-runtime-replace-directory-tree-workers"
	RuntimeDirectoryFilesReadQueueGroup       = "app-runtime-read-directory-files-workers"
	RuntimeFileReplaceBatchQueueGroup         = "app-runtime-replace-in-file-batch-workers"
	RuntimeFileDeleteQueueGroup               = "app-runtime-delete-file-workers"
	RuntimeAppLogReadQueueGroup               = "app-runtime-read-app-log-workers"
	RuntimeAppInvokeQueueGroup                = "app-runtime-request-workers"
)

// BuildAppInvokeSubject 构建 runtime -> app 的调用主题。
func BuildAppInvokeSubject(user, app, version string) string {
	return fmt.Sprintf("app.v1.cmd.invoke.%s.%s.%s", user, app, version)
}

// BuildAppServerAppInvokeReplySubject 构建 app -> app-server 的异步回复主题。
func BuildAppServerAppInvokeReplySubject(user, app, version string) string {
	return fmt.Sprintf("app-server.v1.reply.app.invoke.%s.%s.%s", user, app, version)
}

// BuildAppControlSubject 构建 runtime -> app 的控制主题。
// 当前用于 shutdown 与 onAppUpdate request-reply。
func BuildAppControlSubject(user, app, version string) string {
	return fmt.Sprintf("app.v1.cmd.control.%s.%s.%s", user, app, version)
}

// BuildRuntimeLifecycleEventSubject 构建 app -> runtime 的生命周期事件主题。
func BuildRuntimeLifecycleEventSubject(user, app, version string) string {
	return fmt.Sprintf("runtime.v1.event.lifecycle.%s.%s.%s", user, app, version)
}

// BuildRuntimeAppInvokeCommandSubject 构建 app-server -> runtime 的应用调用主题。
func BuildRuntimeAppInvokeCommandSubject(user, app, version string) string {
	return fmt.Sprintf("runtime.v1.cmd.app.invoke.%s.%s.%s", user, app, version)
}
