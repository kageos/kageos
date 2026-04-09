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
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { ElCard, ElForm, ElFormItem, ElButton, ElEmpty, ElMessage, ElRow, ElCol, ElDialog, ElInput, ElText, ElNotification } from 'element-plus'
import { Search, Refresh } from '@element-plus/icons-vue'
import type { EChartsType, EChartsCoreOption } from 'echarts/core'
import SearchInput from '@/architecture/presentation/components/SearchInput.vue'
import WidgetComponent from '@/architecture/presentation/widgets/WidgetComponent.vue'
import { executeFunction } from '@/api/function'
import type { FieldConfig, FieldValue, FunctionDetail } from '@/architecture/domain/types'
import type { Chart, ChartSeries } from '@/core/types/chart'
import { widgetComponentFactory } from '@/architecture/infrastructure/widgetRegistry'
import { hasAnyRequiredRule } from '@/core/utils/validationUtils'
import { convertToFieldValue } from '@/utils/field'
import { useChartParamURLSync } from '@/architecture/presentation/composables/useChartParamURLSync'
import { convertValueByFieldType } from '@/architecture/presentation/widgets/utils/typeConverter'
import { getWidgetDefaultValue } from '@/architecture/presentation/widgets/composables/useWidgetDefaultValue'
import { useAuthStore } from '@/stores/auth'
import { createEmptyFieldValue, createEmptyRawFieldValue } from '@/core/utils/createFieldValue'

const props = defineProps<{
  functionDetail: FunctionDetail
}>()

const route = useRoute()

// 状态
const loading = ref(false)
const chartData = ref<Chart | null>(null)
const chartContainerRef = ref<HTMLElement | null>(null)
const chartInstance = ref<EChartsType | null>(null)
const chartHeight = ref('600px')

type RenderableChart = Chart & {
  __placeholder?: boolean
  __placeholderMessage?: string
}

type EChartsRuntime = {
  init: (typeof import('echarts/core'))['init']
  use: (typeof import('echarts/core'))['use']
}

let echartsRuntimePromise: Promise<EChartsRuntime> | null = null
let echartsBaseRegistered = false
const registeredChartTypes = new Set<string>()

async function loadChartInstaller(chartType: string): Promise<{ install: (registers: unknown) => void } | null> {
  switch (chartType) {
    case 'bar':
      return import('echarts/lib/chart/bar/install.js')
    case 'line':
      return import('echarts/lib/chart/line/install.js')
    case 'pie':
      return import('echarts/lib/chart/pie/install.js')
    case 'gauge':
      return import('echarts/lib/chart/gauge/install.js')
    default:
      return null
  }
}

async function loadEChartsRuntime(chartType: string): Promise<{ init: (typeof import('echarts/core'))['init'] }> {
  if (!echartsRuntimePromise) {
    echartsRuntimePromise = (async () => {
      const [
        { use, init },
        { install: TitleComponent },
        { install: TooltipComponent },
        { install: AxisPointerComponent },
        { install: LegendComponent },
        { install: GridComponent },
        { install: CanvasRenderer },
      ] = await Promise.all([
        import('echarts/core'),
        import('echarts/lib/component/title/install.js'),
        import('echarts/lib/component/tooltip/install.js'),
        import('echarts/lib/component/axisPointer/install.js'),
        import('echarts/lib/component/legend/install.js'),
        import('echarts/lib/component/grid/install.js'),
        import('echarts/lib/renderer/installCanvasRenderer.js'),
      ])

      if (!echartsBaseRegistered) {
        use([
          TitleComponent,
          TooltipComponent,
          AxisPointerComponent,
          LegendComponent,
          GridComponent,
          CanvasRenderer,
        ])
        echartsBaseRegistered = true
      }

      return { init, use }
    })()
  }

  const runtime = await echartsRuntimePromise

  if (!registeredChartTypes.has(chartType)) {
    const chartInstaller = await loadChartInstaller(chartType)
    if (chartInstaller) {
      runtime.use([chartInstaller.install])
      registeredChartTypes.add(chartType)
    }
  }

  return { init: runtime.init }
}

