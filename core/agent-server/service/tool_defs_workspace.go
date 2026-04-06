package service

func workspaceTools(r *ToolRegistry) []Tool {
	return []Tool{
		&ReadGoFileTool{},
		&ReadGoFileLinesTool{},
		&ReadDocTool{},
		&ReadDirTool{},
		&CreateDirectoryTool{},
		&WriteDocTool{},
		&WriteGoFileTool{},
		&BuildWorkspaceTool{},
		&SearchReplaceFileTool{},
		&DeleteFileTool{},
		&ReadAppLogTool{},
	}
}
