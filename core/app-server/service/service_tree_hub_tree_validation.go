package service

import (
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/naming"
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
	if node.Type != "" && node.Type != "package" {
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
