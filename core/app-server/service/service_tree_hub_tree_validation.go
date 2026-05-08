package service

import (
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/functionschema"
	"github.com/ai-agent-os/ai-agent-os/pkg/naming"
	"github.com/ai-agent-os/ai-agent-os/pkg/servicetree"
)

type hubDirectoryTreeValidationOptions struct {
	requirePath      bool
	validatePathTree bool
}

func validateHubDirectoryTreeForPublishImpl(tree *dto.DirectoryTreeNode) error {
	return validateHubDirectoryTreeImpl(tree, hubDirectoryTreeValidationOptions{
		requirePath:      true,
		validatePathTree: true,
	})
}

func validateHubDirectoryTreeForInstallImpl(tree *dto.DirectoryTreeNode) error {
	return validateHubDirectoryTreeImpl(tree, hubDirectoryTreeValidationOptions{})
}

func validateHubDirectoryTreeImpl(tree *dto.DirectoryTreeNode, opts hubDirectoryTreeValidationOptions) error {
	if tree == nil {
		return fmt.Errorf("目录树不能为空")
	}

	return validateHubDirectoryTreeNodeImpl(tree, "", "root", opts)
}

func validateHubDirectoryTreeNodeImpl(node *dto.DirectoryTreeNode, parentPath, location string, opts hubDirectoryTreeValidationOptions) error {
	if node == nil {
		return fmt.Errorf("%s 节点不能为空", location)
	}

	code, err := validateHubDirectoryCode(node.Code, location)
	if err != nil {
		return err
	}
	if node.Type != "" && node.Type != servicetree.TypePackage {
		return fmt.Errorf("%s 目录节点类型无效: %s", location, node.Type)
	}

	currentPath := strings.TrimSpace(node.Path)
	if opts.requirePath && currentPath == "" {
		return fmt.Errorf("%s 目录 path 不能为空", location)
	}
	if opts.validatePathTree && parentPath != "" {
		expectedPath := strings.TrimSuffix(parentPath, "/") + "/" + code
		if currentPath != expectedPath {
			return fmt.Errorf("%s 目录 path 与父目录不一致: 期望 %s，实际 %s", location, expectedPath, currentPath)
		}
	}

	seenCodes := make(map[string]struct{}, len(node.Subdirectories))
	for index, function := range node.Functions {
		if err := validateHubFunctionInfoImpl(function, fmt.Sprintf("%s -> functions[%d]", location, index)); err != nil {
			return err
		}
	}
	for index, child := range node.Subdirectories {
		childLocation := fmt.Sprintf("%s -> subdirectories[%d]", location, index)
		if child == nil {
			return fmt.Errorf("%s 节点不能为空", childLocation)
		}

		childCode, err := validateHubDirectoryCode(child.Code, childLocation)
		if err != nil {
			return err
		}
		if _, exists := seenCodes[childCode]; exists {
			return fmt.Errorf("%s 下存在重复子目录 code: %s", location, childCode)
		}
		seenCodes[childCode] = struct{}{}

		if err := validateHubDirectoryTreeNodeImpl(child, currentPath, location+"/"+childCode, opts); err != nil {
			return err
		}
	}

	return nil
}

func validateHubFunctionInfoImpl(function *dto.HubFunctionInfo, location string) error {
	if function == nil {
		return fmt.Errorf("%s 函数不能为空", location)
	}
	if len(function.Schema) == 0 {
		return nil
	}
	schema, err := functionschema.Parse(function.Schema)
	if err != nil {
		return fmt.Errorf("%s 函数 schema 非法: %w", location, err)
	}
	if function.TemplateType != "" && schema.Type != function.TemplateType {
		return fmt.Errorf("%s 函数 template_type 与 schema.type 不一致: template_type=%s schema.type=%s", location, function.TemplateType, schema.Type)
	}
	return nil
}

func validateHubDirectoryCode(rawCode, location string) (string, error) {
	code := strings.TrimSpace(rawCode)
	if code == "" {
		return "", fmt.Errorf("%s 目录 code 不能为空", location)
	}
	if code != rawCode {
		return "", fmt.Errorf("%s 目录 code 不能包含首尾空格: %s", location, rawCode)
	}
	if strings.Contains(code, "/") {
		return "", fmt.Errorf("%s 目录 code 不能包含 /: %s", location, code)
	}
	if err := naming.ValidateGoPackageName(code, "目录 code"); err != nil {
		return "", fmt.Errorf("%s 目录 code 不是合法 Go package 名称: %w", location, err)
	}
	return code, nil
}
