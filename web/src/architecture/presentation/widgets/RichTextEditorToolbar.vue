<template>
  <div class="editor-toolbar">
    <div class="toolbar-group toolbar-group-preview">
      <el-tooltip :content="isPreviewMode ? '编辑模式' : '预览模式'" placement="bottom">
        <button
          type="button"
          @click="$emit('toggle-preview')"
          class="toolbar-button preview-toggle"
          :class="{ 'is-active': isPreviewMode }"
        >
          <el-icon v-if="!isPreviewMode"><View /></el-icon>
          <el-icon v-else><Edit /></el-icon>
        </button>
      </el-tooltip>
    </div>

    <div class="toolbar-divider"></div>

    <template v-if="!isPreviewMode">
      <div class="toolbar-group">
        <el-tooltip content="粗体" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleBold().run()"
            :class="{ 'is-active': editor.isActive('bold') }"
            class="toolbar-button"
          >
            <strong style="font-size: 14px;">B</strong>
          </button>
        </el-tooltip>
        <el-tooltip content="斜体" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleItalic().run()"
            :class="{ 'is-active': editor.isActive('italic') }"
            class="toolbar-button"
          >
            <em style="font-size: 14px;">I</em>
          </button>
        </el-tooltip>
        <el-tooltip content="删除线" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleStrike().run()"
            :class="{ 'is-active': editor.isActive('strike') }"
            class="toolbar-button"
          >
            <s style="font-size: 14px;">S</s>
          </button>
        </el-tooltip>
        <el-tooltip content="下划线" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleUnderline().run()"
            :class="{ 'is-active': editor.isActive('underline') }"
            class="toolbar-button"
          >
            <u style="font-size: 14px;">U</u>
          </button>
        </el-tooltip>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <el-tooltip content="正文" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().setParagraph().run()"
            :class="{ 'is-active': editor.isActive('paragraph') }"
            class="toolbar-button"
          >
            <el-icon><Document /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip content="标题 1" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleHeading({ level: 1 }).run()"
            :class="{ 'is-active': editor.isActive('heading', { level: 1 }) }"
            class="toolbar-button"
          >
            <span class="heading-text">H1</span>
          </button>
        </el-tooltip>
        <el-tooltip content="标题 2" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleHeading({ level: 2 }).run()"
            :class="{ 'is-active': editor.isActive('heading', { level: 2 }) }"
            class="toolbar-button"
          >
            <span class="heading-text">H2</span>
          </button>
        </el-tooltip>
        <el-tooltip content="标题 3" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleHeading({ level: 3 }).run()"
            :class="{ 'is-active': editor.isActive('heading', { level: 3 }) }"
            class="toolbar-button"
          >
            <span class="heading-text">H3</span>
          </button>
        </el-tooltip>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <el-tooltip content="无序列表" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleBulletList().run()"
            :class="{ 'is-active': editor.isActive('bulletList') }"
            class="toolbar-button"
          >
            <el-icon><List /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip content="有序列表" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleOrderedList().run()"
            :class="{ 'is-active': editor.isActive('orderedList') }"
            class="toolbar-button"
          >
            <el-icon><Sort /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip content="任务列表" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleTaskList().run()"
            :class="{ 'is-active': editor.isActive('taskList') }"
            class="toolbar-button"
          >
            <el-icon><CircleCheck /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip content="引用" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleBlockquote().run()"
            :class="{ 'is-active': editor.isActive('blockquote') }"
            class="toolbar-button"
          >
            <el-icon><ChatLineRound /></el-icon>
          </button>
        </el-tooltip>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <el-tooltip content="左对齐" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().setTextAlign('left').run()"
            :class="{ 'is-active': editor.isActive({ textAlign: 'left' }) }"
            class="toolbar-button"
          >
            <span style="font-size: 14px; font-weight: bold;">◀</span>
          </button>
        </el-tooltip>
        <el-tooltip content="居中" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().setTextAlign('center').run()"
            :class="{ 'is-active': editor.isActive({ textAlign: 'center' }) }"
            class="toolbar-button"
          >
            <span style="font-size: 14px; font-weight: bold;">⬌</span>
          </button>
        </el-tooltip>
        <el-tooltip content="右对齐" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().setTextAlign('right').run()"
            :class="{ 'is-active': editor.isActive({ textAlign: 'right' }) }"
            class="toolbar-button"
          >
            <span style="font-size: 14px; font-weight: bold;">▶</span>
          </button>
        </el-tooltip>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <el-tooltip content="行内代码" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleCode().run()"
            :class="{ 'is-active': editor.isActive('code') }"
            class="toolbar-button"
          >
            <span style="font-size: 12px; font-family: monospace;">&lt;/&gt;</span>
          </button>
        </el-tooltip>
        <el-tooltip content="代码块" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleCodeBlock().run()"
            :class="{ 'is-active': editor.isActive('codeBlock') }"
            class="toolbar-button"
          >
            <el-icon><Operation /></el-icon>
          </button>
        </el-tooltip>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <el-tooltip content="文字颜色" placement="bottom">
          <div class="color-picker-wrapper">
            <input
              type="color"
              :value="textColor"
              @input="$emit('text-color-change', ($event.target as HTMLInputElement).value)"
              class="color-picker-input"
            />
            <button
              type="button"
              class="toolbar-button color-picker-button"
              :style="{ color: textColor }"
            >
              A
            </button>
          </div>
        </el-tooltip>
        <el-tooltip content="背景高亮" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().toggleHighlight().run()"
            :class="{ 'is-active': editor.isActive('highlight') }"
            class="toolbar-button"
          >
            <span style="background-color: yellow; padding: 2px 4px; border-radius: 2px;">高</span>
          </button>
        </el-tooltip>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <el-tooltip content="链接" placement="bottom">
          <button
            type="button"
            @click="$emit('set-link')"
            :class="{ 'is-active': editor.isActive('link') }"
            class="toolbar-button"
          >
            <el-icon><LinkIcon /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip content="表格" placement="bottom">
          <el-dropdown trigger="click" placement="bottom-start" @command="$emit('table-command', String($event))">
            <button
              type="button"
              :class="{ 'is-active': editor.isActive('table') }"
              class="toolbar-button"
            >
              <el-icon><Grid /></el-icon>
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="insert">
                  <el-icon style="margin-right: 8px;"><Plus /></el-icon>
                  插入表格 (3x3)
                </el-dropdown-item>
                <el-dropdown-item command="addColumnBefore" :disabled="!editor.isActive('table')" divided>
                  <el-icon style="margin-right: 8px;"><Plus /></el-icon>
                  左侧插入列
                </el-dropdown-item>
                <el-dropdown-item command="addColumnAfter" :disabled="!editor.isActive('table')">
                  <el-icon style="margin-right: 8px;"><Plus /></el-icon>
                  右侧插入列
                </el-dropdown-item>
                <el-dropdown-item command="deleteColumn" :disabled="!editor.isActive('table')">
                  <el-icon style="margin-right: 8px;"><Remove /></el-icon>
                  删除当前列
                </el-dropdown-item>
                <el-dropdown-item command="addRowBefore" :disabled="!editor.isActive('table')" divided>
                  <el-icon style="margin-right: 8px;"><Plus /></el-icon>
                  上方插入行
                </el-dropdown-item>
                <el-dropdown-item command="addRowAfter" :disabled="!editor.isActive('table')">
                  <el-icon style="margin-right: 8px;"><Plus /></el-icon>
                  下方插入行
                </el-dropdown-item>
                <el-dropdown-item command="deleteRow" :disabled="!editor.isActive('table')">
                  <el-icon style="margin-right: 8px;"><Remove /></el-icon>
                  删除当前行
                </el-dropdown-item>
                <el-dropdown-item command="deleteTable" :disabled="!editor.isActive('table')" divided>
                  <el-icon style="margin-right: 8px;"><Delete /></el-icon>
                  删除表格
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </el-tooltip>
        <el-tooltip content="分隔线" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().setHorizontalRule().run()"
            class="toolbar-button"
          >
            <el-icon><Minus /></el-icon>
          </button>
        </el-tooltip>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <el-tooltip content="清除格式" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().clearNodes().unsetAllMarks().run()"
            class="toolbar-button"
          >
            <el-icon><Delete /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip content="撤销" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().undo().run()"
            :disabled="!editor.can().undo()"
            class="toolbar-button"
          >
            <el-icon><RefreshLeft /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip content="重做" placement="bottom">
          <button
            type="button"
            @click="editor.chain().focus().redo().run()"
            :disabled="!editor.can().redo()"
            class="toolbar-button"
          >
            <el-icon><RefreshRight /></el-icon>
          </button>
        </el-tooltip>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import type { Editor } from '@tiptap/vue-3'
