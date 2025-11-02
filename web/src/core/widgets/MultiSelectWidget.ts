/**
 * MultiSelectWidget - 多选组件
 * 用于 data.type = "[]string" 的字段
 * 
 * 与 SelectWidget 的区别：
 * - SelectWidget: 单选，返回单个值（string）
 * - MultiSelectWidget: 多选，返回数组（string[]）
 */

import { h, ref } from 'vue'
import { ElSelect, ElOption } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import { selectFuzzy } from '@/api/function'
import type { FieldConfig, FieldValue } from '../types/field'

/**
 * MultiSelect 配置
 */
interface MultiSelectConfig {
  options?: string[] | Array<{ label: string; value: any }>
  placeholder?: string
  creatable?: boolean  // 是否可创建新选项
  max_count?: number   // 最大选择数量（静态配置）
  default?: any[]
}

/**
 * 选项接口
 */
interface SelectOption {
  label: string
  value: any
  displayInfo?: Record<string, any>  // 显示信息
  icon?: string
}

export class MultiSelectWidget extends BaseWidget {
  private selectConfig: MultiSelectConfig
  private options: any
  private loading: any
  private maxSelections: number | null = null  // 动态限制（从回调获取）
  private currentStatistics: Record<string, string> | null = null

  /**
   * MultiSelectWidget 的默认值是空数组
   */
  static getDefaultValue(field: FieldConfig): FieldValue {
    return {
      raw: [],
      display: '',
      meta: {}
    }
  }

  constructor(props: any) {
    super(props)
    
    // 🔥 在构造函数中初始化 ref（避免类属性初始化问题）
    this.options = ref<SelectOption[]>([])
    this.loading = ref(false)
    
    // 解析 MultiSelect 配置
    this.selectConfig = (this.field.widget?.config as MultiSelectConfig) || {}
    
    // 初始化选项
    this.initOptions()
  }

  /**
   * 初始化选项
   */
  private initOptions(): void {
    const configOptions = this.selectConfig.options
    
    if (configOptions && Array.isArray(configOptions)) {
      // 🔥 处理两种格式：字符串数组 或 对象数组
      if (configOptions.length > 0 && typeof configOptions[0] === 'string') {
        // 字符串数组：["加急", "重要", "普通"]
        this.options.value = (configOptions as string[]).map(opt => ({
          label: opt,
          value: opt
        }))
      } else {
        // 对象数组：[{ label: "...", value: "..." }]
        this.options.value = configOptions as SelectOption[]
      }
      
      console.log(`[MultiSelectWidget] ${this.field.code} 初始化选项:`, this.options.value)
    }
    
    // 🔥 如果有初始值且有回调，触发一次搜索获取 displayInfo
    if (this.field.callbacks?.includes('OnSelectFuzzy')) {
      const currentValue = this.formManager.getValue(this.fieldPath)
      const currentRaw = currentValue?.raw
      
      // 检查是否有初始值（数组且不为空）
      if (Array.isArray(currentRaw) && currentRaw.length > 0) {
        console.log(`[MultiSelectWidget] ${this.field.code} 检测到初始值，触发回调获取 displayInfo`)
        this.handleSearch(currentRaw, true) // 静默搜索（by_value）
      }
    }
  }

