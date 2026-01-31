package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// RunWriteDocTool 写文档工具：按 full_code_path 创建或更新文档，使用现有 CreateDocs / UpdateDocs 接口
// 树节点必有 code（URL 标识）和 name（中文描述）。args: full_code_path、doc_code（URL 标识）、doc_name（中文描述）、content、format
func RunWriteDocTool(ctx context.Context, args map[string]interface{}, defaultFullCodePath string) (content string, isError bool) {
	fullCodePath := strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if fullCodePath == "" {
		fullCodePath = strings.TrimSpace(defaultFullCodePath)
	}
	if fullCodePath == "" {
		return "write_doc 需要 full_code_path（或当前目录上下文）", true
	}
	docCode := strings.TrimSpace(GetStringArg(args, "code"))
	docName := strings.TrimSpace(GetStringArg(args, "name"))
	docContent := GetStringArg(args, "content")
	if docContent == "" {
		return "write_doc 缺少必需参数 content（文档内容）", true
	}
	format := GetStringArg(args, "format")
	if format == "" {
		format = "markdown"
	}
	fullCodePath = strings.Trim(fullCodePath, "/")
	if fullCodePath == "" {
		return "write_doc full_code_path 无效", true
	}
	// 若传了 code，则 full_code_path 视为父目录，文档路径 = full_code_path/code；创建时节点 Code=code，Name=name（不填则用 code）
	if docCode != "" {
		fullCodePath = fullCodePath + "/" + strings.Trim(docCode, "/")
	}
	pathForAPI := "/" + fullCodePath

	// 先查节点是否已存在且为 docs 类型
	detail, err := apicall.GetServiceTreeDetailByFullCodePath(ctx, pathForAPI)
	if err == nil && detail != nil && detail.Type == "docs" {
		// 已存在 docs 节点：更新内容
		contentPtr := &docContent
		formatPtr := &format
		err = apicall.UpdateDocs(ctx, detail.ID, &dto.UpdateDocsReq{Content: contentPtr, Format: formatPtr})
		if err != nil {
			logger.Errorf(ctx, "[WriteDocTool] UpdateDocs 失败: %v", err)
			return "write_doc 更新文档失败: " + err.Error(), true
		}
		logger.Infof(ctx, "[WriteDocTool] 文档已更新 - FullCodePath: %s", pathForAPI)
		return fmt.Sprintf("文档已更新: %s", pathForAPI), false
	}

	// 节点不存在或不是 docs：在父目录下创建新 docs 节点
	parts := strings.Split(fullCodePath, "/")
	if len(parts) < 2 {
		return "write_doc full_code_path 至少需要两段（如 user/app/文档/readme）", true
	}
	segmentCode := parts[len(parts)-1]
	parentPath := "/" + strings.Join(parts[:len(parts)-1], "/")

	parent, err := apicall.GetServiceTreeDetailByFullCodePath(ctx, parentPath)
	if err != nil || parent == nil {
		logger.Warnf(ctx, "[WriteDocTool] 父目录不存在 - ParentPath: %s, error: %v", parentPath, err)
		return "父目录不存在，请先使用 create_directory 创建目录（如「文档」文件夹）后再写文档。", true
	}

	pathParts := strings.Split(strings.Trim(parentPath, "/"), "/")
	if len(pathParts) < 2 {
		return "write_doc 父路径格式无效", true
	}
	user, app := pathParts[0], pathParts[1]

	// 节点必有 code（URL 标识）和 name（中文描述）：Code=segmentCode，Name=doc_name 若传了则用，否则用 segmentCode
	nodeName := docName
	if nodeName == "" {
		nodeName = segmentCode
	}

	req := &dto.CreateDocsReq{
		User:     user,
		App:      app,
		Name:     nodeName,
		Code:     segmentCode,
		ParentID: parent.ID,
		Content:  docContent,
		Format:   format,
	}
	resp, err := apicall.CreateDocs(ctx, req)
	if err != nil {
		// 节点已存在（如先有 package「docs」文件夹，或已有 docs 节点）：尝试更新或给出明确提示
		if strings.Contains(err.Error(), "already exists") {
			existDetail, getErr := apicall.GetServiceTreeDetailByFullCodePath(ctx, pathForAPI)
			if getErr == nil && existDetail != nil {
				if existDetail.Type == "docs" {
					contentPtr := &docContent
					formatPtr := &format
					updateErr := apicall.UpdateDocs(ctx, existDetail.ID, &dto.UpdateDocsReq{Content: contentPtr, Format: formatPtr})
					if updateErr != nil {
						logger.Errorf(ctx, "[WriteDocTool] CreateDocs 已存在后 UpdateDocs 失败: %v", updateErr)
						return "write_doc 更新文档失败: " + updateErr.Error(), true
					}
					logger.Infof(ctx, "[WriteDocTool] 文档已存在，已更新内容 - FullCodePath: %s", pathForAPI)
					return fmt.Sprintf("文档已更新: %s", pathForAPI), false
				}
				// 已存在的是 package（文件夹），不能覆盖为文档，提示在子路径写文档
				return fmt.Sprintf("该路径下已存在同名目录（文件夹）「%s」，请在该目录下写文档，例如 full_code_path 填该文件夹路径，code 填 readme，name 填「项目文档」", segmentCode), true
			}
		}
		logger.Errorf(ctx, "[WriteDocTool] CreateDocs 失败: %v", err)
		return "write_doc 创建文档失败: " + err.Error(), true
	}
	logger.Infof(ctx, "[WriteDocTool] 文档已创建 - FullCodePath: %s, ID: %d", resp.FullCodePath, resp.ID)
	return fmt.Sprintf("文档已创建: %s", resp.FullCodePath), false
}

