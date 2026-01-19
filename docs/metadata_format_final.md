# Metadata 格式最终方案

## 格式规范

使用 Go 多行注释 `/* */` 包裹 metadata，格式：`/* @key value */`

```go
/* @file requirement.go */
/* @directory_name 需求管理系统 */
/* @directory_code requirement */
/* @directory_desc 主要是进行需求管理，包括：
   - 需求的创建
   - 需求的更新
   - 需求的查询
   - 需求的状态流转 */
/* @tags 需求管理,项目管理 */
/* @version v1.0.0 */
package requirement
```

## 解析规则

1. **单行值**：`/* @key value */` → `key: "value"`
2. **多行值**：`/* @key value
   多行内容
   可以跨多行 */` → `key: "value\n多行内容\n可以跨多行"`
3. **数组值**：`/* @tags 需求管理,项目管理 */` → `tags: ["需求管理", "项目管理"]`

## 优点

1. ✅ **简单直观**：使用 Go 标准多行注释，格式清晰
2. ✅ **天然支持多行**：多行注释天然支持多行内容，无需特殊处理
3. ✅ **解析简单**：找到 `/* @` 和 `*/` 之间的内容即可
4. ✅ **不依赖缩进**：不需要严格的缩进规则
5. ✅ **LLM 友好**：格式清晰，LLM 容易生成
6. ✅ **人类友好**：编辑时不容易出错
7. ✅ **无依赖**：不需要引入额外库

## 解析实现

```go
package metadata

import (
    "strings"
    "regexp"
)

// Metadata 代码元数据
type Metadata struct {
    File          string   `json:"file"`
    DirectoryName string   `json:"directory_name"`
    DirectoryCode string   `json:"directory_code"`
    DirectoryDesc string   `json:"directory_desc"`
    Tags          []string `json:"tags,omitempty"`
    Version       string   `json:"version,omitempty"`
}

// ParseMetadata 从代码中解析 metadata
func ParseMetadata(code string) (*Metadata, error) {
    metadata := &Metadata{}
    lines := strings.Split(code, "\n")
    
    i := 0
    for i < len(lines) {
        line := strings.TrimSpace(lines[i])
        
        // 只处理 // @ 开头的行
        if !strings.HasPrefix(line, "// @") {
            i++
            continue
        }

        // 去掉 "// @" 前缀，解析键值对
        content := strings.TrimSpace(line[4:])
        parts := strings.Fields(content)
        if len(parts) == 0 {
            i++
            continue
        }

        key := parts[0]
        value := strings.Join(parts[1:], " ")

        // 收集多行值：下一行如果是 // 开头但不是 // @，继续收集
        i++
        for i < len(lines) {
            nextLine := strings.TrimSpace(lines[i])
            
            // 遇到新的 @ 标记，结束当前值
            if strings.HasPrefix(nextLine, "// @") {
                break
            }
            
            // 如果是普通注释，继续收集（去掉 "// " 前缀）
            if strings.HasPrefix(nextLine, "//") {
                commentContent := strings.TrimSpace(nextLine[2:])
                if commentContent != "" {
                    if value != "" {
                        value += "\n" + commentContent
                    } else {
                        value = commentContent
                    }
                }
                i++
                continue
            }
            
            // 非注释行，结束收集
            break
        }

        // 设置字段值
        setMetadataField(metadata, key, value)
    }

    return metadata, nil
}

// setMetadataField 设置 metadata 字段
func setMetadataField(metadata *Metadata, key, value string) {
    switch key {
    case "file":
        metadata.File = value
    case "directory_name":
        metadata.DirectoryName = value
    case "directory_code":
        metadata.DirectoryCode = value
    case "directory_desc":
        metadata.DirectoryDesc = value
    case "tags":
        // 逗号分隔的数组
        if value != "" {
            tags := strings.Split(value, ",")
            for i, tag := range tags {
                tags[i] = strings.TrimSpace(tag)
            }
            metadata.Tags = tags
        }
    case "version":
        metadata.Version = value
    }
}

// RemoveMetadata 从代码中移除 metadata（用于保存到文件）
func RemoveMetadata(code string) string {
    lines := strings.Split(code, "\n")
    result := make([]string, 0, len(lines))
    
    i := 0
    for i < len(lines) {
        line := lines[i]
        trimmed := strings.TrimSpace(line)
        
        // 跳过 // @ 开头的行及其多行值
        if strings.HasPrefix(trimmed, "// @") {
            i++
            // 跳过后续的注释行（多行值），直到遇到新的 @ 或非注释行
            for i < len(lines) {
                nextLine := strings.TrimSpace(lines[i])
                if strings.HasPrefix(nextLine, "// @") || !strings.HasPrefix(nextLine, "//") {
                    break
                }
                i++
            }
            continue
        }
        
        result = append(result, line)
        i++
    }
    
    return strings.Join(result, "\n")
}
```

## 示例

### 单行值
```go
// @file requirement.go
// @directory_name 需求管理系统
// @directory_code requirement
package requirement
```

### 多行值
```go
// @directory_desc 主要是进行需求管理，包括：
//    - 需求的创建
//    - 需求的更新
//    - 需求的查询
//    - 需求的状态流转
package requirement
```

### 数组值
```go
// @tags 需求管理,项目管理,工作流
package requirement
```

### 完整示例
```go
// @file requirement.go
// @directory_name 需求管理系统
// @directory_code requirement
// @directory_desc 主要是进行需求管理，包括：
//    - 需求的创建
//    - 需求的更新
//    - 需求的查询
//    - 需求的状态流转
// @tags 需求管理,项目管理
// @version v1.0.0
package requirement

import (
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
)

// Requirement 需求结构体
type Requirement struct {
    // ...
}
```

## 优点

1. ✅ **简单直观**：每行以 `// @` 开头，清晰明了
2. ✅ **支持多行**：值可以跨多行，自动收集
3. ✅ **不依赖缩进**：不需要严格的缩进规则
4. ✅ **解析简单**：字符串处理即可，不需要复杂解析
5. ✅ **LLM 友好**：格式清晰，LLM 容易生成
6. ✅ **人类友好**：编辑时不容易出错
7. ✅ **无依赖**：不需要引入额外库
8. ✅ **易扩展**：添加新字段只需新增一行
