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
      <el-card class="filter-card">
        <template #header>
          <span>筛选条件</span>
        </template>
        <el-form :inline="true" :model="filterForm" class="filter-form">
          <el-form-item
            v-for="field in requestFields"
            :key="field.code"
            :label="field.name"
            :required="isFieldRequired(field)"
          >
            <div class="widget-wrapper">
              <WidgetComponent
                :field="field"
                :value="getFieldValue(field.code)"
                :model-value="getFieldValue(field.code)"
                @update:model-value="(v) => handleFieldUpdate(field.code, v)"
                :field-path="field.code"
                :form-renderer="formRendererContext"
                :function-method="props.functionDetail.method || 'GET'"
                :function-router="props.functionDetail.router || ''"
                mode="edit"
              />
            </div>
          </el-form-item>
          
          <el-form-item>
            <el-button type="primary" @click="handleSearch" :loading="loading" :icon="Search">
              查询
            </el-button>
            <el-button @click="handleReset" :icon="Refresh">
              重置
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>
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
import * as echarts from 'echarts'
import type { EChartsOption } from 'echarts'
import WidgetComponent from '@/architecture/presentation/widgets/WidgetComponent.vue'
import { executeFunction } from '@/api/function'
import type { FieldConfig, FieldValue, FunctionDetail } from '@/core/types/field'
import type { Chart, ChartSeries } from '@/core/types/chart'
import { widgetComponentFactory } from '@/core/factories-v2'
import { hasAnyRequiredRule } from '@/core/utils/validationUtils'
import { convertToFieldValue } from '@/utils/field'
import { useChartParamURLSync } from '@/architecture/presentation/composables/useChartParamURLSync'
import { convertValueByFieldType } from '@/core/widgets-v2/utils/typeConverter'

const props = defineProps<{
  functionDetail: FunctionDetail
}>()

const route = useRoute()

// 状态
const loading = ref(false)
const chartData = ref<Chart | null>(null)
const chartContainerRef = ref<HTMLElement | null>(null)
const chartInstance = ref<echarts.ECharts | null>(null)
const chartHeight = ref('600px')

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

// 筛选表单数据
const filterForm = ref<Record<string, any>>({})

// 字段值
const fieldValues = ref<Record<string, FieldValue>>({})

