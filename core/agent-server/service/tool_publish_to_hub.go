package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type PublishToHubTool struct{}

type publishToHubArgs struct {
	Directory            string   `json:"directory" schema_desc:"要发布的目录"`
	Name                 string   `json:"name" schema_desc:"应用市场中的目录名称" schema_required:"true"`
	Description          string   `json:"description" schema_desc:"目录描述"`
	Category             string   `json:"category" schema_desc:"分类"`
	Tags                 string   `json:"tags" schema_desc:"标签，逗号分隔"`
	ServiceFeePersonal   *float64 `json:"service_fee_personal" schema_desc:"个人用户服务费"`
	ServiceFeeEnterprise *float64 `json:"service_fee_enterprise" schema_desc:"企业用户服务费"`
}

var publishToHubToolDef = toolDefinition[publishToHubArgs](
	"publish_to_hub",
	"将当前工作区目录或指定目录首次发布到应用市场（Hub）。必填：name（在应用市场上的目录名称）。可选：directory（不传则使用当前工作目录）、description、category、tags（逗号分隔，支持自定义标签）、service_fee_personal（个人用户服务费，元）、service_fee_enterprise（企业用户服务费，元）。发布成功后返回 hub 目录路径与文件统计。",
)

func (t *PublishToHubTool) Definition() dto.ToolDef {
	return publishToHubToolDef
}

func (t *PublishToHubTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[publishToHubArgs](call.Args)
	if err != nil {
		return toolResult("publish_to_hub 参数解析失败: "+err.Error(), true)
	}
	content, isError := runPublishToHubTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

func runPublishToHubTool(ctx context.Context, args publishToHubArgs, fullCodePath string) (string, bool) {
	dirPath := strings.TrimSpace(args.Directory)
	if dirPath == "" {
		dirPath = fullCodePath
	}
	sourceUser, sourceApp, sourceDirectoryPath, errMsg := parseHubSourceFromPath(dirPath)
	if errMsg != "" {
		return "publish_to_hub: " + errMsg, true
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return "publish_to_hub 必填 name（在应用市场上的目录名称）。", true
	}
	req := &dto.PublishDirectoryToHubReq{
		SourceUser:          sourceUser,
		SourceApp:           sourceApp,
		SourceDirectoryPath: sourceDirectoryPath,
		Name:                name,
		Description:         strings.TrimSpace(args.Description),
		Category:            strings.TrimSpace(args.Category),
	}
	if tagsStr := strings.TrimSpace(args.Tags); tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				req.Tags = append(req.Tags, t)
			}
		}
	}
	if args.ServiceFeePersonal != nil && *args.ServiceFeePersonal >= 0 {
		req.ServiceFeePersonal = *args.ServiceFeePersonal
	}
	if args.ServiceFeeEnterprise != nil && *args.ServiceFeeEnterprise >= 0 {
		req.ServiceFeeEnterprise = *args.ServiceFeeEnterprise
	}
	resp, err := apicall.PublishDirectoryToHubViaWorkspace(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[PublishToHub] 失败: %v", err)
		return "publish_to_hub 调用失败: " + err.Error(), true
	}
	return fmt.Sprintf("发布成功。Hub 目录路径: %s，子目录数: %d，文件数: %d。",
		resp.HubFullCodePath, resp.DirectoryCount, resp.FileCount), false
}
