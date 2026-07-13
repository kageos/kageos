package service

import (
	"fmt"
	"path"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
)

func validateCapabilityBundle(bundle *dto.CapabilityBundle) error {
	if bundle == nil {
		return fmt.Errorf("目录 JSON 不能为空")
	}
	if bundle.SchemaVersion != dto.CapabilityBundleSchemaVersion {
		return fmt.Errorf("不支持的目录 JSON schema_version: %s", bundle.SchemaVersion)
	}
	if len(bundle.Files) == 0 && len(bundle.Packages) == 0 {
		if len(bundle.Docs) == 0 {
			return fmt.Errorf("目录 JSON 必须包含 files、packages 或 docs")
		}
	}
	if err := validateCapabilityBundleTreeNodes(bundle.TreeNodes); err != nil {
		return err
	}
	if err := validateCapabilityBundleDocs(bundle.Docs, bundle.TreeNodes); err != nil {
		return err
	}

	seenPackages := make(map[string]struct{}, len(bundle.Packages))
	for index, pkg := range bundle.Packages {
		if pkg == nil {
			return fmt.Errorf("packages[%d] 不能为空", index)
		}
		normalized, err := validateCapabilityPackagePath(pkg.Path, fmt.Sprintf("packages[%d].path", index), false)
		if err != nil {
			return err
		}
		if normalized != pkg.Path {
			return fmt.Errorf("packages[%d].path 必须使用规范相对路径: %s", index, pkg.Path)
		}
		if _, exists := seenPackages[normalized]; exists {
			return fmt.Errorf("目录 JSON 存在重复 package 路径: %s", normalized)
		}
		seenPackages[normalized] = struct{}{}
	}
	if err := validateCapabilityBundleAgentTasks(bundle.AgentTasks, seenPackages); err != nil {
		return err
	}
	seenFiles := make(map[string]struct{}, len(bundle.Files))
	for index, file := range bundle.Files {
		if file == nil {
			return fmt.Errorf("files[%d] 不能为空", index)
		}
		packagePath, err := validateCapabilityPackagePath(file.PackagePath, fmt.Sprintf("files[%d].package_path", index), true)
		if err != nil {
			return err
		}
		if packagePath != file.PackagePath {
			return fmt.Errorf("files[%d].package_path 必须使用规范相对路径: %s", index, file.PackagePath)
		}
		if packagePath != "" {
			if _, exists := seenPackages[packagePath]; !exists {
				return fmt.Errorf("files[%d].package_path 未在 packages 中声明: %s", index, packagePath)
			}
		}

		filePath, err := validateCapabilityFilePath(file.Path, fmt.Sprintf("files[%d].path", index))
		if err != nil {
			return err
		}
		if filePath != file.Path {
			return fmt.Errorf("files[%d].path 必须使用规范相对文件名: %s", index, file.Path)
		}
		key := capabilityFileKey(packagePath, filePath)
		if _, exists := seenFiles[key]; exists {
			return fmt.Errorf("目录 JSON 存在重复文件路径: %s", key)
		}
		seenFiles[key] = struct{}{}
	}
	return nil
}

