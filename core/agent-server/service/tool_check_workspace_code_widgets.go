package service

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/widget"
)

func checkGoStructTags(file goSourceFileForCheck) []checkWorkspaceCodeIssue {
	return checkParsedGoStructTags(parseGoSourceFileForCheck(file))
}

func checkParsedGoStructTags(file parsedGoSourceFileForCheck) []checkWorkspaceCodeIssue {
	if file.ParseErr != nil {
		return nil
	}
	var issues []checkWorkspaceCodeIssue
	ast.Inspect(file.Parsed, func(n ast.Node) bool {
		field, ok := n.(*ast.Field)
		if !ok || field.Tag == nil {
			return true
		}
		line := file.FileSet.Position(field.Tag.Pos()).Line
		tag := strings.Trim(field.Tag.Value, "`")
		widgetTag := structTagValue(tag, "widget")
		callbackTag := structTagValue(tag, "callback")
		if widgetTag != "" && widgetTag != "-" {
			issues = append(issues, checkWidgetTag(file.Source.Name, line, widgetTag, callbackTag)...)
		}
		return true
	})
	return issues
}

func checkWidgetTag(file string, line int, widgetTag string, callbackTag string) []checkWorkspaceCodeIssue {
	parsed := parseSemicolonTag(widgetTag)
	var issues []checkWorkspaceCodeIssue
	for _, invalid := range invalidSemicolonTagSegments(widgetTag) {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "error",
			Category: "widget_tag",
			Message:  fmt.Sprintf("widget tag 片段必须是 key:value，当前片段为 %q", invalid),
		})
	}
	widgetType := parsed["type"]
	if widgetType == "" {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "error",
			Category: "widget_type",
			Message:  "widget tag 必须显式包含 type，例如 widget:\"name:标题;type:input\"",
		})
		return issues
	}
	if (widgetType == widget.TypeSelect || widgetType == widget.TypeMultiSelect) &&
		parsed["options"] == "" &&
		!strings.Contains(callbackTag, "OnSelectFuzzy") {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "warning",
			Category: "widget_select",
			Message:  "select/multiselect 字段必须有静态 options，或添加 callback:\"OnSelectFuzzy\" 并在对应 Template.OnSelectFuzzyMap 注册；纯存储外键不要写成 select",
		})
	}
	for _, badKey := range []string{"readonly", "multiple"} {
		if _, ok := parsed[badKey]; ok {
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file,
				Line:     line,
				Severity: "error",
				Category: "widget_tag",
				Message:  fmt.Sprintf("widget 参数 %q 不在当前白名单中；只读用 hide 场景或回调控制，文件多选用 type:files + max_count，图片/视频列表预览用 thumbnail:true;list_preview:true", badKey),
			})
		}
	}
	if !widget.IsSupportedType(widgetType) {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "error",
			Category: "widget_type",
			Message:  fmt.Sprintf("unsupported widget type %q；文件上传请用 type:files，不要用 file", widgetType),
		})
	} else {
		allowedKeys := stringSetFromSlice(widget.AllowedTagKeys(widgetType))
		for key := range parsed {
			if _, ok := allowedKeys[key]; ok {
				continue
			}
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file,
				Line:     line,
				Severity: "error",
				Category: "widget_tag",
				Message:  fmt.Sprintf("widget 参数 %q 不在 %q 的白名单中；只读用 hide 场景或回调控制，文件多选用 type:files + max_count，图片/视频列表预览用 thumbnail:true;list_preview:true", key, widgetType),
			})
		}
	}
	if strings.Contains(callbackTag, "OnSelectFuzzy") && widgetType != widget.TypeSelect && widgetType != widget.TypeMultiSelect {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "error",
			Category: "onselect_fuzzy",
			Message:  fmt.Sprintf("callback:\"OnSelectFuzzy\" 只能用于 select 或 multiselect 字段，当前 widget type 为 %q", widgetType),
		})
	}
	if colors := parsed["options_colors"]; colors != "" {
		issues = append(issues, checkOptionsColors(file, line, parsed["options"], colors)...)
	}
	return issues
}

var hexColorRe = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)

func checkOptionsColors(file string, line int, options string, colors string) []checkWorkspaceCodeIssue {
	colorParts := splitNonEmpty(colors, ",")
	var issues []checkWorkspaceCodeIssue
	for _, color := range colorParts {
		if !hexColorRe.MatchString(color) {
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file,
				Line:     line,
				Severity: "error",
				Category: "options_colors",
				Message:  fmt.Sprintf("options_colors 只支持不带 # 的 6 位十六进制 RRGGBB，当前包含 %q", color),
			})
		}
	}
	optionParts := splitNonEmpty(options, ",")
	if options != "" && len(optionParts) != len(colorParts) {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "error",
			Category: "options_colors",
			Message:  fmt.Sprintf("options_colors 数量必须和 options 一致，options=%d colors=%d", len(optionParts), len(colorParts)),
		})
	}
	return issues
}
