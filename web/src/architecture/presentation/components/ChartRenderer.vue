<!--
  ChartRenderer - BI 图表渲染器组件
  
  功能：
  - 根据 Request 字段自动生成筛选表单
  - 调用函数接口获取图表数据
  - 使用 ECharts 渲染图表
  - 显示 Metadata 信息
-->

<template>
  <div class="chart-renderer">
    <!-- 筛选表单 -->
    <div v-if="requestFields.length > 0" class="chart-filters">
      <div class="filter-card sci-fi-panel sci-fi-panel-expanded">
        <span class="sci-fi-accent-bar" />
        <div class="filter-card-inner">
          <div class="filter-card-title">{{ t('chartRenderer.filters') }}</div>
        <el-form :model="filterForm" class="filter-form">
          <el-form-item
            v-for="field in requestFields"
            :key="field.code"
            class="filter-field"
            :label="field.name"
            :required="isFieldRequired(field)"
          >
            <div class="widget-wrapper">
              <SearchInput
                v-if="shouldUseChartSearchInput(field)"
                :field="field"
                :model-value="getFieldRawValue(field.code)"
                search-type="eq"
                :function-method="props.functionDetail.method || 'GET'"
                :function-router="props.functionDetail.router || ''"
                @update:model-value="(v) => handleSearchFieldUpdate(field, v)"
              />
              <WidgetComponent
                v-else
                :field="field"
                :value="getFieldValue(field.code)"
                :model-value="getFieldValue(field.code)"
                @update:model-value="(v) => handleFieldUpdate(field.code, v)"
                :field-path="field.code"
                search-type="eq"
                :form-renderer="formRendererContext"
                :function-method="props.functionDetail.method || 'GET'"
                :function-router="props.functionDetail.router || ''"
                mode="search"
              />
            </div>
          </el-form-item>
          
          <el-form-item class="filter-actions">
            <el-button type="primary" @click="handleSearch" :loading="loading" :icon="Search">
              {{ t('chartRenderer.query') }}
            </el-button>
            <el-button @click="handleReset" :icon="Refresh">
              {{ t('chartRenderer.reset') }}
            </el-button>
          </el-form-item>
        </el-form>
        </div>
      </div>
    </div>

    <!-- 图表容器 -->
    <el-card class="chart-card">
      <template #header>
        <div class="chart-header">
          <span v-if="chartData?.title">{{ chartData.title }}</span>
          <span v-else>{{ t('chartRenderer.chart') }}</span>
          <div class="chart-actions">
            <el-button text @click="handleRefresh" :loading="loading" :icon="Refresh">
              {{ t('chartRenderer.refresh') }}
            </el-button>
          </div>
        </div>
      </template>

      <div v-if="isPlaceholderChart" class="chart-placeholder-tip">
        {{ chartPlaceholderMessage }}
      </div>
      
      <div v-loading="loading" class="chart-container">
        <template v-if="chartData">
          <div
            ref="chartContainerRef"
            class="chart-wrapper"
            :style="{ width: '100%', height: chartHeight }"
          ></div>
          <div v-if="shouldShowChartViewportControl" class="chart-viewport-control">
            <div class="chart-viewport-summary">
              <span class="chart-viewport-boundary">{{ chartViewportStartLabel }}</span>
              <span class="chart-viewport-count">{{ chartViewportCountLabel }}</span>
              <span class="chart-viewport-boundary is-end">{{ chartViewportEndLabel }}</span>
            </div>
            <el-slider
              v-model="chartViewportRange"
              class="chart-viewport-range"
              range
              :min="0"
              :max="chartViewportMaxIndex"
              :step="1"
              :format-tooltip="formatChartViewportTooltip"
              :marks="chartViewportMarks"
            />
          </div>
        </template>
        <div v-else class="empty-chart">
          <kageos-empty :description="t('chartRenderer.empty')" />
        </div>
      </div>
    </el-card>
    
    <!-- Metadata 信息展示 -->
    <div v-if="metadataItems.length > 0" class="metadata-card">
      <el-row :gutter="16">
        <el-col
          v-for="item in metadataItems"
          :key="item.key"
          :span="getMetadataSpan(metadataItems.length)"
        >
          <div class="metadata-item">
            <div class="metadata-label">{{ item.key }}</div>
            <el-tooltip
              v-if="item.truncated"
              :content="item.fullValue"
              :show-after="300"
              placement="top"
              popper-class="chart-metadata-tooltip"
            >
              <div class="metadata-value is-long is-truncated">{{ item.previewValue }}</div>
            </el-tooltip>
            <div
              v-else
              class="metadata-value"
              :class="{ 'is-long': item.isLong }"
            >
              {{ item.previewValue }}
            </div>
          </div>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElCard, ElForm, ElFormItem, ElButton, ElEmpty, ElRow, ElCol, ElTooltip, ElSlider } from 'element-plus'
