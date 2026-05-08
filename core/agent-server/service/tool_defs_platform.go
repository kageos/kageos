package service

func platformTools(r *ToolRegistry) []Tool {
	return []Tool{
		&WebSearchTool{},
		&FetchURLContentTool{},
		&SearchToolsTool{registry: r},
		&RecordWorkspaceEventTool{eventRepo: r.eventRepo},
	}
}
