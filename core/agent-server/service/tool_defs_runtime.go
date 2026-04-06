package service

func runtimeTools(r *ToolRegistry) []Tool {
	return []Tool{
		&RunTableSearchTool{},
		&RunFormSubmitTool{},
		&RunChartQueryTool{},
		&RunTableCreateTool{},
		&RunTableUpdateTool{},
		&CreateScheduledTaskTool{},
		&ListScheduledTasksTool{},
		&CancelScheduledTaskTool{},
		&ListScheduledTaskExecutionsTool{},
		&RunOfficialPythonTool{},
		&RunOnSelectFuzzyTool{},
	}
}
