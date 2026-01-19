package metadata

import (
	"testing"
)

func TestParseMetadata(t *testing.T) {
	code := `/*
file: requirement.go
directory_name: 需求管理系统
directory_code: requirement
directory_desc: |
  主要是进行需求管理，包括：
  - 需求的创建
  - 需求的更新
  - 需求的查询
tags:
  - 需求管理
  - 项目管理
*/
package requirement

import (
    "fmt"
)

func main() {
    fmt.Println("Hello")
}
`

	var metadata Metadata
	err := ParseMetadata(code, &metadata)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if metadata.File != "requirement.go" {
		t.Errorf("File 期望 'requirement.go'，实际 '%s'", metadata.File)
	}

	if metadata.DirectoryName != "需求管理系统" {
		t.Errorf("DirectoryName 期望 '需求管理系统'，实际 '%s'", metadata.DirectoryName)
	}

	if metadata.DirectoryCode != "requirement" {
		t.Errorf("DirectoryCode 期望 'requirement'，实际 '%s'", metadata.DirectoryCode)
	}

	expectedDesc := "主要是进行需求管理，包括：\n- 需求的创建\n- 需求的更新\n- 需求的查询\n"
	if metadata.DirectoryDesc != expectedDesc {
		t.Errorf("DirectoryDesc 不匹配\n期望: %q\n实际: %q", expectedDesc, metadata.DirectoryDesc)
	}

	if len(metadata.Tags) != 2 {
		t.Errorf("Tags 期望长度 2，实际 %d", len(metadata.Tags))
	}
	if metadata.Tags[0] != "需求管理" || metadata.Tags[1] != "项目管理" {
		t.Errorf("Tags 内容不匹配: %v", metadata.Tags)
	}

}

func TestRemoveMetadata(t *testing.T) {
	code := `/*
file: requirement.go
directory_name: 需求管理系统
*/
package requirement

import (
    "fmt"
)

func main() {
    fmt.Println("Hello")
}
`

	result := RemoveMetadata(code)
	expected := `package requirement

import (
    "fmt"
)

func main() {
    fmt.Println("Hello")
}
`

	if result != expected {
		t.Errorf("RemoveMetadata 结果不匹配\n期望:\n%s\n实际:\n%s", expected, result)
	}
}
