/**
 * MultiSelectWidget - 多选组件
 * 用于 data.type = "[]string" 的字段
 * 
 * 与 SelectWidget 的区别：
 * - SelectWidget: 单选，返回单个值（string）
 * - MultiSelectWidget: 多选，返回数组（string[]）
 */

import { h, ref } from 'vue'
import { ElSelect, ElOption, ElTag } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import { Logger } from '../utils/logger'
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

  constructor(props: WidgetRenderProps) {
    super(props)
    
    // 🔥 在构造函数中初始化 ref（避免类属性初始化问题）
    this.options = ref<SelectOption[]>([])
    this.loading = ref(false)
    
    // 解析 MultiSelect 配置
    this.selectConfig = this.getConfig<MultiSelectConfig>()
    
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
      
    }
    
    // 🔥 如果有初始值且有回调，触发一次搜索获取 displayInfo
    if (this.field.callbacks?.includes('OnSelectFuzzy')) {
      const currentValue = this.formManager.getValue(this.fieldPath)
      const currentRaw = currentValue?.raw
      
      // 检查是否有初始值（数组且不为空）
      if (Array.isArray(currentRaw) && currentRaw.length > 0) {
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
      Logger.error(`[MultiSelectWidget] ${this.field.code} 无法获取函数路由，取消回调`)
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


      // 调用回调 API
      const response = await selectFuzzy(method || 'POST', router, requestBody)


      // 解析响应
      if (response.error_msg) {
        Logger.error(`[MultiSelectWidget] ${this.field.code} 回调错误:`, response.error_msg)
        this.options.value = []
        return
      }

      // 🔥 处理 max_selections（动态限制）
      if (response.max_selections !== undefined) {
        this.maxSelections = response.max_selections
      }

      // 🔥 处理 statistics（聚合统计）
      if (response.statistics) {
        this.currentStatistics = response.statistics
      }

      // 更新选项
      this.options.value = (response.items || []).map((item: any) => ({
        label: item.label || item.value,
        value: item.value,
        displayInfo: item.display_info || item.displayInfo,
        icon: item.icon
      }))


    } catch (error) {
      Logger.error(`[MultiSelectWidget] ${this.field.code} 回调失败:`, error)
      this.options.value = []
    } finally {
      this.loading.value = false
    }
  }

  /**
   * 处理选择变更
   */
  private handleChange(values: any[]): void {
    
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
   * 注意：不过滤空字符串，清空关键字时也应该触发查询
   */
  private remoteMethod = (query: string) => {
    // 🔥 不判断 query 是否为空，清空关键字时也要重新加载选项
    this.handleSearch(query, false)
  }

  /**
   * 下拉框展开时触发（点击输入框）
   */
  private handleVisibleChange = (visible: boolean) => {
    if (visible && this.field.callbacks?.includes('OnSelectFuzzy')) {
      // 🔥 展开时，如果选项为空，触发一次空查询加载默认选项
      if (!this.options.value || this.options.value.length === 0) {
        this.handleSearch('', false)  // 空关键词查询
      }
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
    
    // 打印调试信息
    if (multipleLimit > 0) {
    }
    
    return h(ElSelect, {
      modelValue: selectedValues,  // 🔥 数组
      multiple: true,              // 🔥 多选模式
      filterable: true,
      remote: !!this.field.callbacks?.includes('OnSelectFuzzy'),
      remoteMethod: this.remoteMethod,
      loading: this.loading.value,
      placeholder: this.selectConfig.placeholder || `请选择${this.field.name}`,
      multipleLimit: multipleLimit,  // 🔥 限制数量（0表示无限制）
      clearable: true,
      onVisibleChange: this.handleVisibleChange,  // 🔥 下拉框展开/收起时触发
      onChange: (values: any[]) => {
        // 验证数量限制
        if (multipleLimit > 0 && values.length > multipleLimit) {
          Logger.warn(`[MultiSelectWidget] ${this.field.code} 超出数量限制! 限制: ${multipleLimit}, 实际: ${values.length}`)
          // Element Plus 应该会自动限制，但这里做二次验证
          values = values.slice(0, multipleLimit)
        }
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

  /**
   * 🔥 渲染表格单元格（覆盖父类方法）
   * 使用 Tag 标签展示选中的选项
   */
  renderTableCell(value: FieldValue): any {
    if (!value || !value.raw) {
      return h('span', { style: { color: 'var(--el-text-color-secondary)' } }, '-')
    }
    
    const raw = value.raw
    const meta = value.meta || {}
    
    // 如果不是数组，降级处理
    if (!Array.isArray(raw)) {
      return h('span', String(raw))
    }
    
    // 如果是空数组
    if (raw.length === 0) {
      return h('span', { style: { color: 'var(--el-text-color-secondary)' } }, '未选择')
    }
    
    // 🔥 尝试从 meta.displayInfo 中提取选项的 label
    let labels: string[] = []
    
    // displayInfo 可能是数组（MultiSelect 多个选项的 displayInfo）
    if (meta.displayInfo && Array.isArray(meta.displayInfo)) {
      labels = meta.displayInfo.map((info: any) => {
        // 如果 displayInfo 有 label 字段
        if (info && typeof info === 'object' && 'label' in info) {
          return info.label
        }
        // 尝试从字段中提取名称
        return info?.商品名称 || info?.名称 || info?.name || String(info)
      })
    }
    
    // 如果没有 labels，回退到显示 raw 值
    if (labels.length === 0) {
      labels = raw.map(v => String(v))
    }
    
    // 🔥 显示策略：
    // - 如果 ≤ 3 个，全部显示为 Tag
    // - 如果 > 3 个，显示前 3 个 + "等 N 项"
    const maxDisplay = 3
    const displayLabels = labels.slice(0, maxDisplay)
    const hasMore = labels.length > maxDisplay
    
    return h('div', { 
      style: { 
        display: 'flex', 
        gap: '4px', 
        flexWrap: 'wrap',
        alignItems: 'center'
      } 
    }, [
      ...displayLabels.map(label => 
        h(ElTag, { 
          size: 'small',
          type: 'info'
        }, { default: () => label })
      ),
      // 如果有更多项，显示省略标识
      hasMore ? h('span', { 
        style: { 
          fontSize: '12px', 
          color: 'var(--el-text-color-secondary)' 
        } 
      }, `等${labels.length}项`) : null
    ])
  }

  /**
   * 🔥 渲染详情展示（用于 TableRenderer 详情抽屉）
   * 显示多个 Tag（全部显示，不限制数量）
   */
  renderForDetail(value?: FieldValue): any {
    const fieldValue = value || this.safeGetValue(this.fieldPath)
    if (!fieldValue || !fieldValue.raw) {
      return h('span', { style: { color: 'var(--el-text-color-secondary)' } }, '-')
    }
    
    const raw = fieldValue.raw
    const meta = fieldValue.meta || {}
    
    // 如果不是数组，降级处理
    if (!Array.isArray(raw)) {
      return h('span', String(raw))
    }
    
    if (raw.length === 0) {
      return h('span', { style: { color: 'var(--el-text-color-secondary)' } }, '-')
    }
    
    // 尝试从 meta.displayInfo 获取标签
    let labels: string[] = []
    if (meta.displayInfo && Array.isArray(meta.displayInfo)) {
      labels = meta.displayInfo.map((info: any) => {
        if (info && typeof info === 'object' && 'label' in info) {
          return info.label
        }
        return info?.商品名称 || info?.名称 || info?.name || String(info)
      })
    }
    
    // 如果没有 labels，使用 display 值或 raw 值
    if (labels.length === 0) {
      if (fieldValue.display && typeof fieldValue.display === 'string') {
        labels = fieldValue.display.split(',').map(s => s.trim())
      } else {
        labels = raw.map(v => String(v))
      }
    }
    
    // 详情中显示所有标签
    return h('div', { 
      style: { 
        display: 'flex', 
        gap: '4px', 
        flexWrap: 'wrap',
        alignItems: 'center'
      } 
    }, labels.map(label => 
      h(ElTag, { 
        size: 'small',
        type: 'info'
      }, { default: () => label })
    ))
  }

  /**
   * 🔥 获取复制文本
   * 复制 label 列表（逗号分隔）
   */
  onCopy(): string {
    const fieldValue = this.safeGetValue(this.fieldPath)
    if (!fieldValue || !fieldValue.raw) {
      return ''
    }
    
    const raw = fieldValue.raw
    const meta = fieldValue.meta || {}
    
    // 如果不是数组，返回字符串
    if (!Array.isArray(raw)) {
      return String(raw)
    }
    
    if (raw.length === 0) {
      return ''
    }
    
    // 尝试从 meta.displayInfo 获取标签
    let labels: string[] = []
    if (meta.displayInfo && Array.isArray(meta.displayInfo)) {
      labels = meta.displayInfo.map((info: any) => {
        if (info && typeof info === 'object' && 'label' in info) {
          return info.label
        }
        return info?.商品名称 || info?.名称 || info?.name || String(info)
      })
    }
    
    // 如果没有 labels，使用 display 值或 raw 值
    if (labels.length === 0) {
      if (fieldValue.display && typeof fieldValue.display === 'string') {
        labels = fieldValue.display.split(',').map(s => s.trim())
      } else {
        labels = raw.map(v => String(v))
      }
    }
    
    return labels.join(', ')
  }
}

