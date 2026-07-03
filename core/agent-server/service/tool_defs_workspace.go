package service

func workspaceTools(r *ToolRegistry) []Tool {
	return []Tool{
		&ChangeRoleTool{},
		&SummarizeTaskStateTool{},
		&ReadFileTool{},
		&ReadDocTool{},
		&ReadDirTool{},
		&CreateDirectoryTool{},
		&WritePRDTool{},
		&WriteDocTool{},
		&WriteFileTool{},
		&EditFileTool{},
		&BuildWorkspaceTool{},
		&DeleteFileTool{},
		&ReadAppLogTool{},
	}
}
