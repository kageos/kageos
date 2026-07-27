package service

func workspaceTools(r *ToolRegistry) []Tool {
	return []Tool{
		&ChangeRoleTool{},
		&SummarizeTaskStateTool{},
		&ReadWorkspaceArtifactTool{},
		&SearchSessionHistoryTool{},
		&ReadSessionMessagesTool{},
		&ReadFileTool{},
		&ReadDocTool{},
		&ReadDirTool{},
		&CreateDirectoryTool{},
		&WritePRDTool{},
		&WriteFileTool{},
		&EditFileTool{},
		&BuildWorkspaceTool{},
		&DeleteFileTool{},
		&ReadAppLogTool{},
	}
}
