<template>
  <div class="workspace-function-renderer">
    <details
      v-if="functionUsageGuide"
      class="function-guide"
      @toggle="handleGuideToggle"
    >
      <summary class="function-guide-summary">
        <span>{{ t('functionGuide.title') }}</span>
        <span class="function-guide-hint">
          {{ functionGuideOpen ? t('functionGuide.collapse') : t('functionGuide.expand') }}
        </span>
      </summary>
      <div class="function-guide-body" v-html="renderMarkdown(functionUsageGuide)" />
    </details>

    <div class="function-runtime">
      <template v-if="matchedFunctionDetail">
        <FormView
          v-if="functionDetail?.template_type === TEMPLATE_TYPE.FORM"
          :ref="formViewRefTarget || undefined"
          :key="`form-${keyBase}`"
          :function-detail="functionDetail"
        />
        <TableView
          v-else-if="functionDetail?.template_type === TEMPLATE_TYPE.TABLE"
          :key="`table-${keyBase}`"
          :function-detail="functionDetail"
        />
        <ChartView
          v-else-if="functionDetail?.template_type === TEMPLATE_TYPE.CHART"
          :key="`chart-${keyBase}`"
          :function-detail="functionDetail"
        />
        <div v-else :key="`empty-${keyBase}`" class="function-loading">
          <el-skeleton :rows="8" animated />
        </div>
      </template>
      <div v-else :key="`loading-${keyBase}`" class="function-loading">
        <el-skeleton :rows="8" animated />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FunctionDetail } from '@/architecture/domain/types'
import type { ServiceTree as ServiceTreeType } from '@/architecture/domain/types'
import { TEMPLATE_TYPE } from '@/architecture/domain/constants/functionTypes'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'

const FormView = defineAsyncComponent(() => import('@/architecture/presentation/views/FormView.vue'))
const TableView = defineAsyncComponent(() => import('@/architecture/presentation/views/TableView.vue'))
const ChartView = defineAsyncComponent(() => import('@/architecture/presentation/views/ChartView.vue'))

const props = defineProps<{
  currentFunction: ServiceTreeType | null
  functionDetail: FunctionDetail | null
  formViewRefTarget?: any
}>()

const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()
const { t } = useI18n()
const functionGuideOpen = ref(false)

const keyBase = computed(() => props.currentFunction?.full_code_path || props.currentFunction?.id || 'unknown')

const functionUsageGuide = computed(() => {
  return (props.functionDetail?.description || props.currentFunction?.description || '').trim()
})

const matchedFunctionDetail = computed(() => {
  if (!props.functionDetail || !props.currentFunction) {
    return false
  }

  return (
    props.functionDetail.id === props.currentFunction.ref_id ||
    props.functionDetail.router === props.currentFunction.full_code_path
  )
})

function handleGuideToggle(event: Event) {
  functionGuideOpen.value = (event.currentTarget as HTMLDetailsElement).open
}
</script>

<style scoped lang="scss">
.workspace-function-renderer {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.function-guide {
  flex: 0 0 auto;
  margin: 4px 0 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  overflow: hidden;
}

.function-guide-summary {
  min-height: 40px;
  padding: 0 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  list-style: none;
  user-select: none;
}

.function-guide-summary::-webkit-details-marker {
  display: none;
}

.function-guide-summary::before {
  content: '';
  width: 0;
  height: 0;
  border-top: 5px solid transparent;
  border-bottom: 5px solid transparent;
  border-left: 6px solid var(--el-color-primary);
  transition: transform 0.18s ease;
}

.function-guide[open] .function-guide-summary::before {
  transform: rotate(90deg);
}

.function-guide-hint {
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.function-guide-body {
  padding: 0 16px 16px 34px;
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.72;
}

.function-runtime {
  flex: 1;
  min-height: 0;
}

.function-loading {
  padding: 24px;
  width: 100%;
}
</style>