// 请求字段和筛选表单
const requestFields = computed(() => {
  if (!props.functionDetail.request) return []
  return (props.functionDetail.request as FieldConfig[]).filter(field => {
    // 只显示有 widget 配置的字段
    return field.widget && field.widget.type
  }).map(field => {
    // 为 select 类型的字段添加 clearable 支持
    if (field.widget && (field.widget.type === 'select' || field.widget.type === 'multiselect')) {
      return {
        ...field,
        widget: {
          ...field.widget,
          clearable: true // 添加清空功能
        }
      }
    }
    return field
  })
})

const chartPlaceholderMessage = computed(() => {
  const data = chartData.value as RenderableChart | null
  return data?.__placeholderMessage || '当前暂无图表数据，已按 0 值占位显示。'
})

const isPlaceholderChart = computed(() => {
  const data = chartData.value as RenderableChart | null
  return Boolean(data?.__placeholder)
})

// 筛选表单数据
const filterForm = ref<Record<string, any>>({})

// 字段值
const fieldValues = ref<Record<string, FieldValue>>({})

// 初始化字段值（优先 URL 参数，其次 widget.config.default 默认值）
const initializeFieldValues = () => {
  const values: Record<string, FieldValue> = {}
  requestFields.value.forEach((field: FieldConfig) => {
    // 1. 优先从 URL 查询参数中获取初始值
    const queryValue = route.query[field.code]
    const value = Array.isArray(queryValue) ? queryValue[0] : queryValue

    if (value !== undefined && value !== null && value !== '') {
      // 使用统一的类型转换工具
      const rawValue = convertValueByFieldType(value, field)
      values[field.code] = convertToFieldValue(rawValue, field)
      filterForm.value[field.code] = rawValue
    } else {
      // 2. 无 URL 参数时，使用 widget.config.default 默认值（如 Now(-90d)、Now()）
      const defaultValue = getWidgetDefaultValue(field, undefined, () => useAuthStore())
      if (defaultValue.raw !== null && defaultValue.raw !== undefined && defaultValue.raw !== '') {
        values[field.code] = defaultValue
        filterForm.value[field.code] = defaultValue.raw
      } else {
        values[field.code] = createEmptyFieldValue(field)
        filterForm.value[field.code] = null
      }
    }
  })
  fieldValues.value = values
}

// FormRenderer 上下文
const formRendererContext = computed(() => {
  return {
    getFunctionMethod: () => props.functionDetail.method || 'GET',
    getFunctionRouter: () => props.functionDetail.router || '',
    getSubmitData: () => {
      const data: Record<string, any> = {}
      Object.keys(fieldValues.value).forEach(key => {
        const value = fieldValues.value[key]
        if (value && value.raw !== null && value.raw !== undefined) {
          data[key] = value.raw
        }
      })
      return data
    },
    registerWidget: () => {},
    unregisterWidget: () => {},
    getFieldError: () => null
  }
})

// 获取字段值
const getFieldValue = (fieldCode: string): FieldValue => {
  return fieldValues.value[fieldCode] || createEmptyRawFieldValue()
}

const getFieldRawValue = (fieldCode: string): any => {
  return getFieldValue(fieldCode).raw ?? null
}

const shouldUseChartSearchInput = (field: FieldConfig): boolean => {
  return !field.callbacks?.includes('OnSelectFuzzy')
}

// 🔥 使用 Chart 参数 URL 同步
const { watchChartData } = useChartParamURLSync({
  functionDetail: computed(() => props.functionDetail),
  fieldValues,
  enabled: true,
  debounceMs: 300
})

