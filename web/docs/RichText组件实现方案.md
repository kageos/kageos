# RichText 富文本组件实现方案

## 一、技术选型

### 推荐方案：TipTap
**理由**：
1. ✅ 专为 Vue 3 设计，API 现代化
2. ✅ 基于 ProseMirror，性能优秀
3. ✅ 模块化架构，易于扩展
4. ✅ 支持实时协作（未来可扩展）
5. ✅ 轻量级，按需加载

**备选方案**：
- Quill：轻量级，但 Vue 3 支持一般
- TinyMCE：功能强大但体积大
- WangEditor：中文友好但 Vue 3 支持一般

## 二、后端设计

### 2.1 结构体定义

```go
type RichText struct {
    // 可选参数
    Height int    `json:"height"`    // 编辑器高度（默认300px）
    Placeholder string `json:"placeholder"` // 占位符文本
    Toolbar string `json:"toolbar"`   // 工具栏配置（full/minimal/custom）
    Default string `json:"default"`   // 默认内容（HTML格式）
}
```

### 2.2 配置参数

- `height`: 编辑器高度（单位：px，默认 300）
- `placeholder`: 占位符文本（默认："请输入内容..."）
- `toolbar`: 工具栏配置
  - `full`: 完整工具栏（默认）
  - `minimal`: 精简工具栏（仅基础格式）
  - `custom`: 自定义工具栏（通过配置字符串）
- `default`: 默认内容（HTML 格式）

## 三、前端设计

### 3.1 组件结构

```vue
<template>
  <div class="rich-text-widget">
    <!-- 编辑模式：TipTap 编辑器 -->
    <div v-if="mode === 'edit'" class="editor-container">
      <editor-content :editor="editor" />
    </div>
    
    <!-- 响应/表格/详情模式：HTML 渲染 -->
    <div v-else class="html-content" v-html="htmlContent"></div>
    
    <!-- 搜索模式：文本输入 -->
    <el-input v-else-if="mode === 'search'" ... />
  </div>
</template>
```

### 3.2 功能实现

1. **编辑模式**：
   - 使用 TipTap Editor
   - 支持工具栏配置
   - 实时保存 HTML 内容

2. **显示模式**：
   - 渲染 HTML 内容
   - 使用 `v-html` 或专门的 HTML 渲染组件
   - 注意 XSS 安全（后端已处理）

3. **搜索模式**：
   - 文本输入框
   - 支持 HTML 标签过滤搜索

## 四、实现步骤

1. **安装依赖**：
   ```bash
   npm install @tiptap/vue-3 @tiptap/starter-kit @tiptap/extension-link @tiptap/extension-image
   ```

2. **后端实现**：
   - 创建 `sdk/agent-app/widget/richtext.go`
   - 注册到 `widget.go`

3. **前端实现**：
   - 创建 `web/src/core/widgets-v2/components/RichTextWidget.vue`
   - 注册到 `factories-v2/index.ts`
   - 更新 `constants/widget.ts`

4. **搜索配置**：
   - 更新 `searchComponentConfig.ts`

## 五、注意事项

1. **XSS 安全**：后端需要清理 HTML 内容
2. **性能优化**：大量富文本内容时考虑懒加载
3. **移动端适配**：TipTap 在移动端表现良好
4. **内容存储**：存储为 HTML 字符串

