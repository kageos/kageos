/**
 * SelectWidget - 下拉选择组件
 * 支持搜索、回调、displayInfo、聚合统计
 */

import { h, ref, computed } from 'vue'
import { ElSelect, ElOption, ElMessage } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps } from '../types/widget'
import { selectFuzzy } from '@/api/function'

/**
 * Select 配置
 */
export interface SelectConfig {
  placeholder?: string
  clearable?: boolean
  filterable?: boolean
  remote?: boolean
  multiple?: boolean
  [key: string]: any
}

/**
 * Select 选项
 */
export interface SelectOption {
  label: string
  value: any
  disabled?: boolean
  [key: string]: any
}

/**
 * Select 组件数据（用于快照）
 */
interface SelectComponentData {
  options: SelectOption[]
  loading: boolean
}

export class SelectWidget extends BaseWidget {
  // 选项列表
  private options: any
  
  // 加载状态
  private loading: any
  
  // Select 配置
  private selectConfig: SelectConfig
  
  // 当前聚合统计信息（用于后续聚合计算）
  private currentStatistics: Record<string, string> | null = null

  constructor(props: WidgetRenderProps) {
    super(props)
    
    // 🔥 在构造函数中初始化 ref（避免类属性初始化问题）
    this.options = ref<SelectOption[]>([])
    this.loading = ref(false)
    
    // 解析 Select 配置
    this.selectConfig = (this.field.widget?.config as SelectConfig) || {}
    
    // 初始化选项
    this.initOptions()
  }

  /**
   * 初始化选项
   */
  private initOptions(): void {
    // 从配置中获取初始选项（如果有）
    const initialOptions = this.selectConfig.options
    if (initialOptions && Array.isArray(initialOptions) && initialOptions.length > 0) {
      // 🔥 兼容两种格式：
      // 1. 字符串数组：["低", "中", "高"]
      // 2. 对象数组：[{ label: "低", value: "低" }]
      if (typeof initialOptions[0] === 'string') {
        // 字符串数组 -> SelectOption[]
        this.options.value = (initialOptions as string[]).map(opt => ({
          label: opt,
          value: opt
        }))
      } else {
        // 已经是 SelectOption[] 格式
        this.options.value = initialOptions as SelectOption[]
      }
      
      console.log(`[SelectWidget] ${this.field.code} 初始化选项:`, this.options.value)
    }
    
    // 如果有初始值，触发一次搜索获取 displayInfo
    const currentValue = this.formManager.getValue(this.fieldPath)
    if (currentValue?.raw !== null && currentValue?.raw !== undefined) {
      this.handleSearch('', true) // 静默搜索（by_field_values）
    }
  }