// 更新字段值
const handleFieldUpdate = (fieldCode: string, value: FieldValue): void => {
  const oldValue = fieldValues.value[fieldCode]
  const oldRaw = oldValue?.raw ?? null
  const newRaw = value?.raw ?? null
  
  fieldValues.value[fieldCode] = value
  filterForm.value[fieldCode] = value.raw
  
  // 🔥 如果值发生变化（选中、修改或清除），自动刷新数据
  // 判断值是否真正发生变化（考虑 null、undefined、空字符串都视为空值）
  const oldIsEmpty = oldRaw == null || oldRaw === ''
  const newIsEmpty = newRaw == null || newRaw === ''
  const valueChanged = oldRaw !== newRaw && (oldIsEmpty !== newIsEmpty || (!oldIsEmpty && !newIsEmpty))
  
  if (valueChanged) {
    loadChartData()
  }
}

const handleSearchFieldUpdate = (field: FieldConfig, rawValue: any): void => {
  handleFieldUpdate(field.code, convertToFieldValue(rawValue, field))
}

// 判断字段是否必填
const isFieldRequired = (field: FieldConfig): boolean => {
  return hasAnyRequiredRule(field)
}

// 格式化 Metadata 值
const formatMetadataValue = (value: any): string => {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'object') {
    return JSON.stringify(value)
  }
  return String(value)
}

// 计算统计组件的span值（固定一行显示4个）
const getMetadataSpan = (count: number): number => {
  // 固定返回 6，因为 el-col 的 span 总共是 24，24/4=6
  return 6
}

const hasRenderableSeriesData = (series: ChartSeries, chartType: string): boolean => {
  if (!series || !Array.isArray(series.data) || series.data.length === 0) {
    return false
  }

  switch (chartType) {
    case 'pie':
      return series.data.some((item: any) => {
        if (typeof item === 'number') {
          return item !== 0
        }
        if (typeof item === 'object' && item !== null && 'value' in item) {
          return Number((item as any).value) !== 0
        }
        return Boolean(item)
      })
    case 'gauge':
      return series.data.some((item: any) => {
        if (typeof item === 'number') {
          return item !== 0
        }
        if (typeof item === 'object' && item !== null && 'value' in item) {
          return Number((item as any).value) !== 0
        }
        return Boolean(item)
      })
    default:
      return series.data.some((item: any) => Number(item) !== 0 || String(item).trim() !== '')
  }
}

const hasRenderableChartData = (chart: Chart | null | undefined): boolean => {
  if (!chart || !Array.isArray(chart.series) || chart.series.length === 0) {
    return false
  }

  return chart.series.some((series) => hasRenderableSeriesData(series, chart.chart_type))
}

const createZeroValueChart = (base?: Partial<RenderableChart> | null): RenderableChart => {
  const chartType = base?.chart_type || 'bar'
  const placeholderMessage = base?.__placeholderMessage || '当前暂无图表数据，已按 0 值占位显示。'

  if (chartType === 'pie') {
    const pieSeries = Array.isArray(base?.series) && base!.series!.length > 0
      ? base!.series!.map((series, index) => {
          const candidateNames = series.data
            ?.map((item: any) => (typeof item === 'object' && item !== null ? item.name : null))
            .filter(Boolean) as string[] | undefined

          const pieData = (candidateNames && candidateNames.length > 0 ? candidateNames : ['暂无数据'])
            .map((name) => ({ name, value: 0 }))

          return {
            ...series,
            name: series.name || `系列${index + 1}`,
            data: pieData,
          }
        })
      : [{
          name: '数值',
          data: [{ name: '暂无数据', value: 0 }],
        }]

    return {
      chart_type: 'pie',
      title: base?.title,
      metadata: base?.metadata,
      series: pieSeries,
      widget_type: base?.widget_type,
      data_type: base?.data_type,
      __placeholder: true,
      __placeholderMessage: placeholderMessage,
    }
  }

  if (chartType === 'gauge') {
    const gaugeSeries = Array.isArray(base?.series) && base!.series!.length > 0
      ? base!.series!.map((series, index) => ({
          ...series,
          name: series.name || `系列${index + 1}`,
          data: [{ value: 0 }],
        }))
      : [{
          name: '当前值',
          data: [{ value: 0 }],
        }]

    return {
      chart_type: 'gauge',
      title: base?.title,
      metadata: base?.metadata,
      series: gaugeSeries,
      widget_type: base?.widget_type,
      data_type: base?.data_type,
      __placeholder: true,
      __placeholderMessage: placeholderMessage,
    }
  }

  const xAxis = Array.isArray(base?.x_axis) && base!.x_axis!.length > 0 ? base!.x_axis! : ['暂无数据']
  const commonSeries = Array.isArray(base?.series) && base!.series!.length > 0
    ? base!.series!.map((series, index) => ({
        ...series,
        name: series.name || `系列${index + 1}`,
        data: xAxis.map(() => 0),
      }))
    : [{
        name: '数值',
        data: xAxis.map(() => 0),
      }]

  return {
    chart_type: chartType === 'line' ? 'line' : 'bar',
    title: base?.title,
    x_axis: xAxis,
    metadata: base?.metadata,
    series: commonSeries,
    widget_type: base?.widget_type,
    data_type: base?.data_type,
    __placeholder: true,
    __placeholderMessage: placeholderMessage,
  }
}

