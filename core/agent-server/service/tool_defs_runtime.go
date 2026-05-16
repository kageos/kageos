package service

func runtimeTools(r *ToolRegistry) []Tool {
	return []Tool{
		&RunTableSearchTool{},
		&RunFormSubmitTool{},
		&RunChartQueryTool{},
		&RunTableCreateTool{},
		&RunTableUpdateTool{},
		&RunTableDeleteTool{},
		&RunPythonTool{},
		&RunOnSelectFuzzyTool{},
	}
}
