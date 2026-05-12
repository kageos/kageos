package service

func workspaceTools(r *ToolRegistry) []Tool {
	return []Tool{
		&ChangeRoleTool{},
		&SummarizeTaskStateTool{},
		&ReadGoFileTool{},
		&ReadGoFileLinesTool{},
		&ReadDocTool{},
		&ReadDirTool{},
		&CreateDirectoryTool{},
		&WritePRDTool{},
		&WriteDocTool{},
		&CreateWorkflowTool{},
		&WriteGoFileTool{},
		&BuildWorkspaceTool{},
		&SearchReplaceFileTool{},
		&DeleteFileTool{},
		&ReadAppLogTool{},
	}
}
