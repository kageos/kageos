package service

func runtimeTools(r *ToolRegistry) []Tool {
	return []Tool{
		&RunTableSearchTool{},
		&RunFormSubmitTool{},
		&RunChartQueryTool{},
		&RunTableCreateTool{},
		&RunTableUpdateTool{},
		&RunTableDeleteTool{},
		&CreateScheduledFunctionTaskTool{},
		&CreateScheduledAgentTaskTool{},
		&ListScheduledTasksTool{},
		&ManageScheduledTaskTool{},
		&ListScheduledTaskExecutionsTool{},
		&RunPythonTool{},
		&RunOnSelectFuzzyTool{},
	}
}