// RunCreateDirectoryTool 创建目录工具：在 directory（父目录）下创建 package 类型子目录
// args: directory（可选）、name（必填）、code（必填）、description、tags、admins 可选
func RunCreateDirectoryTool(ctx context.Context, args map[string]interface{}, defaultFullCodePath string) (content string, isError bool) {
	fullCodePath := getDirectory(args, defaultFullCodePath)
	if fullCodePath == "" {
		return "create_directory 需要 directory（或当前目录上下文）", true
	}
	name := GetStringArg(args, "name")
	code := GetStringArg(args, "code")
	if name == "" || code == "" {
		return "create_directory 缺少必需参数 name 或 code（目录名称与代码，如 name=\"文档\" code=\"docs\"）", true
	}
	description := GetStringArg(args, "description")
	tags := strings.TrimSpace(GetStringArg(args, "tags"))
	admins := strings.TrimSpace(GetStringArg(args, "admins"))
	if admins == "" {
		admins = contextx.GetRequestUser(ctx)
		if admins == "" {
			admins = "system"
		}
	}

	fullCodePath = strings.Trim(fullCodePath, "/")
	pathForAPI := "/" + fullCodePath

	parent, err := apicall.GetServiceTreeDetailByFullCodePath(ctx, pathForAPI)
	if err != nil || parent == nil {
		logger.Warnf(ctx, "[CreateDirectoryTool] 父目录不存在 - FullCodePath: %s, error: %v", pathForAPI, err)
		return "父目录不存在，请确认 directory 正确。", true
	}

	pathParts := strings.Split(fullCodePath, "/")
	if len(pathParts) < 2 {
		return "create_directory 路径格式无效", true
	}
	user, app := pathParts[0], pathParts[1]

	req := &dto.CreatePackageReq{
		User:        user,
		App:         app,
		Name:        name,
		Code:        code,
		ParentID:    parent.ID,
		Description: description,
		Tags:        tags,
		Admins:      admins,
	}
	resp, err := apicall.CreatePackage(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[CreateDirectoryTool] CreatePackage 失败: %v", err)
		return "create_directory 创建目录失败: " + err.Error(), true
	}
	logger.Infof(ctx, "[CreateDirectoryTool] 目录已创建 - FullCodePath: %s, ID: %d", resp.FullCodePath, resp.ID)
	return fmt.Sprintf("目录已创建: %s（可在该目录下使用 write_go_file 写代码、write_doc 写文档）", resp.FullCodePath), false
}