import { Search, Refresh } from '@element-plus/icons-vue'
import SearchInput from '@/architecture/presentation/components/SearchInput.vue'
import WidgetComponent from '@/architecture/presentation/widgets/WidgetComponent.vue'
import type { FunctionDetail } from '@/architecture/domain/types'
import type { Chart } from '@/architecture/domain/types/chart'
import { useChartDataLoader } from '@/architecture/presentation/composables/useChartDataLoader'
import { useChartFilterState } from '@/architecture/presentation/composables/useChartFilterState'
import { useChartInstanceLifecycle } from '@/architecture/presentation/composables/useChartInstanceLifecycle'
import {
  buildChartEChartsOption,
  buildChartMetadataPreview,
  CARTESIAN_DATA_ZOOM_THRESHOLD,
  CARTESIAN_DEFAULT_VISIBLE_POINTS,
  createPendingQueryChart,
  formatChartMetadataValue as formatMetadataValue,
  getChartMetadataSpan as getMetadataSpan,
  normalizeRenderableChart,
  type RenderableChart,
} from './utils/chartRendererOption'
import { loadEChartsRuntime } from './utils/chartEChartsRuntime'

const props = defineProps<{
  functionDetail: FunctionDetail
}>()

const { t } = useI18n()
const chartData = ref<Chart | null>(null)
const chartContainerRef = ref<HTMLElement | null>(null)
const chartHeight = ref('600px')
const chartViewportRange = ref<[number, number]>([0, 0])

const metadataItems = computed(() => {
  const metadata = chartData.value?.metadata
  if (!metadata) return []

  return Object.entries(metadata).map(([key, value]) => {
    const fullValue = formatMetadataValue(value)
    const preview = buildChartMetadataPreview(fullValue)
    return {
      key,
      fullValue,
      previewValue: preview.text,
      truncated: preview.truncated,
      isLong: fullValue.length > 20,
    }
  })
})

const functionDetailRef = computed(() => props.functionDetail)
let triggerChartReload: () => Promise<void> = async () => {}
const {
  requestFields,
  filterForm,
  formRendererContext,
  initializeFieldValues,
  watchChartData,
  getFieldValue,
  getFieldRawValue,
  shouldUseChartSearchInput,
  isFieldRequired,
  handleFieldUpdate,
  handleSearchFieldUpdate,
  buildRequestParams,
  resetFilterValues,
} = useChartFilterState({
  functionDetail: functionDetailRef,
  onAutoSearch: () => {
    void triggerChartReload()
  }
})

const chartPlaceholderMessage = computed(() => {
  const data = chartData.value as RenderableChart | null
  return data?.__placeholderMessage || t('chartRenderer.placeholder')
})

const isPlaceholderChart = computed(() => {
  const data = chartData.value as RenderableChart | null
  return Boolean(data?.__placeholder)
})

const isCartesianChart = computed(() => {
  const chartType = chartData.value?.chart_type
  return chartType === 'line' || chartType === 'bar'
})