func validateCapabilityBundleAgentTasks(tasks []*dto.CapabilityBundleAgentTask, packagePaths map[string]struct{}) error {
	seen := make(map[string]struct{}, len(tasks))
	for index, task := range tasks {
		if task == nil {
			return fmt.Errorf("agent_tasks[%d] 不能为空", index)
		}
		relativePath, err := validateCapabilityPackagePath(task.RelativePath, fmt.Sprintf("agent_tasks[%d].relative_path", index), false)
		if err != nil {
			return err
		}
		if relativePath != task.RelativePath {
			return fmt.Errorf("agent_tasks[%d].relative_path 必须使用规范相对路径: %s", index, task.RelativePath)
		}
		if _, exists := packagePaths[relativePath]; !exists {
			return fmt.Errorf("agent_tasks[%d].relative_path 未在 packages 中声明: %s", index, relativePath)
		}
		code := normalizeCapabilityAgentTaskCode(task.Code)
		if code == "" {
			return fmt.Errorf("agent_tasks[%d].code 不能为空", index)
		}
		if code != task.Code {
			return fmt.Errorf("agent_tasks[%d].code 必须使用规范标识: %s", index, task.Code)
		}
		if strings.TrimSpace(task.Message) == "" {
			return fmt.Errorf("agent_tasks[%d].message 不能为空", index)
		}
		if err := task.Schedule.Validate(); err != nil {
			return fmt.Errorf("agent_tasks[%d].schedule 无效: %w", index, err)
		}
		policy := strings.TrimSpace(task.Policy)
		if policy != "" && policy != agentTaskPolicyCreateIfMissing {
			return fmt.Errorf("agent_tasks[%d].policy 不支持: %s", index, policy)
		}
		key := capabilityAgentTaskKey(relativePath, code)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("目录 JSON 存在重复 Agent 任务: %s", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateCapabilityBundleDocs(docs []*dto.CapabilityBundleDoc, treeNodes []*dto.CapabilityBundleTreeNode) error {
	nodesByPath := make(map[string]*dto.CapabilityBundleTreeNode, len(treeNodes))
	for _, node := range treeNodes {
		if node != nil {
			nodesByPath[node.RelativePath] = node
		}
	}

	seen := make(map[string]struct{}, len(docs))
	for index, doc := range docs {
		if doc == nil {
			return fmt.Errorf("docs[%d] 不能为空", index)
		}
		relativePath, err := validateCapabilityTreeNodePath(doc.RelativePath, fmt.Sprintf("docs[%d].relative_path", index), false)
		if err != nil {
			return err
		}
		if relativePath != doc.RelativePath {
			return fmt.Errorf("docs[%d].relative_path 必须使用规范相对路径: %s", index, doc.RelativePath)
		}
		if _, exists := seen[relativePath]; exists {
			return fmt.Errorf("目录 JSON 存在重复 docs 路径: %s", relativePath)
		}
		seen[relativePath] = struct{}{}
		if len(nodesByPath) > 0 {
			node, exists := nodesByPath[relativePath]
			if !exists {
				return fmt.Errorf("docs[%d].relative_path 未在 tree_nodes 中声明: %s", index, relativePath)
			}
			if node.Type != model.ServiceTreeTypeDocs {
				return fmt.Errorf("docs[%d].relative_path 对应的 tree node 不是 docs: %s", index, relativePath)
			}
		}
	}
	return nil
}

func validateCapabilityBundleTreeNodes(nodes []*dto.CapabilityBundleTreeNode) error {
	seen := make(map[string]struct{}, len(nodes))
	for index, node := range nodes {
		if node == nil {
			return fmt.Errorf("tree_nodes[%d] 不能为空", index)
		}
		relativePath, err := validateCapabilityTreeNodePath(node.RelativePath, fmt.Sprintf("tree_nodes[%d].relative_path", index), false)
		if err != nil {
			return err
		}
		if relativePath != node.RelativePath {
			return fmt.Errorf("tree_nodes[%d].relative_path 必须使用规范相对路径: %s", index, node.RelativePath)
		}
		if _, exists := seen[relativePath]; exists {
			return fmt.Errorf("目录 JSON 存在重复 tree node 路径: %s", relativePath)
		}
		seen[relativePath] = struct{}{}
		if node.Type != model.ServiceTreeTypePackage && node.Type != model.ServiceTreeTypeFunction && node.Type != model.ServiceTreeTypeDocs {
			return fmt.Errorf("tree_nodes[%d].type 不支持: %s", index, node.Type)
		}
		if strings.TrimSpace(node.Code) == "" {
			return fmt.Errorf("tree_nodes[%d].code 不能为空", index)
		}
		if path.Base(relativePath) != node.Code {
			return fmt.Errorf("tree_nodes[%d].code 必须等于 relative_path 的最后一段: want=%s got=%s", index, path.Base(relativePath), node.Code)
		}
		parentPath, err := validateCapabilityTreeNodePath(node.ParentPath, fmt.Sprintf("tree_nodes[%d].parent_path", index), true)
		if err != nil {
			return err
		}
		expectedParent := capabilityParentPath(relativePath)
		if parentPath != expectedParent {
			return fmt.Errorf("tree_nodes[%d].parent_path 必须等于 relative_path 的父路径: want=%s got=%s", index, expectedParent, parentPath)
		}
	}
	for index, node := range nodes {
		parentPath := strings.TrimSpace(node.ParentPath)
		if parentPath == "" {
			continue
		}
		if _, exists := seen[parentPath]; !exists {
			return fmt.Errorf("tree_nodes[%d].parent_path 未在 tree_nodes 中声明: %s", index, parentPath)
		}
	}
	return nil
}

func validateCapabilityTreeNodePath(nodePath string, field string, allowEmpty bool) (string, error) {
	if nodePath != strings.TrimSpace(nodePath) {
		return "", fmt.Errorf("%s 不能包含首尾空格: %s", field, nodePath)
	}
	if strings.HasPrefix(nodePath, "/") || strings.HasSuffix(nodePath, "/") {
		return "", fmt.Errorf("%s 必须是相对节点路径: %s", field, nodePath)
	}
	nodePath = strings.Trim(nodePath, "/")
	if nodePath == "" {
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("%s 不能为空", field)
	}
	if strings.Contains(nodePath, "\\") || path.IsAbs(nodePath) {
		return "", fmt.Errorf("%s 必须是相对节点路径: %s", field, nodePath)
	}
	if cleaned := path.Clean(nodePath); cleaned != nodePath || cleaned == "." {
		return "", fmt.Errorf("%s 必须使用规范相对路径: %s", field, nodePath)
	}
	parts := strings.Split(nodePath, "/")
	if err := rejectWorkspaceBoundCapabilityPath(parts, field, nodePath); err != nil {
		return "", err
	}
	for _, part := range parts {
		if part == "" || strings.HasPrefix(part, ".") {
			return "", fmt.Errorf("%s 包含非法路径片段: %s", field, nodePath)
		}
	}
	return nodePath, nil
}
