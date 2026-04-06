package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type PushToHubTool struct{}

type pushToHubArgs struct {
	Directory            string   `json:"directory" schema_desc:"要推送的目录"`
	UpdateDescription    string   `json:"update_description" schema_desc:"本版本更新说明"`
	ServiceFeePersonal   *float64 `json:"service_fee_personal" schema_desc:"个人用户服务费"`
	ServiceFeeEnterprise *float64 `json:"service_fee_enterprise" schema_desc:"企业用户服务费"`
}

var pushToHubToolDef = toolDefinition[pushToHubArgs](
	"push_to_hub",
	"将已发布到应用市场的目录推送更新（版本号由后端自动递增）。可选：directory（不传则当前工作目录）、update_description（本版本更新说明）、service_fee_personal（个人用户服务费，元）、service_fee_enterprise（企业用户服务费，元）。若目录尚未发布过，需先使用 publish_to_hub。",
)

func (t *PushToHubTool) Definition() dto.ToolDef {
	return pushToHubToolDef
}

func (t *PushToHubTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[pushToHubArgs](call.Args)
	if err != nil {
		return toolResult("push_to_hub 参数解析失败: "+err.Error(), true)
	}
	content, isError := runPushToHubTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

func runPushToHubTool(ctx context.Context, args pushToHubArgs, fullCodePath string) (string, bool) {
	dirPath := strings.TrimSpace(args.Directory)
	if dirPath == "" {
		dirPath = fullCodePath
	}
	sourceUser, sourceApp, sourceDirectoryPath, errMsg := parseHubSourceFromPath(dirPath)
	if errMsg != "" {
		return "push_to_hub: " + errMsg, true
	}
	req := &dto.PushDirectoryToHubReq{
		SourceUser:          sourceUser,
		SourceApp:           sourceApp,
		SourceDirectoryPath: sourceDirectoryPath,
		UpdateDescription:   strings.TrimSpace(args.UpdateDescription),
	}
	if args.ServiceFeePersonal != nil && *args.ServiceFeePersonal >= 0 {
		req.ServiceFeePersonal = *args.ServiceFeePersonal
	}
	if args.ServiceFeeEnterprise != nil && *args.ServiceFeeEnterprise >= 0 {
		req.ServiceFeeEnterprise = *args.ServiceFeeEnterprise
	}
	resp, err := apicall.PushDirectoryToHubViaWorkspace(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[PushToHub] 失败: %v", err)
		return "push_to_hub 调用失败: " + err.Error(), true
	}
	return fmt.Sprintf("推送成功。Hub 目录路径: %s，版本: %s -> %s，子目录数: %d，文件数: %d。",
		resp.HubFullCodePath, resp.OldVersion, resp.NewVersion, resp.DirectoryCount, resp.FileCount), false
}