const normalizeChartData = (chart: Chart): RenderableChart => {
  if (hasRenderableChartData(chart)) {
    return chart as RenderableChart
  }

  return createZeroValueChart({
    ...chart,
    __placeholderMessage: requestFields.value.length > 0
      ? '当前暂无图表数据，已按 0 值占位显示，可继续调整筛选条件。'
      : '当前暂无图表数据，已按 0 值占位显示。'
  })
}

const createPendingQueryChart = (): RenderableChart => {
  return createZeroValueChart({
    chart_type: 'bar',
    title: props.functionDetail.name || '图表',
    __placeholderMessage: '请先设置筛选条件后查询，当前以 0 值占位显示。'
  })
}

const formatSeriesTooltip = (params: any): string => {
  if (!params) {
    return '无数据'
  }

  if (Array.isArray(params)) {
    if (params.length === 0) {
      return '无数据'
    }

    const title = params[0]?.axisValue || params[0]?.name || ''
    const lines = params.map((param: any) => {
      const value = typeof param.value === 'number'
        ? (param.value % 1 === 0 ? param.value : param.value.toFixed(2))
        : param.value
      const name = param.seriesName || param.name || ''
      const color = param.color || '#5470c6'

      return `<div style="display: flex; align-items: center; margin-bottom: 4px;">
        <span style="display: inline-block; width: 10px; height: 10px; background-color: ${color}; border-radius: 50%; margin-right: 8px;"></span>
        <span style="flex: 1;">${name}:</span>
        <span style="font-weight: bold; margin-left: 10px;">${value}</span>
      </div>`
    }).join('')

    return `<div style="font-weight: bold; margin-bottom: 8px;">${title}</div>${lines}`
  }

  const value = typeof params.value === 'number'
    ? (params.value % 1 === 0 ? params.value : params.value.toFixed(2))
    : params.value
  const title = params.name || params.axisValue || params.seriesName || ''
  const color = params.color || '#5470c6'
  const name = params.seriesName || '数值'

  return `<div style="font-weight: bold; margin-bottom: 8px;">${title}</div>
    <div style="display: flex; align-items: center;">
      <span style="display: inline-block; width: 10px; height: 10px; background-color: ${color}; border-radius: 50%; margin-right: 8px;"></span>
      <span style="flex: 1;">${name}:</span>
      <span style="font-weight: bold; margin-left: 10px;">${value}</span>
    </div>`
}

