package service

func platformTools(r *ToolRegistry) []Tool {
	return []Tool{
		&SearchSkillsTool{},
		&ReadSkillTool{},
		&WebSearchTool{},
		&FetchURLContentTool{},
		&SearchToolsTool{registry: r},
		&PublishToHubTool{},
		&PushToHubTool{},
		&SearchHubDirectoryTool{},
		&CopyDirectoryTool{},
		&RecordWorkspaceEventTool{eventRepo: r.eventRepo},
	}
}