const chartXAxis = computed(() => {
  return Array.isArray(chartData.value?.x_axis) ? chartData.value.x_axis : []
})

const shouldShowChartViewportControl = computed(() => {
  return isCartesianChart.value && chartXAxis.value.length > CARTESIAN_DATA_ZOOM_THRESHOLD
})

const chartViewportMaxIndex = computed(() => {
  return Math.max(0, chartXAxis.value.length - 1)
})

const normalizedChartViewportStart = computed(() => {
  const [start, end] = chartViewportRange.value
  const minValue = Math.min(start, end)
  return Math.min(Math.max(0, minValue), chartViewportMaxIndex.value)
})

const normalizedChartViewportEnd = computed(() => {
  const [start, end] = chartViewportRange.value
  const maxValue = Math.max(start, end)
  return Math.min(Math.max(normalizedChartViewportStart.value, maxValue), chartViewportMaxIndex.value)
})

const chartViewportEndExclusive = computed(() => {
  return Math.min(chartXAxis.value.length, normalizedChartViewportEnd.value + 1)
})

const chartViewportVisibleCount = computed(() => {
  return Math.max(0, chartViewportEndExclusive.value - normalizedChartViewportStart.value)
})

const chartViewportStartLabel = computed(() => {
  return chartXAxis.value[normalizedChartViewportStart.value] || ''
})

const chartViewportEndLabel = computed(() => {
  return chartXAxis.value[normalizedChartViewportEnd.value] || ''
})

const chartViewportCountLabel = computed(() => {
  return `${chartViewportVisibleCount.value} / ${chartXAxis.value.length}`
})

const CHART_VIEWPORT_MARK_COUNT = 5
const CHART_VIEWPORT_MARK_LABEL_RE = /^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::\d{2})?)?$/

const formatChartViewportMarkLabel = (value: string): string => {
  const label = String(value || '')
  const match = label.match(CHART_VIEWPORT_MARK_LABEL_RE)
  if (match) {
    const [, , month, day, hour, minute] = match
    return hour && minute ? `${month}-${day} ${hour}:${minute}` : `${month}-${day}`
  }
  return label.length > 12 ? `${label.slice(0, 11)}...` : label
}

const chartViewportMarks = computed<Record<number, string>>(() => {
  const max = chartViewportMaxIndex.value
  if (max <= 0) return {}

  const indexes = new Set<number>()
  for (let i = 0; i < CHART_VIEWPORT_MARK_COUNT; i += 1) {
    indexes.add(Math.round((max * i) / (CHART_VIEWPORT_MARK_COUNT - 1)))
  }

  return Object.fromEntries(
    Array.from(indexes)
      .sort((a, b) => a - b)
      .map((index) => [
        index,
        formatChartViewportMarkLabel(chartXAxis.value[index] || String(index + 1)),
      ])
  )
})

const formatChartViewportTooltip = (value: number): string => {
  return chartXAxis.value[value] || ''
}

const visibleChartData = computed<Chart | null>(() => {
  const data = chartData.value
  if (!data || !shouldShowChartViewportControl.value) {
    return data
  }

  const start = normalizedChartViewportStart.value
  const end = chartViewportEndExclusive.value
  return {
    ...data,
    x_axis: chartXAxis.value.slice(start, end),
    series: data.series.map((series) => ({
      ...series,
      data: Array.isArray(series.data) ? series.data.slice(start, end) : series.data,
    })),
  }
})

watch(
  () => `${chartData.value?.chart_type || ''}:${chartXAxis.value.length}`,
  () => {
    const end = chartViewportMaxIndex.value
    const start = Math.max(0, end - CARTESIAN_DEFAULT_VISIBLE_POINTS + 1)
    chartViewportRange.value = [start, end]
  },
  { flush: 'post' }
)

