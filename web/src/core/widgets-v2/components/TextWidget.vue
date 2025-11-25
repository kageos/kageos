<!--
  TextWidget - 文本显示组件
  用于输出参数，支持多种格式化显示（json、yaml、xml、markdown、html、csv等）
-->

<template>
  <div class="text-widget">
    <!-- 响应模式（主要使用场景） -->
    <div v-if="mode === 'response'" class="response-text">
      <div v-if="formattedContent" class="formatted-content" :class="formatClass">
        <pre v-if="isCodeFormat" class="code-content">{{ formattedContent }}</pre>
        <div v-else-if="format === 'html'" v-html="formattedContent" class="html-content"></div>
        <div v-else class="text-content">{{ formattedContent }}</div>
      </div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 表格单元格模式 -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-text">
      <div v-if="formattedContent" class="formatted-content" :class="formatClass">
        <pre v-if="isCodeFormat" class="code-content">{{ formattedContent }}</pre>
        <div v-else-if="format === 'html'" v-html="formattedContent" class="html-content"></div>
        <div v-else class="text-content">{{ formattedContent }}</div>
      </div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-text">
      <div class="detail-content">
        <div v-if="formattedContent" class="formatted-content" :class="formatClass">
          <pre v-if="isCodeFormat" class="code-content">{{ formattedContent }}</pre>
          <div v-else-if="format === 'html'" v-html="formattedContent" class="html-content"></div>
          <div v-else class="text-content">{{ formattedContent }}</div>
        </div>
        <span v-else class="empty-text">-</span>
      </div>
    </div>
    
    <!-- 编辑模式（通常不使用，但保留兼容性） -->
    <div v-else-if="mode === 'edit'" class="edit-text">
      <el-input
        v-model="internalValue"
        type="textarea"
        :rows="6"
        :placeholder="field.desc || `请输入${field.name}`"
        :disabled="field.widget?.config?.disabled"
        @blur="handleBlur"
      />
    </div>
    
    <!-- 其他模式（默认显示） -->
    <div v-else class="default-text">
      <div v-if="formattedContent" class="formatted-content" :class="formatClass">
        <pre v-if="isCodeFormat" class="code-content">{{ formattedContent }}</pre>
        <div v-else-if="format === 'html'" v-html="formattedContent" class="html-content"></div>
        <div v-else class="text-content">{{ formattedContent }}</div>
      </div>
      <span v-else class="empty-text">-</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElInput } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 配置
const config = computed(() => props.field.widget?.config || {})

// 格式化类型
const format = computed(() => {
  return (config.value.format || '').toLowerCase()
})

// 是否为代码格式（需要代码高亮）
const isCodeFormat = computed(() => {
  return ['json', 'yaml', 'xml', 'csv', 'javascript', 'typescript', 'python', 'java', 'go', 'sql'].includes(format.value)
})

// 格式类名（用于样式）
const formatClass = computed(() => {
  return `format-${format.value || 'text'}`
})

// 原始内容
const rawContent = computed(() => {
  const value = props.value
  if (!value) {
    return ''
  }
  
  // 优先使用 display，如果没有则使用 raw
  if (value.display) {
    return String(value.display)
  }
  
  const raw = value.raw
  if (raw === null || raw === undefined || raw === '') {
    return ''
  }
  
  return String(raw)
})

// 格式化后的内容
const formattedContent = computed(() => {
  const content = rawContent.value
  if (!content) {
    return ''
  }
  
  const fmt = format.value
  
  // JSON 格式化
  if (fmt === 'json') {
    try {
      const parsed = typeof content === 'string' ? JSON.parse(content) : content
      return JSON.stringify(parsed, null, 2)
    } catch {
      return content
    }
  }
  
  // YAML 格式化（简单处理，实际可以使用 yaml 库）
  if (fmt === 'yaml') {
    return content
  }
  
  // XML 格式化（简单处理，实际可以使用 xml 格式化库）
  if (fmt === 'xml') {
    return content
  }
  
  // Markdown 和 HTML 直接返回（可以后续集成 markdown 渲染器）
  if (fmt === 'markdown' || fmt === 'html') {
    return content
  }
  
  // CSV 格式化（简单处理）
  if (fmt === 'csv') {
    return content
  }
  
  // 其他格式直接返回
  return content
})

// 内部值（用于编辑模式）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit') {
      return rawContent.value
    }
    return ''
  },
  set: (newValue: string) => {
    if (props.mode === 'edit') {
      const fieldValue = {
        raw: newValue,
        display: newValue,
        meta: {}
      }
      
      formDataStore.setValue(props.fieldPath, fieldValue)
      emit('update:modelValue', fieldValue)
    }
  }
})

// 失焦处理
function handleBlur(): void {
  // 可以在这里添加验证逻辑
}
</script>

<style scoped>
.text-widget {
  width: 100%;
}

.response-text,
.table-cell-text,
.default-text {
  width: 100%;
}

.formatted-content {
  width: 100%;
  overflow: auto;
}

.code-content {
  margin: 0;
  padding: 12px;
  background-color: var(--el-fill-color-light);
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 13px;
  line-height: 1.5;
  overflow-x: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.html-content {
  padding: 8px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  background-color: var(--el-bg-color);
}

.text-content {
  padding: 8px;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.empty-text {
  color: var(--el-text-color-placeholder);
}

.detail-text {
  margin-bottom: 16px;
}

.detail-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}

.detail-content {
  color: var(--el-text-color-regular);
}

.edit-text {
  width: 100%;
}

/* 不同格式的样式 */
.format-json .code-content {
  color: #2c3e50;
}

.format-yaml .code-content {
  color: #2c3e50;
}

.format-xml .code-content {
  color: #2c3e50;
}

.format-csv .code-content {
  color: #2c3e50;
}
</style>