// 初始化字段值
const initializeFieldValues = () => {
  const values: Record<string, FieldValue> = {}
  requestFields.value.forEach((field: FieldConfig) => {
    // 从 URL 查询参数中获取初始值
    const queryValue = route.query[field.code]
    const value = Array.isArray(queryValue) ? queryValue[0] : queryValue
    
    if (value !== undefined && value !== null && value !== '') {
      // 🔥 使用统一的类型转换工具
      const rawValue = convertValueByFieldType(value, field)
      values[field.code] = convertToFieldValue(rawValue, field)
      filterForm.value[field.code] = rawValue
    } else {
      values[field.code] = { raw: null, display: '', meta: {} }
      filterForm.value[field.code] = null
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
  return fieldValues.value[fieldCode] || { raw: null, display: '', meta: {} }
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

// 构建 ECharts 配置
const buildEChartsOption = (chart: Chart): EChartsOption => {
  // 先检查数据是否有效
  if (!chart || !chart.series || chart.series.length === 0) {
    return {}
  }
  
  const option: EChartsOption = {
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
        trigger: 'axis',
        axisPointer: {
          type: 'shadow'
        },
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
          // params 是数组，包含所有系列在该点的数据
          if (!Array.isArray(params) || params.length === 0) {
            return '无数据'
          }
          
          let result = `<div style="font-weight: bold; margin-bottom: 8px;">${params[0].axisValue || ''}</div>`
          
          params.forEach((param: any) => {
            const value = typeof param.value === 'number'
              ? (param.value % 1 === 0 ? param.value : param.value.toFixed(2))
              : param.value
            const name = param.seriesName || param.name || ''
            const color = param.color || '#5470c6'
            
            result += `<div style="display: flex; align-items: center; margin-bottom: 4px;">
              <span style="display: inline-block; width: 10px; height: 10px; background-color: ${color}; border-radius: 50%; margin-right: 8px;"></span>
              <span style="flex: 1;">${name}:</span>
              <span style="font-weight: bold; margin-left: 10px;">${value}</span>
            </div>`
          })
          
          return result
        }
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
        trigger: 'axis',
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
          // params 是数组，包含所有系列在该点的数据
          if (!Array.isArray(params) || params.length === 0) {
            return '无数据'
          }
          
          let result = `<div style="font-weight: bold; margin-bottom: 8px;">${params[0].axisValue || ''}</div>`
          
          params.forEach((param: any) => {
            const value = typeof param.value === 'number'
              ? (param.value % 1 === 0 ? param.value : param.value.toFixed(2))
              : param.value
            const name = param.seriesName || param.name || ''
            const color = param.color || '#5470c6'
            
            result += `<div style="display: flex; align-items: center; margin-bottom: 4px;">
              <span style="display: inline-block; width: 10px; height: 10px; background-color: ${color}; border-radius: 50%; margin-right: 8px;"></span>
              <span style="flex: 1;">${name}:</span>
              <span style="font-weight: bold; margin-left: 10px;">${value}</span>
            </div>`
          })
          
          return result
        }
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
        ...option.tooltip,
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

    case 'scatter':
      option.xAxis = {
        type: 'value',
        axisLabel: {
          fontSize: 12,
          color: '#374151' // 深灰色，提高可读性
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
      option.yAxis = {
        type: 'value',
        axisLabel: {
          fontSize: 12,
          color: '#374151' // 深灰色，提高可读性
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
        type: 'scatter',
        data: s.data,
        symbolSize: 8, // 增加点的大小
        ...s.config
      }))
      break

    case 'area':
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
          color: '#374151' // 深灰色，提高可读性
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
        areaStyle: {},
        data: s.data,
        lineWidth: 2.5, // 增加线宽，更清晰
        symbol: 'circle',
        symbolSize: 6, // 数据点大小
        label: {
          show: false // 默认不显示标签
        },
        ...s.config
      }))
      break

    default:
      ElMessage.warning(`不支持的图表类型: ${chart.chart_type}`)
      return {}
  }

  // 合并自定义 ECharts 配置
  // 注意：tooltip 配置已经在 switch case 中设置，如果 echarts_config 中有 tooltip，需要保护我们的配置
  if (chart.echarts_config) {
    const echartsConfig = { ...chart.echarts_config }
    // 如果 echarts_config 中有 tooltip，删除它，使用我们在 switch case 中设置的 tooltip
    if (echartsConfig.tooltip) {
      console.log('[ChartRenderer] 警告：echarts_config 中有 tooltip，将被忽略')
      delete echartsConfig.tooltip
    }
    // 使用深度合并，但保护 tooltip 配置
    // 先合并其他配置，然后确保 tooltip 不被覆盖
    const savedTooltip = option.tooltip
    Object.assign(option, echartsConfig)
    if (savedTooltip) {
      option.tooltip = savedTooltip
    }
    console.log('[ChartRenderer] 合并 echarts_config 后，tooltip:', option.tooltip)
  }

  return option
}

// 渲染图表
const renderChart = () => {
  if (!chartContainerRef.value || !chartData.value) return

  // 检查是否有数据
  const hasData = chartData.value.series && chartData.value.series.length > 0 && 
    chartData.value.series.some(s => s.data && s.data.length > 0)
  
  if (!hasData) {
    // 如果没有数据，销毁旧实例并返回
    if (chartInstance.value) {
      chartInstance.value.dispose()
      chartInstance.value = null
    }
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
  chartInstance.value = echarts.init(chartContainerRef.value, null, {
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
  const option = buildEChartsOption(chartData.value)
  
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
  chartInstance.value.setOption(option)
  
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
      chartData.value = response.chart
      
      // 渲染图表
      await nextTick()
      renderChart()
    } else {
      console.error('返回数据格式不正确，响应数据：', response)
      ElMessage.warning('返回数据格式不正确')
      chartData.value = null
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
    fieldValues.value[field.code] = { raw: null, display: '', meta: {} }
    filterForm.value[field.code] = null
  })
  
  // 清空图表数据
  chartData.value = null
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
      renderChart()
    })
}, { flush: 'post' }) // 使用 post 确保在 DOM 更新后执行
</script>

<style scoped lang="scss">
.chart-renderer {
  width: 100%;
  padding: 20px;
  
  .filter-card {
    margin-bottom: 20px;
    
    .widget-wrapper {
      min-width: 200px; // 设置下拉框最小宽度
      width: 100%;
      
      // 确保下拉框可以清空
      :deep(.el-select) {
        width: 100%;
      }
    }
    
    .filter-form {
      margin-top: 0;
    }
  }
  
  .chart-card {
    margin-bottom: 20px;
    
    .chart-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      
      .chart-actions {
        display: flex;
        gap: 10px;
      }
    }
    
    .chart-container {
      width: 100%;
      min-height: 400px;
      
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
    margin-top: 20px;
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