  /**
   * 处理搜索（OnSelectFuzzy 回调）
   * @param query 搜索关键词或值（可以是字符串或数组）
   * @param isByValue 是否是按值查询（true: by_value, false: by_keyword）
   */
  private async handleSearch(query: string | any[], isByValue = false): Promise<void> {
    // 如果没有回调，不处理
    if (!this.field.callbacks?.includes('OnSelectFuzzy')) {
      return
    }
    
    const method = this.formRenderer?.getFunctionMethod?.()
    const router = this.formRenderer?.getFunctionRouter?.()
    
    if (!router) {
      console.error(`[MultiSelectWidget] ${this.field.code} 无法获取函数路由，取消回调`)
      return
    }

    this.loading.value = true

    try {
      // 🔥 构建回调请求体
      const queryType: 'by_keyword' | 'by_value' = isByValue ? 'by_value' : 'by_keyword'
      const requestBody = {
        code: this.field.code,
        type: queryType,
        value: query,
        request: this.formRenderer.getSubmitData?.() || {},
        value_type: this.field.data?.type || '[]string'
      }

      console.log(`[MultiSelectWidget] ${this.field.code} 触发回调`)
      console.log(`[MultiSelectWidget]   Query Type: ${requestBody.type}`)
      console.log(`[MultiSelectWidget]   Search Value:`, query)

      // 调用回调 API
      const response = await selectFuzzy(method || 'POST', router, requestBody)

      console.log(`[MultiSelectWidget] ${this.field.code} 回调响应:`, response)

      // 解析响应
      if (response.error_msg) {
        console.error(`[MultiSelectWidget] ${this.field.code} 回调错误:`, response.error_msg)
        this.options.value = []
        return
      }

      // 🔥 处理 max_selections（动态限制）
      if (response.max_selections !== undefined) {
        this.maxSelections = response.max_selections
        console.log(`[MultiSelectWidget] ${this.field.code} 动态限制最多选择: ${this.maxSelections}`)
      }

      // 🔥 处理 statistics（聚合统计）
      if (response.statistics) {
        this.currentStatistics = response.statistics
        console.log(`[MultiSelectWidget] ${this.field.code} 收到聚合统计:`, this.currentStatistics)
      }

      // 更新选项
      this.options.value = (response.items || []).map((item: any) => ({
        label: item.label || item.value,
        value: item.value,
        displayInfo: item.display_info || item.displayInfo,
        icon: item.icon
      }))

      console.log(`[MultiSelectWidget] ${this.field.code} 查询成功，共 ${this.options.value.length} 个选项`)

    } catch (error) {
      console.error(`[MultiSelectWidget] ${this.field.code} 回调失败:`, error)
      this.options.value = []
    } finally {
      this.loading.value = false
    }
  }

  /**
   * 处理选择变更
   */
  private handleChange(values: any[]): void {
    console.log(`[MultiSelectWidget] ${this.field.code} 选择变更:`, values)
    
    // 🔥 收集多个值的 displayInfo
    const displayInfos = values.map(val => {
      const option = this.options.value.find((opt: SelectOption) => opt.value === val)
      return option?.displayInfo || null
    })
    
    // 🔥 生成 display 文本
    const displayText = values.map(val => {
      const option = this.options.value.find((opt: SelectOption) => opt.value === val)
      return option?.label || val
    }).join(', ')
    
    this.setValue({
      raw: values,  // 🔥 数组
      display: displayText,
      meta: {
        displayInfo: displayInfos,  // 🔥 数组
        statistics: this.currentStatistics
      }
    })
  }

  /**
   * 远程搜索方法
   */
  private remoteMethod = (query: string) => {
    if (query) {
      this.handleSearch(query, false)
    }
  }

  /**
   * 🔥 重写：返回数组
   */
  getRawValueForSubmit(): any[] {
    const raw = this.value.value.raw
    // 确保返回数组
    return Array.isArray(raw) ? raw : []
  }

  render() {
    const currentValue = this.getValue()
    const selectedValues = Array.isArray(currentValue?.raw) ? currentValue.raw : []
    
    // 🔥 计算最大选择数量（优先使用动态限制）
    const multipleLimit = this.maxSelections || this.selectConfig.max_count || 0
    
    return h(ElSelect, {
      modelValue: selectedValues,  // 🔥 数组
      multiple: true,              // 🔥 多选模式
      filterable: true,
      remote: !!this.field.callbacks?.includes('OnSelectFuzzy'),
      remoteMethod: this.remoteMethod,
      loading: this.loading.value,
      placeholder: this.selectConfig.placeholder || `请选择${this.field.name}`,
      multipleLimit: multipleLimit,  // 🔥 限制数量
      clearable: true,
      onChange: (values: any[]) => {
        this.handleChange(values)
      }
    }, {
      default: () => (this.options.value || []).map((option: SelectOption) => {
        return h(ElOption, {
          key: option.value,
          label: option.label,
          value: option.value
        })
      })
    })
  }
}

