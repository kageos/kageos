package service

func runtimeTools(r *ToolRegistry) []Tool {
	return []Tool{
		&RunTableSearchTool{},
		&RunFormSubmitTool{},
		&RunChartQueryTool{},
		&RunTableCreateTool{},
		&RunTableBatchCreateTool{},
		&RunTableUpdateTool{},
		&RunTableDeleteTool{},
		&CreateScheduledTaskTool{},
		&ListScheduledTasksTool{},
		&CancelScheduledTaskTool{},
		&ListScheduledTaskExecutionsTool{},
		&CreateScheduledAgentTaskTool{registry: r},
		&ListScheduledAgentTasksTool{registry: r},
		&ListScheduledAgentTaskExecutionsTool{registry: r},
		&RunScheduledAgentTaskNowTool{registry: r},
		&RunPythonTool{},
		&RunOnSelectFuzzyTool{},
	}
}