// 构建 ECharts 配置
const buildEChartsOption = (chart: RenderableChart): EChartsCoreOption => {
  // 先检查数据是否有效
  if (!chart || !chart.series || chart.series.length === 0) {
    return {}
  }
  
  const option: EChartsCoreOption = {
    // 设置背景色为白色，提高对比度
    backgroundColor: '#ffffff',
    title: chart.title ? { 
      text: chart.title,
      left: 'center',
      top: 10,
      textStyle: {
        fontSize: 18,
        fontWeight: 'bold',
        color: '#1f2937' // 深灰色，提高对比度
      }
    } : undefined,
    // tooltip 不在基础配置中设置，让每个图表类型自己配置
    legend: {
      data: chart.series.map(s => s.name),
      top: chart.title ? 40 : 10,
      textStyle: {
        fontSize: 13,
        color: '#374151' // 深灰色，提高可读性
      },
      itemWidth: 20,
      itemHeight: 14
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: chart.title ? '15%' : '10%',
      containLabel: true
    }
  }

  // 根据图表类型构建配置
  switch (chart.chart_type) {
    case 'bar':
      // 柱状图 tooltip 配置（参照折线图的样式）
      option.tooltip = {
        show: true, // 明确启用 tooltip
        trigger: 'item',
        backgroundColor: 'rgba(50, 50, 50, 0.9)',
        borderColor: '#333',
        borderWidth: 1,
        padding: [10, 15],
        textStyle: {
          color: '#fff',
          fontSize: 13,
          lineHeight: 20
        },
        formatter: formatSeriesTooltip
      }
      option.xAxis = {
        type: 'category',
        data: chart.x_axis || [],
        axisLabel: {
          fontSize: 12,
          color: '#374151' // 深灰色，提高可读性
        },
        axisLine: {
          lineStyle: {
            color: '#d1d5db' // 浅灰色轴线
          }
        }
      }
      option.yAxis = {
        type: 'value',
        axisLabel: {
          fontSize: 12,
          color: '#374151', // 深灰色，提高可读性
          formatter: (value: number) => {
            // 格式化 Y 轴标签，大数值显示为 K、M 等单位
            if (value >= 1000000) {
              return (value / 1000000).toFixed(1) + 'M'
            } else if (value >= 1000) {
              return (value / 1000).toFixed(1) + 'K'
            }
            return value.toString()
          }
        },
        axisLine: {
          lineStyle: {
            color: '#d1d5db' // 浅灰色轴线
          }
        },
        splitLine: {
          lineStyle: {
            color: '#e5e7eb', // 浅灰色网格线
            type: 'dashed'
          }
        }
      }
      option.series = chart.series.map(s => ({
        name: s.name,
        type: 'bar',
        data: s.data,
        ...s.config
      }))
      break

    case 'line':
      // 折线图 tooltip 配置（参照 gauge 图表的样式）
      option.tooltip = {
        show: true, // 明确启用 tooltip
        trigger: 'item',
        backgroundColor: 'rgba(50, 50, 50, 0.9)',
        borderColor: '#333',
        borderWidth: 1,
        padding: [10, 15],
        textStyle: {
          color: '#fff',
          fontSize: 13,
          lineHeight: 20
        },
        formatter: formatSeriesTooltip
      }
      // 折线图必须有 X 轴数据
      if (!chart.x_axis || chart.x_axis.length === 0) {
        // 如果没有 X 轴数据，返回最小配置，避免 resize 时出错
        return {
          ...option,
          xAxis: { type: 'category', data: [] },
          yAxis: { type: 'value' },
          series: []
        }
      }
      option.xAxis = {
        type: 'category',
        data: chart.x_axis,
        axisLabel: {
          fontSize: 12,
          color: '#374151' // 深灰色，提高可读性
        },
        axisLine: {
          lineStyle: {
            color: '#d1d5db' // 浅灰色轴线
          }
        }
      }
      option.yAxis = {
        type: 'value',
        axisLabel: {
          fontSize: 12,
          color: '#374151', // 深灰色，提高可读性
          formatter: (value: number) => {
            // 格式化 Y 轴标签，大数值显示为 K、M 等单位
            if (value >= 1000000) {
              return (value / 1000000).toFixed(1) + 'M'
            } else if (value >= 1000) {
              return (value / 1000).toFixed(1) + 'K'
            }
            return value.toString()
          }
        },
        axisLine: {
          lineStyle: {
            color: '#d1d5db' // 浅灰色轴线
          }
        },
        splitLine: {
          lineStyle: {
            color: '#e5e7eb', // 浅灰色网格线
            type: 'dashed'
          }
        }
      }
      option.series = chart.series.map(s => ({
        name: s.name,
        type: 'line',
        data: s.data || []
        // 暂时移除所有额外配置，确保 tooltip 能正常工作
        // ...s.config
      }))
      break

    case 'pie':
      // 饼图使用 item trigger 的 tooltip
      option.tooltip = {
        trigger: 'item',
        formatter: (params: any) => {
          const value = typeof params.value === 'number'
            ? (params.value % 1 === 0 ? params.value : params.value.toFixed(2))
            : (typeof params.value === 'object' && params.value?.value !== undefined)
            ? (typeof params.value.value === 'number' 
              ? (params.value.value % 1 === 0 ? params.value.value : params.value.value.toFixed(2))
              : params.value.value)
            : params.value
          const name = params.name || ''
          const percent = params.percent ? ` (${params.percent}%)` : ''
          return `<div style="font-weight: bold; margin-bottom: 8px;">${name}</div>
            <div style="display: flex; align-items: center;">
              <span style="display: inline-block; width: 10px; height: 10px; background-color: ${params.color || '#5470c6'}; border-radius: 50%; margin-right: 8px;"></span>
              <span style="flex: 1;">数值:</span>
              <span style="font-weight: bold; margin-left: 10px;">${value}${percent}</span>
            </div>`
        }
      }
      option.series = chart.series.map(s => ({
        name: s.name,
        type: 'pie',
        radius: '50%',
        stillShowZeroSum: true,
        data: s.data.map((item: any) => {
          if (typeof item === 'object' && item !== null) {
            return item
          }
          return { value: item }
        }),
        label: {
          fontSize: 13,
          color: '#374151', // 深灰色，提高可读性
          fontWeight: 'normal',
          formatter: '{b}: {c} ({d}%)' // 显示名称、数值和百分比
        },
        labelLine: {
          lineStyle: {
            color: '#6b7280' // 标签线颜色
          }
        },
        emphasis: {
          label: {
            fontSize: 14,
            fontWeight: 'bold'
          },
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        },
        ...s.config
      }))
      break

    case 'gauge':
      // 仪表盘使用 item trigger 的 tooltip
      option.tooltip = {
        trigger: 'item',
        backgroundColor: 'rgba(50, 50, 50, 0.9)',
        borderColor: '#333',
        borderWidth: 1,
        padding: [10, 15],
        textStyle: {
          color: '#fff',
          fontSize: 13,
          lineHeight: 20
        },
        formatter: (params: any) => {
          const value = typeof params.value === 'number'
            ? (params.value % 1 === 0 ? params.value : params.value.toFixed(2))
            : (typeof params.value === 'object' && params.value?.value !== undefined)
            ? (typeof params.value.value === 'number' 
              ? (params.value.value % 1 === 0 ? params.value.value : params.value.value.toFixed(2))
              : params.value.value)
            : params.value
          const name = params.seriesName || params.name || ''
          return `<div style="font-weight: bold; margin-bottom: 8px;">${name}</div>
            <div style="display: flex; align-items: center;">
              <span style="flex: 1;">当前值:</span>
              <span style="font-weight: bold; margin-left: 10px;">${value}</span>
            </div>`
        }
      }
      option.series = chart.series.map(s => {
        // gauge 图表的数据格式：单个数值或对象数组
        let gaugeData: any[] = []
        if (s.data.length > 0) {
          const firstItem = s.data[0]
          if (typeof firstItem === 'number') {
            gaugeData = [{ value: firstItem }]
          } else if (typeof firstItem === 'object' && firstItem !== null) {
            gaugeData = [firstItem]
          } else {
            gaugeData = [{ value: parseFloat(String(firstItem)) || 0 }]
          }
        }
        
        // 默认配置
        const defaultConfig: any = {
          name: s.name,
          type: 'gauge',
          data: gaugeData,
          detail: {
            fontSize: 16,
            color: '#1f2937', // 深灰色，提高可读性
            fontWeight: 'bold',
            formatter: '{value}%' // 默认显示百分比
          },
          axisLabel: {
            fontSize: 12,
            color: '#374151' // 深灰色，提高可读性
          }
        }
        
        // 如果 s.config 中有配置，深度合并（特别是 detail 和 axisLabel）
        if (s.config) {
          // 先合并顶层配置
          Object.assign(defaultConfig, s.config)
          
          // 深度合并 detail 配置
          if (s.config.detail) {
            defaultConfig.detail = {
              ...defaultConfig.detail,
              ...s.config.detail
            }
          }
          
          // 深度合并 axisLabel 配置
          if (s.config.axisLabel) {
            defaultConfig.axisLabel = {
              ...defaultConfig.axisLabel,
              ...s.config.axisLabel
            }
          }
        }
        
        return defaultConfig
      })
      break

    default:
      ElMessage.warning(`不支持的图表类型: ${chart.chart_type}`)
      return {}
  }

  return option
}

