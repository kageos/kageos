# Metadata 格式最终推荐（考虑注释中的缩进问题）

## 问题分析

YAML 在注释中的问题：
1. **缩进敏感**：YAML 依赖缩进，但在注释中每行都有 `// ` 前缀
2. **解析复杂**：需要去掉 `// ` 前缀，但可能破坏缩进关系
3. **LLM 易错**：LLM 生成时可能缩进错误
4. **人类易错**：编辑时也容易出错

## 推荐方案：键值对格式（简化版，不依赖缩进）

### 格式示例

```go
// @metadata
// @file requirement.go
// @directory_name 需求管理系统
// @directory_code requirement
// @directory_desc 主要是进行需求管理，包括需求的创建、更新、查询等功能
// @tags 需求管理,项目管理
// @version v1.0.0
// @end_metadata
package requirement
```

**优点：**
- ✅ **不依赖缩进**：每行独立，格式简单
- ✅ **解析简单**：字符串分割即可，不需要复杂解析
- ✅ **LLM 友好**：格式清晰，LLM 不容易出错
- ✅ **人类友好**：编辑时不容易出错
- ✅ **无依赖**：不需要引入 YAML 库

**缺点：**
- ⚠️ 多行值需要特殊处理（可以用 `\n` 转义或特殊标记）
- ⚠️ 类型都是字符串，需要手动转换（但通常 metadata 都是字符串）

### 多行值处理方案

**方案 1：使用 `\n` 转义**
```go
// @directory_desc 主要是进行需求管理，\n包括：\n- 需求的创建\n- 需求的更新
```

**方案 2：使用特殊标记（推荐）**
```go
// @directory_desc_start
// 主要是进行需求管理，包括：
// - 需求的创建
// - 需求的更新
// - 需求的查询
// @directory_desc_end
```

**方案 3：使用 JSON 字符串（推荐）**
```go
// @directory_desc {"type":"text","content":"主要是进行需求管理，包括：\n- 需求的创建\n- 需求的更新"}
```

---

## 备选方案：JSON 单行（如果值不长）

```go
// @metadata {"file":"requirement.go","directory_name":"需求管理系统","directory_code":"requirement","directory_desc":"主要是进行需求管理..."}
package requirement
```

**优点：**
- ✅ 格式严格，不容易出错
- ✅ Go 有原生 JSON 解析
- ✅ 支持复杂类型（对象、数组）

**缺点：**
- ❌ 可读性差（单行 JSON）
- ❌ 多行值需要转义，可读性更差
- ❌ LLM 生成时可能引号转义出错

---

## 最终推荐：键值对格式（带多行值支持）

### 完整格式规范

```go
// @metadata
// @file requirement.go
// @directory_name 需求管理系统
// @directory_code requirement
// @directory_desc_start
// 主要是进行需求管理，包括：
// - 需求的创建
// - 需求的更新
// - 需求的查询
// - 需求的状态流转
// @directory_desc_end
// @tags 需求管理,项目管理
// @version v1.0.0
// @end_metadata
package requirement
```

### 解析规则

1. **单行值**：`@key value` → `key: "value"`
2. **多行值**：
   - `@key_start` 到 `@key_end` 之间的内容（去掉 `// ` 前缀）
   - 或使用 `@key` 后跟多行（直到下一个 `@` 开头）
3. **数组值**：`@tags 需求管理,项目管理` → `tags: ["需求管理", "项目管理"]`
4. **结束标记**：`@end_metadata` 表示 metadata 结束

