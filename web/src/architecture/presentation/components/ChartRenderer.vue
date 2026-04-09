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
          <div class="filter-card-title">筛选条件</div>
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
                :search-type="field.search || 'eq'"
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
                :search-type="field.search || 'eq'"
                :form-renderer="formRendererContext"
                :function-method="props.functionDetail.method || 'GET'"
                :function-router="props.functionDetail.router || ''"
                mode="search"
              />
            </div>
          </el-form-item>
          
          <el-form-item class="filter-actions">
            <el-button type="primary" @click="handleSearch" :loading="loading" :icon="Search">
              查询
            </el-button>
            <el-button @click="handleReset" :icon="Refresh">
              重置
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
          <span v-else>图表</span>
          <div class="chart-actions">
            <el-button text @click="handleRefresh" :loading="loading" :icon="Refresh">
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <div v-if="isPlaceholderChart" class="chart-placeholder-tip">
        {{ chartPlaceholderMessage }}
      </div>
      
      <div v-loading="loading" class="chart-container">
        <div
          v-if="chartData"
          ref="chartContainerRef"
          class="chart-wrapper"
          :style="{ width: '100%', height: chartHeight }"
        ></div>
        <div v-else class="empty-chart">
          <el-empty description="暂无数据，请设置筛选条件后查询" />
        </div>
      </div>
    </el-card>
    
    <!-- Metadata 信息展示 -->
    <div v-if="chartData?.metadata && Object.keys(chartData.metadata).length > 0" class="metadata-card">
      <el-row :gutter="16">
        <el-col 
          v-for="(value, key) in chartData.metadata" 
          :key="key"
          :span="getMetadataSpan(Object.keys(chartData.metadata).length)"
        >
          <div class="metadata-item">
            <div class="metadata-label">{{ key }}</div>
            <div class="metadata-value">{{ formatMetadataValue(value) }}</div>
          </div>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElCard, ElForm, ElFormItem, ElButton, ElEmpty, ElRow, ElCol } from 'element-plus'
import { Search, Refresh } from '@element-plus/icons-vue'
import SearchInput from '@/architecture/presentation/components/SearchInput.vue'
import WidgetComponent from '@/architecture/presentation/widgets/WidgetComponent.vue'
import type { FieldConfig, FunctionDetail } from '@/architecture/domain/types'
import type { Chart } from '@/core/types/chart'
import { useChartDataLoader } from '@/architecture/presentation/composables/useChartDataLoader'
import { useChartFilterState } from '@/architecture/presentation/composables/useChartFilterState'
import { useChartInstanceLifecycle } from '@/architecture/presentation/composables/useChartInstanceLifecycle'
import {
  buildChartEChartsOption,
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

const chartData = ref<Chart | null>(null)
const chartContainerRef = ref<HTMLElement | null>(null)
const chartHeight = ref('600px')

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
  return data?.__placeholderMessage || '当前暂无图表数据，已按 0 值占位显示。'
})

const isPlaceholderChart = computed(() => {
  const data = chartData.value as RenderableChart | null
  return Boolean(data?.__placeholder)
})
useChartInstanceLifecycle({
  chartData: chartData as any,
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
      
      .metadata-label {
        font-size: 13px;
        color: var(--el-text-color-regular); // 使用 Element Plus 的常规文字色
        margin-bottom: 8px;
      }
      
      .metadata-value {
        font-size: 24px;
        font-weight: 600;
        color: var(--el-color-primary); // 使用 Element Plus 的主色，更突出
      }
    }
  }
}

</style>