// 渲染图表
const renderChart = async () => {
  if (!chartContainerRef.value || !chartData.value) return

  const { init } = await loadEChartsRuntime(chartData.value.chart_type)

  if (!chartContainerRef.value || !chartData.value) {
    return
  }

  // 🔥 优化：如果实例已存在且容器未变化，只更新配置，不重新创建实例
  const needRecreate = !chartInstance.value || 
                       !chartContainerRef.value || 
                       chartInstance.value.getDom() !== chartContainerRef.value

  if (needRecreate) {
  // 销毁旧实例
  if (chartInstance.value) {
    chartInstance.value.dispose()
  }

  // 创建新实例（完全按照官方示例）
  chartInstance.value = init(chartContainerRef.value, null, {
    renderer: 'canvas',
    useDirtyRect: false
  })
    // 🔥 优化：减少日志输出，仅在开发环境且需要调试时输出
    if (import.meta.env.DEV && import.meta.env.VITE_DEBUG_CHART) {
  console.log('[ChartRenderer] ECharts 实例已创建:', chartInstance.value)
  console.log('[ChartRenderer] DOM 元素:', chartContainerRef.value)
    }
  } else {
    // 🔥 优化：减少日志输出
    if (import.meta.env.DEV && import.meta.env.VITE_DEBUG_CHART) {
      console.log('[ChartRenderer] 复用现有 ECharts 实例，只更新配置')
    }
  }

  // 构建配置
  const option = buildEChartsOption(chartData.value as RenderableChart)
  
  // 如果配置为空（没有数据），不渲染
  if (!option || Object.keys(option).length === 0) {
    if (chartInstance.value) {
      chartInstance.value.dispose()
      chartInstance.value = null
    }
    return
  }

  // 🔥 优化：减少日志输出，仅在开发环境且需要调试时输出
  if (import.meta.env.DEV && import.meta.env.VITE_DEBUG_CHART) {
  console.log('[ChartRenderer] ECharts option:', JSON.stringify(option, null, 2))
  console.log('[ChartRenderer] tooltip config:', option.tooltip)
  const seriesArray = Array.isArray(option.series) ? option.series : [option.series]
  console.log('[ChartRenderer] series data:', seriesArray.map((s: any) => ({ 
    name: s.name, 
    type: s.type, 
    dataLength: s.data?.length,
    firstDataValue: s.data?.[0],
    firstDataValueType: typeof s.data?.[0],
    sampleData: s.data?.slice(0, 3)
  })))
  }

  // 设置配置（完全按照官方示例，不使用 notMerge）
  const chart = chartInstance.value
  if (!chart) return
  chart.setOption(option)
  
  // 🔥 优化：减少日志输出
  if (import.meta.env.DEV && import.meta.env.VITE_DEBUG_CHART) {
  console.log('[ChartRenderer] ✅ 配置已设置')
  }

  // 响应式调整大小
  window.addEventListener('resize', handleResize)
}