### 解析实现示例

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
    // 1. 提取 metadata 块
    metadataBlock := extractMetadataBlock(code)
    if metadataBlock == "" {
        return nil, nil
    }

    // 2. 解析键值对
    metadata := &Metadata{}
    lines := strings.Split(metadataBlock, "\n")
    
    i := 0
    for i < len(lines) {
        line := strings.TrimSpace(lines[i])
        if line == "" || !strings.HasPrefix(line, "//") {
            i++
            continue
        }

        // 去掉 "// " 前缀
        content := strings.TrimSpace(line[2:])
        
        // 检查是否是开始标记
        if strings.HasPrefix(content, "@metadata") {
            i++
            continue
        }
        if strings.HasPrefix(content, "@end_metadata") {
            break
        }

        // 解析键值对
        if strings.HasPrefix(content, "@") {
            parts := strings.Fields(content)
            if len(parts) < 2 {
                i++
                continue
            }

            key := strings.TrimPrefix(parts[0], "@")
            value := strings.Join(parts[1:], " ")

            // 检查是否是多行值的开始
            if strings.HasSuffix(key, "_start") {
                baseKey := strings.TrimSuffix(key, "_start")
                // 收集多行值直到找到对应的 _end
                multiLineValue := collectMultiLineValue(lines, i+1, baseKey+"_end")
                setMetadataField(metadata, baseKey, multiLineValue)
                // 跳过已处理的行
                i = findEndMarker(lines, i+1, baseKey+"_end") + 1
                continue
            }

            // 单行值
            setMetadataField(metadata, key, value)
        }

        i++
    }

    return metadata, nil
}

// extractMetadataBlock 提取 metadata 块
func extractMetadataBlock(code string) string {
    re := regexp.MustCompile(`(?s)//\s*@metadata\s*\n(.*?)\n\s*//\s*@end_metadata`)
    matches := re.FindStringSubmatch(code)
    if len(matches) < 2 {
        return ""
    }
    return matches[1]
}

// collectMultiLineValue 收集多行值
func collectMultiLineValue(lines []string, start int, endMarker string) string {
    var result []string
    for i := start; i < len(lines); i++ {
        line := strings.TrimSpace(lines[i])
        if strings.HasPrefix(line, "//") {
            content := strings.TrimSpace(line[2:])
            if strings.HasPrefix(content, "@"+endMarker) {
                break
            }
            // 去掉 "// " 前缀，保留内容
            result = append(result, strings.TrimSpace(line[2:]))
        } else {
            result = append(result, line)
        }
    }
    return strings.Join(result, "\n")
}

// findEndMarker 查找结束标记
func findEndMarker(lines []string, start int, endMarker string) int {
    for i := start; i < len(lines); i++ {
        line := strings.TrimSpace(lines[i])
        if strings.HasPrefix(line, "//") {
            content := strings.TrimSpace(line[2:])
            if strings.HasPrefix(content, "@"+endMarker) {
                return i
            }
        }
    }
    return len(lines) - 1
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

// RemoveMetadata 从代码中移除 metadata
func RemoveMetadata(code string) string {
    re := regexp.MustCompile(`(?s)\s*//\s*@metadata\s*\n.*?\n\s*//\s*@end_metadata\s*\n\s*`)
    return re.ReplaceAllString(code, "")
}
```

---

## 更简化的方案（如果多行值不常用）

如果多行值不常用，可以使用最简单的格式：

```go
// @metadata file=requirement.go directory_name=需求管理系统 directory_code=requirement directory_desc=主要是进行需求管理
package requirement
```

或者：

```go
// @file requirement.go
// @directory_name 需求管理系统
// @directory_code requirement
// @directory_desc 主要是进行需求管理，包括需求的创建、更新、查询等功能
package requirement
```

**解析更简单：**
```go
lines := strings.Split(code, "\n")
for _, line := range lines {
    if strings.HasPrefix(line, "// @") {
        parts := strings.Fields(line[4:]) // 去掉 "// @"
        if len(parts) >= 2 {
            key := parts[0]
            value := strings.Join(parts[1:], " ")
            // 设置到 metadata
        }
    }
}
```

---

## 最终建议

**推荐使用：键值对格式（简化版）**

```go
// @metadata
// @file requirement.go
// @directory_name 需求管理系统
// @directory_code requirement
// @directory_desc 主要是进行需求管理，包括需求的创建、更新、查询等功能
// @tags 需求管理,项目管理
// @version v1.0.0
// @end_metadata
package requirement
```

**理由：**
1. ✅ **不依赖缩进**：每行独立，格式简单
2. ✅ **解析简单**：字符串分割即可
3. ✅ **LLM 友好**：格式清晰，不容易出错
4. ✅ **人类友好**：编辑时不容易出错
5. ✅ **无依赖**：不需要引入额外库
6. ✅ **易扩展**：添加新字段只需新增一行

**多行值处理：**
- 如果目录描述很长，可以使用 `\n` 转义
- 或者使用 `@directory_desc_start` / `@directory_desc_end` 标记
- 或者限制描述长度，要求简洁
