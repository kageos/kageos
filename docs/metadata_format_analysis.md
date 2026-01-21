# Metadata 格式分析与建议

## 格式对比

### 1. XML 风格注释（文档中的格式）

```go
//<文件名>requirement.go</文件名>
//<目录名称>需求管理系统</目录名称>
//<目录标识>requirement</目录标识>
//<目录介绍>主要是进行需求管理，包括需求的创建、更新、查询等功能</目录介绍>
package requirement
```

**优点：**
- ✅ 清晰直观，类似 HTML/XML
- ✅ 人类可读性好
- ✅ LLM 容易理解格式

**缺点：**
- ❌ 解析需要正则表达式，相对复杂
- ❌ 扩展性一般（添加新字段需要修改正则）
- ❌ 多行值处理麻烦（如目录介绍很长）

**解析示例：**
```go
re := regexp.MustCompile(`//<(\w+)>(.*?)</\1>`)
matches := re.FindAllStringSubmatch(code, -1)
```

---

### 2. YAML Front Matter（推荐 ⭐）

```go
// ---
// file: requirement.go
// directory_name: 需求管理系统
// directory_code: requirement
// directory_desc: 主要是进行需求管理，包括需求的创建、更新、查询等功能
// ---
package requirement
```

**优点：**
- ✅ **结构化**：标准 YAML 格式，有成熟的解析库
- ✅ **易扩展**：添加新字段只需新增一行
- ✅ **多行值支持**：YAML 支持多行字符串（`|` 或 `>`）
- ✅ **LLM 友好**：格式清晰，LLM 容易生成
- ✅ **可读性好**：人类也容易阅读和编辑
- ✅ **类型支持**：YAML 支持字符串、数字、布尔、数组等

**缺点：**
- ⚠️ 需要引入 YAML 解析库（但 Go 有 `gopkg.in/yaml.v3`，很成熟）

**解析示例：**
```go
import "gopkg.in/yaml.v3"

// 提取 YAML 块
yamlBlock := extractYAMLFrontMatter(code)
var metadata map[string]interface{}
yaml.Unmarshal([]byte(yamlBlock), &metadata)
```

---

### 3. 键值对注释（类似 JSDoc）

```go
// @file requirement.go
// @directory_name 需求管理系统
// @directory_code requirement
// @directory_desc 主要是进行需求管理，包括需求的创建、更新、查询等功能
package requirement
```

**优点：**
- ✅ 简单直观
- ✅ 不需要额外库（字符串解析即可）
- ✅ LLM 容易生成

**缺点：**
- ❌ 多行值处理麻烦（需要特殊标记）
- ❌ 类型推断困难（都是字符串）
- ❌ 扩展性一般

**解析示例：**
```go
lines := strings.Split(code, "\n")
for _, line := range lines {
    if strings.HasPrefix(line, "// @") {
        parts := strings.Fields(line[4:]) // 去掉 "// @"
        key := parts[0]
        value := strings.Join(parts[1:], " ")
    }
}
```

---

### 4. JSON 单行注释

```go
// {"file":"requirement.go","directory_name":"需求管理系统","directory_code":"requirement","directory_desc":"..."}
package requirement
```

**优点：**
- ✅ 结构化，Go 有原生 JSON 解析
- ✅ 类型支持完整

**缺点：**
- ❌ **可读性差**：单行 JSON 在注释中不友好
- ❌ 多行 JSON 在注释中更难看
- ❌ LLM 生成容易出错（引号转义等）

---

### 5. 结构化注释块

```go
/*
 * Metadata:
 *   File: requirement.go
 *   DirectoryName: 需求管理系统
 *   DirectoryCode: requirement
 *   DirectoryDesc: 主要是进行需求管理...
 */
package requirement
```

**优点：**
- ✅ 可读性好
- ✅ 支持多行

**缺点：**
- ❌ 解析相对复杂（需要处理多行和缩进）
- ❌ 格式不够标准化

---

## 推荐方案：YAML Front Matter

### 为什么选择 YAML Front Matter？

1. **标准化**：YAML 是广泛使用的配置格式，有成熟的工具和库
2. **扩展性强**：未来添加新字段（如 `tags`、`version`、`author` 等）非常容易
3. **类型支持**：支持字符串、数字、布尔、数组、对象等
4. **多行值**：支持多行字符串，适合长描述
5. **LLM 友好**：格式清晰，LLM 容易生成正确的格式
6. **可读性好**：人类也容易阅读和编辑

### 完整示例

```go
// ---
// file: requirement.go
// directory_name: 需求管理系统
// directory_code: requirement
// directory_desc: |
//   主要是进行需求管理，包括：
//   - 需求的创建
//   - 需求的更新
//   - 需求的查询
//   - 需求的状态流转
// tags:
//   - 需求管理
//   - 项目管理
// version: v1.0.0
// ---
package requirement