// 处理窗口大小变化
const handleResize = () => {
  if (chartInstance.value) {
    // 直接 resize，不重新设置配置，避免配置丢失
    // ECharts 会自动保持现有配置
    chartInstance.value.resize()
  }
}

// 加载图表数据
const loadChartData = async () => {
  if (!props.functionDetail.router || !props.functionDetail.method) {
    return
  }

  loading.value = true
  try {
    // 构建请求参数
    const params: Record<string, any> = {}
    Object.keys(fieldValues.value).forEach(key => {
      const value = fieldValues.value[key]
      if (value && value.raw !== null && value.raw !== undefined) {
        params[key] = value.raw
      }
    })

    // ⭐ 使用标准 API：/chart/query/{full-code-path}
    const response = await executeFunction(
      props.functionDetail.method,
      props.functionDetail.router,
      params,
      'chart'  // 指定 template_type 为 chart
    )

    // 解析响应数据
    // 后端返回格式：RunFunctionResp.Data() 返回 ChartData，ChartData 结构是 { chart: {...} }
    // 所以最终返回的是 { chart: {...} }，而不是 { chart_data: { chart: {...} } }
    if (response && response.chart) {
      chartData.value = normalizeChartData(response.chart)
      
      // 渲染图表
      await nextTick()
      await renderChart()
    } else {
      chartData.value = requestFields.value.length > 0 ? createPendingQueryChart() : null
    }
  } catch (error: any) {
    ElMessage.error(error?.message || '加载图表数据失败')
    chartData.value = null
  } finally {
    loading.value = false
  }
}

