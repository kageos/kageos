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
		&RunOfficialPythonTool{},
		&RunOnSelectFuzzyTool{},
	}
}