import (
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// Requirement 需求结构体
type Requirement struct {
    // ...
}
```

### 解析实现

```go
package metadata

import (
    "regexp"
    "strings"
    "gopkg.in/yaml.v3"
)

// Metadata 代码元数据
type Metadata struct {
    File          string   `yaml:"file"`
    DirectoryName string   `yaml:"directory_name"`
    DirectoryCode string   `yaml:"directory_code"`
    DirectoryDesc string   `yaml:"directory_desc"`
    Tags          []string `yaml:"tags,omitempty"`
    Version       string   `yaml:"version,omitempty"`
    // 未来可以扩展更多字段
}

// ParseMetadata 从代码中解析 metadata
func ParseMetadata(code string) (*Metadata, error) {
    // 1. 提取 YAML Front Matter
    yamlBlock := extractYAMLFrontMatter(code)
    if yamlBlock == "" {
        return nil, nil // 没有 metadata，返回 nil
    }

    // 2. 解析 YAML
    var metadata Metadata
    if err := yaml.Unmarshal([]byte(yamlBlock), &metadata); err != nil {
        return nil, fmt.Errorf("解析 metadata 失败: %w", err)
    }

    return &metadata, nil
}

// extractYAMLFrontMatter 提取 YAML Front Matter
func extractYAMLFrontMatter(code string) string {
    // 匹配模式：// --- 到 // --- 之间的内容
    re := regexp.MustCompile(`(?s)^\s*//\s*---\s*\n(.*?)\n\s*//\s*---\s*\n`)
    matches := re.FindStringSubmatch(code)
    if len(matches) < 2 {
        return ""
    }

    // 去掉每行开头的 "// "
    yamlLines := strings.Split(matches[1], "\n")
    result := make([]string, 0, len(yamlLines))
    for _, line := range yamlLines {
        trimmed := strings.TrimSpace(line)
        if strings.HasPrefix(trimmed, "//") {
            trimmed = strings.TrimSpace(trimmed[2:])
        }
        if trimmed != "" {
            result = append(result, trimmed)
        }
    }

    return strings.Join(result, "\n")
}

// RemoveMetadata 从代码中移除 metadata（用于保存到文件）
func RemoveMetadata(code string) string {
    re := regexp.MustCompile(`(?s)^\s*//\s*---\s*\n.*?\n\s*//\s*---\s*\n\s*`)
    return re.ReplaceAllString(code, "")
}
```

### 扩展性示例

未来如果需要添加新字段，只需：

```go
// ---
// file: requirement.go
// directory_name: 需求管理系统
// directory_code: requirement
// directory_desc: 主要是进行需求管理...
// tags:
//   - 需求管理
//   - 项目管理
// version: v1.0.0
// author: beiluo                    # 新增：作者
// created_at: 2024-01-19            # 新增：创建时间
// dependencies:                      # 新增：依赖
//   - ticket
//   - user
// ---
package requirement
```

只需在 `Metadata` 结构体中添加字段即可，解析器自动支持。

---

## 备选方案：简化版键值对（如果不想引入 YAML 库）

如果不想引入 YAML 库，可以使用简化版的键值对格式：

```go
// @metadata
// @file requirement.go
// @directory_name 需求管理系统
// @directory_code requirement
// @directory_desc 主要是进行需求管理，包括需求的创建、更新、查询等功能
// @end_metadata
package requirement
```

**优点：**
- ✅ 不需要额外库
- ✅ 解析简单
- ✅ LLM 容易生成

**缺点：**
- ❌ 多行值需要特殊处理（如使用 `@directory_desc_line1`、`@directory_desc_line2`）
- ❌ 类型都是字符串，需要手动转换

---

## 最终推荐

**首选：YAML Front Matter**
- 最标准化、最易扩展、最易维护
- 虽然需要引入 YAML 库，但这是值得的投资

**备选：简化版键值对**
- 如果项目不想引入额外依赖，可以使用这个方案
- 但需要自己处理多行值和类型转换