// 查询
const handleSearch = () => {
  loadChartData()
}

// 重置
const handleReset = () => {
  // 重置字段值
  requestFields.value.forEach((field: FieldConfig) => {
    fieldValues.value[field.code] = createEmptyFieldValue(field)
    filterForm.value[field.code] = null
  })
  
  // 清空图表数据
  chartData.value = requestFields.value.length > 0 ? createPendingQueryChart() : null
  if (chartInstance.value) {
    chartInstance.value.dispose()
    chartInstance.value = null
  }
}

// 刷新
const handleRefresh = () => {
  loadChartData()
}


// ResizeObserver 用于监听容器大小变化
let resizeObserver: ResizeObserver | null = null

// 生命周期
onMounted(() => {
  // 初始化字段值
  initializeFieldValues()
  
  // 🔥 开始监听图表筛选条件变化，自动同步到 URL
  watchChartData()
  
  // 自动加载数据（进入页面即加载，无需点击搜索）
  loadChartData()
  
  // 使用 ResizeObserver 监听容器大小变化
  nextTick(() => {
    if (chartContainerRef.value) {
      resizeObserver = new ResizeObserver(() => {
        if (chartInstance.value) {
          chartInstance.value.resize()
        }
      })
      resizeObserver.observe(chartContainerRef.value)
    }
  })
})

onUnmounted(() => {
  
  // 销毁图表实例
  if (chartInstance.value) {
    chartInstance.value.dispose()
    chartInstance.value = null
  }
  
  // 移除窗口大小监听
  window.removeEventListener('resize', handleResize)
  
  // 移除 ResizeObserver
  if (resizeObserver && chartContainerRef.value) {
    resizeObserver.unobserve(chartContainerRef.value)
    resizeObserver.disconnect()
    resizeObserver = null
  }
})

// 🔥 优化：监听 chartData 变化，使用浅层监听 + 手动检查，减少不必要的重新渲染
let lastChartDataHash: string | null = null
watch(() => chartData.value, (newData) => {
  if (!newData) {
    if (chartInstance.value) {
      chartInstance.value.dispose()
      chartInstance.value = null
    }
    lastChartDataHash = null
    return
  }
  
  // 🔥 使用简单的哈希比较，避免深度监听导致的性能问题
  const currentHash = JSON.stringify(newData)
  if (currentHash === lastChartDataHash) {
    return // 数据未变化，跳过渲染
  }
  lastChartDataHash = currentHash
  
    nextTick(() => {
      void renderChart()
    })
}, { flush: 'post' }) // 使用 post 确保在 DOM 更新后执行
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
