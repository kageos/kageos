package service

import (
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/functionschema"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
)

func metadataForDisplayFileFields(fields ...string) *dto.ToolResultMetadata {
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	if len(out) == 0 {
		return nil
	}
	return &dto.ToolResultMetadata{DisplayFileFields: out}
}

func formResponseDisplayFileFields(ctx context.Context, fullCodePath string) []string {
	fn, err := apicall.GetFunctionInfo(ctx, functionschema.TypeForm, fullCodePath)
	if err != nil || fn == nil || fn.Schema == nil || fn.Schema.Form == nil {
		if err != nil {
			logger.Warnf(ctx, "[ToolResultMetadata] 获取 Form 输出 schema 失败: fullCodePath=%s, err=%v", fullCodePath, err)
		}
		return nil
	}
	return collectDisplayFileFields(fn.Schema.Form.Response)
}

func collectDisplayFileFields(responseFields []*widget.Field) []string {
	fields := make([]string, 0)
	for _, field := range responseFields {
		if field == nil || field.Code == "" {
			continue
		}
		if field.Widget.Type == widget.TypeFiles {
			fields = append(fields, field.Code)
		}
	}
	return fields
}