  /**
   * 处理搜索（OnSelectFuzzy 回调）
   * 
   * @param query 搜索值（关键字或实际值）
   * @param isByValue 是否根据值查询
   *   - false: by_keyword - 根据用户输入的关键字模糊搜索（用户主动搜索）
   *   - true: by_value - 根据字段的实际值查询（编辑回显、URL 恢复、初始化默认值）
   */
  private async handleSearch(query: string, isByValue = false): Promise<void> {
    // 🔥 检查是否配置了 OnSelectFuzzy 回调
    const callbacks = this.field.callbacks
    if (!callbacks || !callbacks.includes('OnSelectFuzzy')) {
      console.log(`[SelectWidget] ${this.field.code} 未配置 OnSelectFuzzy 回调，跳过`)
      return
    }

    // 🔥 获取函数的 method 和 router（用于构建回调 URL）
    // Debug: 检查 formRenderer 是否存在
    if (!this.formRenderer) {
      console.error(`[SelectWidget] ${this.field.code} formRenderer 为 undefined，无法调用回调`)
      return
    }
    
    if (!this.formRenderer.getFunctionMethod || !this.formRenderer.getFunctionRouter) {
      console.error(`[SelectWidget] ${this.field.code} formRenderer 不完整:`, {
        hasRegisterWidget: !!this.formRenderer.registerWidget,
        hasGetFunctionMethod: !!this.formRenderer.getFunctionMethod,
        hasGetFunctionRouter: !!this.formRenderer.getFunctionRouter
      })
      return
    }
    
    const method = this.formRenderer.getFunctionMethod()
    const router = this.formRenderer.getFunctionRouter()
    
    if (!router) {
      console.error(`[SelectWidget] ${this.field.code} 无法获取函数路由，取消回调`)
      return
    }

    this.loading.value = true

    try {
      // 🔥 构建回调请求体
      const queryType: 'by_keyword' | 'by_value' = isByValue ? 'by_value' : 'by_keyword'
      const requestBody = {
        code: this.field.code,
        type: queryType,                                // 查询类型
        value: query,                                   // 搜索值
        request: this.formManager.prepareSubmitData(), // 🔥 当前表单的所有字段值
        value_type: this.field.data?.type || 'string'  // 字段类型
      }

      console.log(`[SelectWidget] ${this.field.code} 触发回调`)
      console.log(`[SelectWidget]   Query Type: ${requestBody.type}`)
      console.log(`[SelectWidget]   Search Value:`, query)

      // 🔥 调用回调接口
      const response = await selectFuzzy(method, router, requestBody)
      
      // 🔥 Debug: 查看完整响应
      console.log(`[SelectWidget] ${this.field.code} 回调响应:`, response)

      // 🔥 解析响应（axios 拦截器已返回 data，无需再访问 .data）
      const { items, error_msg, statistics } = response || {}

      // 检查错误信息
      if (error_msg) {
        ElMessage.error(error_msg)
        this.options.value = []
        return
      }

      // 🔥 更新选项列表
      if (items && Array.isArray(items)) {
        this.options.value = items.map((item: any) => ({
          label: item.label || String(item.value),
          value: item.value,
          disabled: false,
          displayInfo: item.display_info,  // 额外展示信息
          icon: item.icon                  // 图标（可选）
        }))

        console.log(`[SelectWidget] ${this.field.code} 查询成功，共 ${items.length} 个选项`)
      } else {
        this.options.value = []
        console.log(`[SelectWidget] ${this.field.code} 查询结果为空`)
      }

      // 🔥 保存聚合统计信息（后续用于聚合计算）
      if (statistics && typeof statistics === 'object') {
        this.currentStatistics = statistics
        console.log(`[SelectWidget] ${this.field.code} 收到聚合统计:`, statistics)
      }

    } catch (error: any) {
      console.error(`[SelectWidget] ${this.field.code} 回调失败:`, error)
      ElMessage.error(error?.message || '查询失败')
      this.options.value = []
    } finally {
      this.loading.value = false
    }
  }

  /**
   * 处理值变化
   * 保存选中项的 displayInfo 和聚合统计信息
   */
  private handleChange(value: any): void {
    // 🔥 查找选中项的 displayInfo
    const selectedOption = this.options.value.find((opt: SelectOption) => opt.value === value)
    const displayValue = selectedOption?.label || String(value)
    
    // 🔥 构建 meta 信息
    const meta: any = {
      displayInfo: selectedOption?.displayInfo || null  // 选项的额外展示信息
    }
    
    // 🔥 保存聚合统计信息（如果有）
    if (this.currentStatistics) {
      meta.statistics = this.currentStatistics
    }
    
    // 更新 FieldValue
    const newFieldValue: FieldValue = {
      raw: value,
      display: displayValue,
      meta
    }
    
    // 🔥 更新值（使用 BaseWidget 的 setValue 方法）
    this.setValue(newFieldValue)
    
    console.log(`[SelectWidget] ${this.field.code} 值变化:`, {
      field_path: this.fieldPath,
      raw: value,
      display: displayValue,
      has_displayInfo: !!meta.displayInfo,
      has_statistics: !!meta.statistics
    })
  }

  /**
   * 渲染组件
   */
  render() {
    const currentValue = this.getValue()
    
    return h(ElSelect, {
      modelValue: currentValue?.raw,
      placeholder: this.selectConfig.placeholder || `请选择${this.field.name}`,
      clearable: this.selectConfig.clearable !== false,
      filterable: this.selectConfig.filterable !== false,
      remote: true,
      remoteMethod: (query: string) => this.handleSearch(query, false),
      loading: this.loading.value,
      onChange: (value: any) => this.handleChange(value),
      style: { width: '100%' }
    }, {
      default: () => (this.options.value || []).map((option: SelectOption) => 
        h(ElOption, {
          key: option.value,
          label: option.label,
          value: option.value,
          disabled: option.disabled
        })
      )
    })
  }

  /**
   * 捕获组件数据（用于快照）
   */
  protected captureComponentData(): SelectComponentData {
    return {
      options: this.options.value,
      loading: this.loading.value
    }
  }

  /**
   * 恢复组件数据（从快照）
   */
  protected restoreComponentData(data: SelectComponentData): void {
    if (data.options) {
      this.options.value = data.options
    }
  }
}

