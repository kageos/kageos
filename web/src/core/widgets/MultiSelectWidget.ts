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

  /**
   * 🔥 重写 loadFromRawData：正确处理数组类型数据
   */
  static loadFromRawData(rawValue: any, field: FieldConfig): FieldValue {
    // 🔥 如果已经是 FieldValue 格式，直接返回
    if (rawValue && typeof rawValue === 'object' && 'raw' in rawValue && 'display' in rawValue) {
      return rawValue as FieldValue
    }
    
    // 🔥 空值处理：返回默认值（空数组）
    if (rawValue === null || rawValue === undefined || rawValue === '') {
      return this.getDefaultValue(field)
    }
    
    // 🔥 确保是数组类型
    let rawArray: any[] = []
    if (Array.isArray(rawValue)) {
      rawArray = rawValue
    } else if (typeof rawValue === 'string') {
      // 尝试解析 JSON 字符串
      try {
        const parsed = JSON.parse(rawValue)
        if (Array.isArray(parsed)) {
          rawArray = parsed
        } else {
          rawArray = [rawValue]
        }
      } catch {
        // 如果不是 JSON，当作单个值处理
        rawArray = [rawValue]
      }
    } else {
      // 其他类型，转换为数组
      rawArray = [rawValue]
    }
    
    // 🔥 生成 display 文本（使用逗号分隔）
    const displayText = rawArray.length > 0 
      ? rawArray.map(v => String(v)).join(', ')
      : ''
    
    return {
      raw: rawArray,
      display: displayText,
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
   * @param isByValue 是否是按值查询（true: by_value/by_values, false: by_keyword）
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
      // 🔥 判断查询类型：
      // - 如果是按值查询且 query 是数组，使用 by_values
      // - 如果是按值查询且 query 是单个值，使用 by_value
      // - 否则使用 by_keyword
      let queryType: 'by_keyword' | 'by_value' | 'by_values'
      if (isByValue) {
        queryType = Array.isArray(query) ? 'by_values' : 'by_value'
      } else {
        queryType = 'by_keyword'
      }
      
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
    
    // 🔥 生成 display 文本（确保即使没有找到 option 也能显示值）
    const displayText = values.map(val => {
      const option = this.options.value.find((opt: SelectOption) => opt.value === val)
      // 🔥 优先使用 option.label，如果没有则使用值本身
      return option?.label || String(val)
    }).join(', ')
    
    // 🔥 确保 display 文本不为空（即使 values 为空数组，也要有占位文本）
    const finalDisplay = displayText || '未选择'
    
    this.setValue({
      raw: values,  // 🔥 数组（可能是空数组）
      display: finalDisplay,
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
  renderTableCell(value?: FieldValue): any {
    // 🔥 处理 value 为 null/undefined 的情况
    if (!value) {
      return h('span', { style: { color: 'var(--el-text-color-secondary)' } }, '-')
    }
    
    // 🔥 调试日志：检查 value 格式
    if (process.env.NODE_ENV === 'development') {
      console.log('[MultiSelectWidget.renderTableCell]', this.field.code, 'value:', value, 'raw:', value.raw, 'display:', value.display, 'raw type:', typeof value.raw, 'isArray:', Array.isArray(value.raw))
    }
    
    // 🔥 处理 value.raw 为 null/undefined 的情况
    const raw = value.raw
    // 🔥 使用更严格的检查：null、undefined、空字符串都视为未选择
    if (raw === null || raw === undefined || raw === '') {
      return h('span', { style: { color: 'var(--el-text-color-secondary)' } }, '未选择')
    }
    
    const meta = value.meta || {}
    
    // 🔥 确保 raw 是数组（处理 Proxy 对象）
    let rawArray: any[] = []
    if (Array.isArray(raw)) {
      rawArray = raw
    } else if (raw && typeof raw === 'object') {
      // 如果是对象但不是数组，尝试转换
      try {
        rawArray = Array.from(raw as any)
      } catch {
        rawArray = []
      }
    } else {
      // 其他类型，降级处理
      return h('span', String(raw))
    }
    
    // 如果是空数组
    if (rawArray.length === 0) {
      return h('span', { style: { color: 'var(--el-text-color-secondary)' } }, '未选择')
    }
    
    // 🔥 尝试从多个来源获取 labels
    let labels: string[] = []
    
    // 1. 优先从 meta.displayInfo 中提取选项的 label
    if (meta.displayInfo && Array.isArray(meta.displayInfo) && meta.displayInfo.length > 0) {
      const displayInfoLabels = meta.displayInfo.map((info: any) => {
        // 如果 displayInfo 有 label 字段
        if (info && typeof info === 'object' && 'label' in info && info.label != null) {
          return String(info.label)
        }
        // 尝试从字段中提取名称
        if (info && typeof info === 'object') {
          return String(info?.商品名称 || info?.名称 || info?.name || '')
        }
        return ''
      }).filter(label => label && label !== 'null' && label !== 'undefined' && label.length > 0)
      
      // 只有当 displayInfoLabels 长度与 rawArray 匹配且不为空时，才使用它
      if (displayInfoLabels.length === rawArray.length && displayInfoLabels.every(l => l.length > 0)) {
        labels = displayInfoLabels
      }
    }
    
    // 2. 如果没有有效的 labels，尝试使用 display 字段（可能包含逗号分隔的标签）
    if (labels.length === 0 && value.display && typeof value.display === 'string' && value.display.trim() !== '') {
      const displayLabels = value.display.split(',').map(s => s.trim()).filter(s => s.length > 0)
      if (displayLabels.length === rawArray.length) {
        labels = displayLabels
      }
    }
    
    // 3. 如果还是没有有效的 labels，尝试从配置的 options 中查找 label
    if (labels.length === 0) {
      // 🔥 获取配置的 options（从 field.widget.config.options 或 this.selectConfig）
      const configOptions = this.selectConfig?.options || this.field.widget?.config?.options || []
      labels = rawArray.map(val => {
        // 查找配置中的选项
        if (Array.isArray(configOptions) && configOptions.length > 0) {
          // 如果是对象数组 [{ label, value }]
          if (typeof configOptions[0] === 'object' && configOptions[0] !== null) {
            const option = configOptions.find((opt: any) => opt.value === val)
            if (option && option.label) {
              return String(option.label)
            }
          }
          // 如果是字符串数组，直接匹配
          if (typeof configOptions[0] === 'string') {
            const option = configOptions.find((opt: string) => opt === val)
            if (option) {
              return String(option)
            }
          }
        }
        // 回退到显示 raw 值
        return String(val)
      })
    }
    
    // 4. 最后回退：直接显示 raw 值（确保 labels 不为空且长度匹配）
    if (labels.length === 0 || labels.length !== rawArray.length) {
      labels = rawArray.map(v => String(v))
    }
    
    // 🔥 最终验证：确保所有 labels 都是有效的字符串
    labels = labels.map((label, index) => {
      // 如果 label 是 null、undefined 或 'null'、'undefined'，使用 rawArray 中对应的值
      if (!label || label === 'null' || label === 'undefined' || label.trim() === '') {
        return String(rawArray[index] || '')
      }
      return String(label)
    })
    
    // 🔥 调试日志：检查 labels
    if (process.env.NODE_ENV === 'development') {
      console.log('[MultiSelectWidget.renderTableCell]', this.field.code, 'labels:', labels, 'rawArray:', rawArray, 'display:', value.display, 'labels.length:', labels.length)
    }
    
    // 🔥 确保 labels 不为空
    if (!labels || labels.length === 0) {
      console.warn('[MultiSelectWidget.renderTableCell]', this.field.code, 'labels 为空，使用 rawArray')
      labels = rawArray.map(v => String(v))
    }
    
    // 🔥 显示策略：
    // - 如果 ≤ 3 个，全部显示为 Tag
    // - 如果 > 3 个，显示前 3 个 + "等 N 项"
    const maxDisplay = 3
    const displayLabels = labels.slice(0, maxDisplay)
    const hasMore = labels.length > maxDisplay
    
    // 🔥 调试日志：检查最终渲染内容
    if (process.env.NODE_ENV === 'development') {
      console.log('[MultiSelectWidget.renderTableCell]', this.field.code, '最终渲染:', displayLabels, 'hasMore:', hasMore, 'displayLabels.length:', displayLabels.length)
    }
    
    // 🔥 构建 Tag 列表
    const tagNodes = displayLabels.map((label, index) => {
      // 🔥 确保 label 是字符串
      const labelStr = label ? String(label) : String(rawArray[index] || '')
      
      // 🔥 调试日志：检查每个 label
      if (process.env.NODE_ENV === 'development' && index === 0) {
        console.log('[MultiSelectWidget.renderTableCell]', this.field.code, '创建 Tag，labelStr:', labelStr, 'type:', typeof labelStr)
      }
      
      return h(ElTag, { 
        key: `tag-${index}-${labelStr}`,
        size: 'small',
        type: 'info'
      }, { default: () => labelStr })
    })
    
    // 🔥 调试日志：检查 tagNodes
    if (process.env.NODE_ENV === 'development') {
      console.log('[MultiSelectWidget.renderTableCell]', this.field.code, 'tagNodes:', tagNodes, 'tagNodes.length:', tagNodes.length)
    }
    
    // 🔥 构建完整的节点列表
    const children: any[] = [...tagNodes]
    if (hasMore) {
      children.push(h('span', { 
        style: { 
          fontSize: '12px', 
          color: 'var(--el-text-color-secondary)' 
        } 
      }, `等${labels.length}项`))
    }
    
    // 🔥 调试日志：检查最终 children
    if (process.env.NODE_ENV === 'development') {
      console.log('[MultiSelectWidget.renderTableCell]', this.field.code, '最终 children:', children, 'children.length:', children.length)
    }
    
    return h('div', { 
      style: { 
        display: 'flex', 
        gap: '4px', 
        flexWrap: 'wrap',
        alignItems: 'center'
      } 
    }, children)
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
  getCopyText(): string {
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