useChartInstanceLifecycle({
  chartData: visibleChartData,
  chartContainerRef,
  loadChartRuntime: loadEChartsRuntime,
  buildOption: (chart) => buildChartEChartsOption(chart as RenderableChart),
})
const {
  loading,
  loadChartData,
  handleSearch,
  handleReset,
  handleRefresh,
} = useChartDataLoader({
  functionDetail: functionDetailRef,
  requestFields,
  chartData,
  initializeFieldValues,
  watchChartData,
  buildRequestParams,
  resetFilterValues,
  normalizeChartData: normalizeRenderableChart,
  createPendingChart: createPendingQueryChart,
})
triggerChartReload = loadChartData
</script>

<style scoped lang="scss">
.chart-renderer {
  width: 100%;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;

  .chart-filters {
    margin: 0;
  }
  
  .filter-card {
    margin: 0;
    position: relative;
    border-radius: 8px;
    border: 1px solid var(--el-border-color-lighter);
    background: var(--el-bg-color);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
    overflow: hidden;

    .sci-fi-accent-bar {
      position: absolute;
      left: 0;
      top: 0;
      bottom: 0;
      width: 3px;
      background: linear-gradient(180deg, rgba(0, 212, 255, 0.9), rgba(0, 212, 255, 0.4));
      box-shadow: 0 0 10px rgba(0, 212, 255, 0.5);
    }

    .filter-card-inner {
      padding: 20px 20px 20px 24px;
    }

    .filter-card-title {
      margin-bottom: 14px;
      font-size: 14px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      letter-spacing: 0.3px;
    }
    
    .filter-form {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 12px 14px;
      margin-top: 0;
      align-items: start;
    }

    :deep(.filter-form .el-form-item) {
      margin-bottom: 0;
      display: flex;
      flex-direction: column;
      align-items: stretch;
    }

    :deep(.filter-form .el-form-item__label) {
      width: 100%;
      justify-content: flex-start;
      line-height: 1.25;
      margin: 0;
      padding: 0;
      color: var(--el-text-color-regular);
      font-size: 12px;
      font-weight: 600;
    }

    :deep(.filter-form .el-form-item__label-wrap) {
      width: 100%;
      margin: 0 0 6px;
    }

    :deep(.filter-form .el-form-item__content) {
      width: 100%;
      min-width: 0;
      display: flex;
    }

    .widget-wrapper {
      min-width: 0;
      width: 100%;
    }

    :deep(.widget-wrapper > *) {
      width: 100%;
    }

    .filter-actions {
      grid-column: 1 / -1;
    }

    :deep(.filter-actions .el-form-item__content) {
      justify-content: flex-end;
      gap: 8px;
      flex-wrap: wrap;
    }

    @media (max-width: 768px) {
      .filter-form {
        grid-template-columns: 1fr;
      }

      :deep(.filter-actions .el-form-item__content) {
        justify-content: flex-start;
      }
    }
  }
  
  .chart-card {
    margin: 0;

    :deep(.el-card__header) {
      padding: 16px 20px 14px;
      border-bottom: 1px solid var(--el-border-color-lighter);
    }

    :deep(.el-card__body) {
      padding: 0 20px 20px;
    }
    
    .chart-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      
      .chart-actions {
        display: flex;
        gap: 10px;
      }
    }

    .chart-placeholder-tip {
      margin: 14px 0 0;
      padding: 10px 12px;
      border-radius: 8px;
      background: var(--el-fill-color-light);
      color: var(--el-text-color-regular);
      font-size: 13px;
      line-height: 1.5;
    }
    
    .chart-container {
      width: 100%;
      min-height: 400px;
      margin-top: 16px;
      
      .chart-wrapper {
        width: 100%;
        height: 100%;
      }

      .chart-viewport-control {
        margin: 12px 0 0;
        padding: 4px 6px 10px;
        border-top: 1px solid var(--el-border-color-extra-light);
      }

      .chart-viewport-summary {
        display: grid;
        grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
        align-items: center;
        gap: 10px;
        min-height: 22px;
        margin-bottom: 0;
      }

      .chart-viewport-boundary {
        min-width: 0;
        max-width: 100%;
        justify-self: start;
        color: var(--el-text-color-secondary);
        font-size: 12px;
        line-height: 1.4;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .chart-viewport-boundary.is-end {
        justify-self: end;
        text-align: right;
      }

      .chart-viewport-count {
        justify-self: center;
        color: var(--el-color-primary);
        font-size: 12px;
        font-weight: 600;
        font-variant-numeric: tabular-nums;
        white-space: nowrap;
      }

      .chart-viewport-range {
        width: 100%;
        margin: 0;
        padding: 0 24px 10px;

        :deep(.el-slider__runway) {
          height: 4px;
          margin: 12px 0 22px;
          border-radius: 999px;
          background: var(--el-fill-color);
        }

        :deep(.el-slider__bar) {
          height: 4px;
          background: var(--el-color-primary);
          border-radius: 999px;
        }

        :deep(.el-slider__button-wrapper) {
          top: -16px;
          width: 32px;
          height: 32px;
        }

        :deep(.el-slider__button) {
          width: 12px;
          height: 12px;
          border: 2px solid var(--el-color-primary);
          background: var(--el-bg-color);
          box-shadow: 0 1px 4px var(--el-border-color);
        }

        :deep(.el-slider__stop) {
          width: 2px;
          height: 2px;
          background: var(--el-border-color);
        }

        :deep(.el-slider__marks-text) {
          margin-top: 2px;
          max-width: 72px;
          color: var(--el-text-color-secondary);
          font-size: 11px;
          line-height: 1.2;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        :deep(.el-slider__marks-stop) {
          width: 1px;
          height: 6px;
          margin-top: -1px;
          border-radius: 0;
          background: var(--el-border-color);
        }
      }

      @media (max-width: 768px) {
        .chart-viewport-control {
          padding: 4px 2px 10px;
        }

        .chart-viewport-summary {
          gap: 8px;
        }

        .chart-viewport-range {
          padding: 0 18px 10px;
        }

        .chart-viewport-range :deep(.el-slider__marks-text) {
          max-width: 56px;
        }
      }
      
      .empty-chart {
        display: flex;
        justify-content: center;
        align-items: center;
        min-height: 400px;
      }
    }
  }
  
  .metadata-card {
    margin-top: 0;
    padding: 16px;
    background-color: var(--el-fill-color-light); // 使用 Element Plus 的浅色填充色
    border-radius: 8px;
    border: 1px solid var(--el-border-color); // 使用 Element Plus 的边框色
    width: 100%;
    box-sizing: border-box;
    
    // 确保栅格系统不会产生额外的边距
    :deep(.el-row) {
      margin: 0 !important;
    }
    
    :deep(.el-col) {
      padding: 0 8px !important; // 左右各留 8px 间距
    }
    
    .metadata-item {
      text-align: center;
      padding: 12px 0;
      min-width: 0;
      
      .metadata-label {
        font-size: 13px;
        color: var(--el-text-color-regular); // 使用 Element Plus 的常规文字色
        margin-bottom: 8px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
      
      .metadata-value {
        max-width: 100%;
        font-size: 24px;
        font-weight: 600;
        color: var(--el-color-primary); // 使用 Element Plus 的主色，更突出
        line-height: 1.2;
        overflow-wrap: anywhere;
      }

      .metadata-value.is-long {
        display: block;
        font-size: 14px;
        font-weight: 500;
        line-height: 1.45;
        color: var(--el-text-color-primary);
      }

      .metadata-value.is-truncated {
        cursor: help;
      }
    }
  }
}

:global(.chart-metadata-tooltip) {
  max-width: min(720px, calc(100vw - 32px));
  line-height: 1.5;
  white-space: normal;
  overflow-wrap: anywhere;
}

</style>
