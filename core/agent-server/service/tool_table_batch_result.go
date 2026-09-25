package service

// Preserve per-row results on errors so callers can reconcile partial writes.
func tableBatchWriteResult(out map[string]interface{}, succeeded, failed int, notice string) ToolResult {
	status := "success"
	if failed > 0 {
		status = "failed"
		if succeeded > 0 {
			status = "partial_success"
		}
		out["message"] = "请按逐条结果核对已完成和失败项，不要重复提交已完成项。"
	}
	out["status"] = status
	return toolResultWithStructuredData(out, failed > 0, notice)
}
