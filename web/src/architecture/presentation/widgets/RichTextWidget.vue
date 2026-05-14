<template>
  <div class="rich-text-widget">
    <RichTextEditorWidget
      v-if="mode === 'edit'"
      v-bind="props"
      @update:modelValue="handleModelValueUpdate"
      @statistics:updated="handleStatisticsUpdated"
      @drawer:change="handleDrawerChange"
    />

    <RichTextResponseWidget
      v-else-if="mode === 'response' || mode === 'detail'"
      v-bind="props"
    />

    <div v-else-if="mode === 'table-cell'" class="table-cell-value">
      <div v-if="previewText" class="html-content-preview">{{ previewText }}</div>
      <span v-else class="empty-text">-</span>
    </div>

    <el-input
      v-else-if="mode === 'search'"
      v-model="searchValue"
      :placeholder="`搜索${field.name}`"
      :clearable="true"
      @input="handleSearchChange"
      @clear="handleSearchClear"
    />

    <RichTextResponseWidget v-else v-bind="props" />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { ElInput } from 'element-plus'
import type { FieldValue, WidgetComponentEmits, WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import RichTextResponseWidget from '@/architecture/presentation/widgets/RichTextResponseWidget.vue'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'
import { sanitizeHtml } from '@/utils/sanitizeHtml'

const RichTextEditorWidget = defineAsyncComponent(
  () => import('@/architecture/presentation/widgets/RichTextEditorWidget.vue'),
)

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {},
  }),
})

const emit = defineEmits<WidgetComponentEmits>()
const formDataStore = useFormDataStore()
const searchValue = ref('')

const previewText = computed(() => {
  const fieldValue = props.value ?? (props as any).modelValue
  const raw = fieldValue?.raw

  if (raw === null || raw === undefined || raw === '') {
    return ''
  }

  return stripHtml(String(raw))
})

function handleModelValueUpdate(value: FieldValue): void {
  emit('update:modelValue', value)
}

function handleStatisticsUpdated(statistics: Record<string, unknown>): void {
  emit('statistics:updated', statistics)
}

function handleDrawerChange(show: boolean): void {
  emit('drawer:change', show)
}

function createSearchFieldValue(raw: string | null): FieldValue {
  return {
    raw,
    display: raw ?? '',
    meta: {},
  } as FieldValue
}

function handleSearchChange(): void {
  const newFieldValue = searchValue.value
    ? createSearchFieldValue(searchValue.value)
    : createSearchFieldValue(null)

  formDataStore.setValue(props.fieldPath, newFieldValue)
  emit('update:modelValue', newFieldValue)
}

function handleSearchClear(): void {
  searchValue.value = ''
  handleSearchChange()
}

watch(
  () => props.value,
  (newValue: FieldValue | undefined) => {
    if (props.mode !== 'search') {
      return
    }

    const raw = newValue?.raw
    searchValue.value = raw === null || raw === undefined ? '' : String(raw)
  },
  { immediate: true, deep: true },
)

function stripHtml(html: string): string {
  if (!html) return ''

  const safeHtml = sanitizeHtml(html)

  try {
    const parser = new DOMParser()
    const doc = parser.parseFromString(safeHtml, 'text/html')
    return doc.body.textContent || doc.body.innerText || ''
  } catch {
    const container = document.createElement('div')
    container.innerHTML = safeHtml
    return container.textContent || container.innerText || ''
  }
}
</script>

<style scoped>
.rich-text-widget {
  width: 100%;
}

.table-cell-value {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.html-content-preview {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-text {
  color: var(--el-text-color-placeholder);
}
</style>
