package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type writeDocCommand struct {
	FullCodePath string
	Name         string
	Code         string
	Content      string
	Format       string
}

type createDirectoryCommand struct {
	Directory    string
	FullCodePath string
	Name         string
	Code         string
	Description  string
	Tags         string
	Admins       string
}

// runWriteDocCommand 按 full_code_path 创建或更新文档，使用现有 CreateDocs / UpdateDocs 接口。
// 树节点必有 code（URL 标识）和 name（中文描述）。
func runWriteDocCommand(ctx context.Context, cmd writeDocCommand, defaultFullCodePath string) (content string, isError bool) {
	fullCodePath := resolveFullCodePathArg(cmd.FullCodePath, defaultFullCodePath)
	if fullCodePath == "" {
		return "write_doc 需要 full_code_path（或当前目录上下文）", true
	}
	docCode := strings.TrimSpace(cmd.Code)
	docName := strings.TrimSpace(cmd.Name)
	docContent := cmd.Content
	if docContent == "" {
		return "write_doc 缺少必需参数 content（文档内容）", true
	}
	format := strings.TrimSpace(cmd.Format)
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
		User:               user,
		App:                app,
		Name:               nodeName,
		Code:               segmentCode,
		ParentFullCodePath: parentPath,
		Content:            docContent,
		Format:             format,
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

// runCreateDirectoryCommand 在 directory（父目录）下创建 package 类型子目录。
func runCreateDirectoryCommand(ctx context.Context, cmd createDirectoryCommand, defaultFullCodePath string) (content string, isError bool) {
	fullCodePath := resolveDirectoryArg(cmd.Directory, cmd.FullCodePath, defaultFullCodePath)
	if fullCodePath == "" {
		return "create_directory 需要 directory（或当前目录上下文）", true
	}
	name := strings.TrimSpace(cmd.Name)
	code := strings.TrimSpace(cmd.Code)
	if name == "" || code == "" {
		return "create_directory 缺少必需参数 name 或 code（目录名称与代码，如 name=\"文档\" code=\"docs\"）", true
	}
	description := cmd.Description
	tags := strings.TrimSpace(cmd.Tags)
	admins := strings.TrimSpace(cmd.Admins)
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
		User:               user,
		App:                app,
		Name:               name,
		Code:               code,
		ParentFullCodePath: pathForAPI,
		Description:        description,
		Tags:               tags,
		Admins:             admins,
	}
	resp, err := apicall.CreatePackage(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[CreateDirectoryTool] CreatePackage 失败: %v", err)
		return "create_directory 创建目录失败: " + err.Error(), true
	}
	logger.Infof(ctx, "[CreateDirectoryTool] 目录已创建 - FullCodePath: %s, ID: %d", resp.FullCodePath, resp.ID)
	// 返回时必须带 init_.go 的完整真实代码，不能省略；若 API 返回的 FullCodePath 不足以构造，用父路径+code 拼出
	pathForInit := strings.Trim(resp.FullCodePath, "/")
	if pathForInit == "" {
		pathForInit = strings.Trim(fullCodePath, "/") + "/" + code
	}
	initGo := prompt.BuildInitGoContent(pathForInit, resp.Name, resp.Description)
	if initGo == "" {
		initGo = prompt.BuildInitGoContent(strings.Trim(fullCodePath, "/")+"/"+code, name, description)
	}
	// 始终返回带完整 init_.go 真实代码的那句，不返回“省略”版短句
	return fmt.Sprintf(`目录已创建: %s。

系统已自动在该目录下生成 `+"`init_.go`"+`，完整内容如下（无需再 write_go_file 创建 init.go 或 init_.go，可直接在该目录下写业务 .go 并用 packageContext.GET(...) 注册路由）：

`+"```go\n%s\n```", resp.FullCodePath, initGo), false
}
