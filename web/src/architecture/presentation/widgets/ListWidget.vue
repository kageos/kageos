<!--
  ListWidget - 自由输入列表组件
  支持 type:list;item_type:number 和 type:list;item_type:text
-->

<template>
  <div class="list-widget">
    <div v-if="mode === 'edit' || mode === 'search'" class="list-editor">
      <el-input
        v-model="inputText"
        :disabled="disabled"
        :placeholder="placeholder"
        @blur="commitInput"
        @keyup.enter="commitInput"
      />
      <div v-if="errorMessage" class="list-error">{{ errorMessage }}</div>
      <div v-if="currentValues.length > 0" class="list-tags">
        <el-tag
          v-for="value in currentValues"
          :key="String(value)"
          class="list-tag"
          :closable="!disabled"
          @close="removeValue(value)"
        >
          {{ value }}
        </el-tag>
      </div>
    </div>

    <div v-else class="list-display" :class="displayClass">
      <el-tag
        v-for="value in currentValues"
        :key="String(value)"
        class="list-tag"
        :size="mode === 'table-cell' ? 'small' : undefined"
      >
        {{ value }}
      </el-tag>
      <span v-if="currentValues.length === 0" class="empty-text">-</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElInput, ElTag } from 'element-plus'
import type { WidgetComponentEmits, WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { ListWidgetConfig } from '@/core/types/widget-configs'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()
const inputText = ref('')
const errorMessage = ref('')

const config = computed(() => {
  return (props.field.widget?.config || {}) as ListWidgetConfig
})

const itemType = computed<'number' | 'text'>(() => {
  return config.value.item_type === 'number' ? 'number' : 'text'
})

const disabled = computed(() => Boolean(config.value.disabled))

const maxCount = computed(() => {
  const value = Number(config.value.max_count || 0)
  return Number.isFinite(value) && value > 0 ? value : 0
})

const placeholder = computed(() => {
  if (config.value.placeholder) {
    return config.value.placeholder
  }
  if (itemType.value === 'number') {
    return `请输入${props.field.name}，例如 1,2,3`
  }
  return `请输入${props.field.name}，例如 a,b,c`
})

const currentValues = computed(() => normalizeRawValues(props.value?.raw))

const displayClass = computed(() => {
  if (props.mode === 'response') return 'response-list'
  if (props.mode === 'table-cell') return 'table-cell-list'
  return 'detail-list'
})

watch(
  () => props.value?.raw,
  raw => {
    inputText.value = normalizeRawValues(raw).map(String).join(config.value.separator || ',')
  },
  { immediate: true, deep: true }
)

onMounted(() => {
  if (props.mode !== 'edit') {
    return
  }
  if (props.value?.raw !== null && props.value?.raw !== undefined && props.value?.raw !== '') {
    return
  }
  const values = normalizeRawValues(null)
  if (values.length > 0) {
    setValues(values)
  }
})

function normalizeRawValues(raw: any): Array<string | number> {
  if (Array.isArray(raw)) {
    return normalizeParts(raw.map(value => String(value)))
  }
  if (typeof raw === 'string' && raw.trim() !== '') {
    return parseParts(raw).values
  }
  const defaultValue = config.value.render_default
  if (Array.isArray(defaultValue)) {
    return normalizeParts(defaultValue.map(value => String(value)))
  }
  if (typeof defaultValue === 'string' && defaultValue.trim() !== '') {
    return parseParts(defaultValue).values
  }
  return []
}

function parseParts(text: string): { values: Array<string | number>; invalid: string[] } {
  const separator = config.value.separator || ','
  const escapedSeparator = escapeRegExp(separator)
  const pattern = new RegExp(`${escapedSeparator}|[，;；\\n\\r\\t ]+`, 'g')
  const parts = text
    .split(pattern)
    .map(part => part.trim())
    .filter(Boolean)
  return normalizePartsWithInvalid(parts)
}

function normalizeParts(parts: string[]): Array<string | number> {
  return normalizePartsWithInvalid(parts).values
}

function normalizePartsWithInvalid(parts: string[]): { values: Array<string | number>; invalid: string[] } {
  const values: Array<string | number> = []
  const invalid: string[] = []
  const seen = new Set<string>()

  for (const part of parts) {
    const value = itemType.value === 'number' ? Number(part) : part
    if (itemType.value === 'number' && !Number.isFinite(value as number)) {
      invalid.push(part)
      continue
    }

    const key = String(value)
    if (config.value.unique && seen.has(key)) {
      continue
    }
    seen.add(key)
    values.push(value)
  }

  if (maxCount.value > 0 && values.length > maxCount.value) {
    return {
      values: values.slice(0, maxCount.value),
      invalid
    }
  }

  return { values, invalid }
}

function commitInput(): void {
  if (props.mode !== 'edit' && props.mode !== 'search') {
    return
  }
  const parsed = parseParts(inputText.value)
  if (parsed.invalid.length > 0) {
    errorMessage.value = `存在非法数字：${parsed.invalid.join(', ')}`
    return
  }
  errorMessage.value = ''
  setValues(parsed.values)
}

function removeValue(value: string | number): void {
  const nextValues = currentValues.value.filter(item => String(item) !== String(value))
  inputText.value = nextValues.map(String).join(config.value.separator || ',')
  setValues(nextValues)
}

function setValues(values: Array<string | number>): void {
  const display = values.map(String).join(', ')
  const fieldValue = createFieldValue(props.field, values, display)
  formDataStore.setValue(props.fieldPath, fieldValue)
  emit('update:modelValue', fieldValue)
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
</script>

<style scoped>
.list-widget {
  width: 100%;
}

.list-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.list-tags,
.list-display {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.list-tag {
  margin: 0;
}

.list-error {
  color: var(--el-color-danger);
  font-size: 12px;
  line-height: 1.4;
}

.empty-text {
  color: var(--el-text-color-placeholder);
}
</style>