import {
  Document,
  List,
  Sort,
  ChatLineRound,
  Link as LinkIcon,
  Grid,
  Minus,
  RefreshLeft,
  RefreshRight,
  Operation,
  Delete,
  CircleCheck,
  Plus,
  Remove,
  View,
  Edit,
} from '@element-plus/icons-vue'

defineProps<{
  editor: Editor
  isPreviewMode: boolean
  textColor: string
}>()

defineEmits<{
  (e: 'toggle-preview'): void
  (e: 'set-link'): void
  (e: 'table-command', command: string): void
  (e: 'text-color-change', color: string): void
}>()
</script>

<style scoped>
.editor-toolbar {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 8px 12px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-light);
  flex-wrap: wrap;
}

:global(.form-view-flat) .editor-toolbar {
  padding: 10px 12px;
  background: transparent;
  border-bottom: 1px solid var(--app-auth-input-border);
}

.toolbar-group {
  display: flex;
  align-items: center;
  gap: 2px;
}

.toolbar-group-preview {
  margin-right: auto;
}

.toolbar-divider {
  width: 1px;
  height: 20px;
  background-color: var(--border-light);
  margin: 0 10px;
}

:global(.form-view-flat) .toolbar-divider {
  background-color: rgba(203, 213, 225, 0.7);
}

.toolbar-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background-color: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 16px;
  transition: all 0.2s cubic-bezier(0.25, 0.8, 0.25, 1);
}

:global(.form-view-flat) .toolbar-button {
  border-radius: 8px;
}

.toolbar-button:hover:not(:disabled) {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.toolbar-button.is-active {
  background-color: var(--color-primary-light-9);
  color: var(--color-primary);
}

.toolbar-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.heading-text {
  font-size: 12px;
  font-weight: bold;
}

.color-picker-wrapper {
  position: relative;
  display: inline-block;
}

.color-picker-input {
  position: absolute;
  width: 32px;
  height: 32px;
  opacity: 0;
  cursor: pointer;
}

.color-picker-button {
  position: relative;
  font-weight: bold;
  font-size: 16px;
}

.preview-toggle {
  margin-right: 8px;
}
</style>
