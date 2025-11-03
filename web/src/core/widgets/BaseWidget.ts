/**
 * BaseWidget - 所有 Widget 的基类
 */

import { ref, type Ref } from 'vue'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps, WidgetSnapshot, FormRendererContext } from '../types/widget'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'
import { Logger } from '../utils/logger'

/**
 * Widget 快照接口
 */
export interface IWidgetSnapshot {
  captureSnapshot(): WidgetSnapshot
  restoreSnapshot(snapshot: WidgetSnapshot): void
}

/**
 * BaseWidget 基类
 */
export abstract class BaseWidget implements IWidgetSnapshot {
  protected field: FieldConfig
  protected fieldPath: string
  protected fieldCode: string
  protected value: Ref<FieldValue>
  protected formManager: ReactiveFormDataManager | null  // ✅ 类型诚实
  protected formRenderer: FormRendererContext | null  // ✅ 类型安全
  protected depth: number
  protected onChange: (newValue: FieldValue) => void

  // 最大嵌套深度
  protected static readonly MAX_DEPTH = 10

  /**
   * ✅ 辅助属性：是否是临时 Widget
   * 临时 Widget 没有 formManager，用于只读渲染（表格单元格、搜索输入配置等）
   */
  protected get isTemporary(): boolean {
    return this.formManager === null
  }

  /**
   * ✅ 辅助属性：是否有 formManager
   * 语义更清晰的检查方式
   */
  protected get hasFormManager(): boolean {
    return this.formManager !== null
  }

  /**
   * ✅ 安全获取值
   * 如果是临时 Widget，返回当前 value；否则从 formManager 读取
   */
  protected safeGetValue(fieldPath?: string): FieldValue {
    if (!this.formManager) {
      return this.value.value
    }
    return this.formManager.getValue(fieldPath || this.fieldPath)
  }

  /**
   * ✅ 安全设置值
   * 如果是临时 Widget，不做任何操作；否则写入 formManager
   */
  protected safeSetValue(fieldPath: string, value: FieldValue): void {
    if (!this.formManager) {
      return  // 临时 Widget 不需要设置值
    }
    this.formManager.setValue(fieldPath, value)
  }

  /**
   * ✅ 要求 formManager 存在（用于必需 formManager 的操作）
   * 如果是临时 Widget 却调用了需要 formManager 的方法，抛出清晰的错误
   */
  protected requireFormManager(operation: string): ReactiveFormDataManager {
    if (!this.formManager) {
      throw new Error(`[${this.constructor.name}] ${operation} requires formManager, but this is a temporary widget`)
    }
    return this.formManager
  }

  /**
   * ✅ 获取配置（类型安全的配置提取）
   * 避免每个子类都要写 (this.field.widget?.config as XxxConfig) || {}
   */
  protected getConfig<T = any>(): T {
    return (this.field.widget?.config as T) || {} as T
  }

  /**
   * 获取字段的默认值
   * 每个 Widget 子类可以重写此方法来提供自定义的默认值逻辑
   */
  static getDefaultValue(field: FieldConfig): FieldValue {
    // 1. 优先使用 widget.config.default
    if (field.widget?.config && typeof field.widget.config === 'object' && field.widget.config.default !== undefined && field.widget.config.default !== '') {
      return {
        raw: field.widget.config.default,
        display: String(field.widget.config.default),
        meta: {}
      }
    }

    // 2. 根据字段类型设置默认值
    const fieldType = field.data?.type || 'string'
    
    switch (fieldType.toLowerCase()) {
      case 'int':
      case 'float':
      case 'number':
        return {
          raw: undefined,
          display: '',
          meta: {}
        }
      case 'bool':
        return {
          raw: false,
          display: '否',
          meta: {}
        }
      case 'array':
      case '[]struct':
        return {
          raw: [],
          display: '[]',
          meta: {}
        }
      case 'struct':
        return {
          raw: {},
          display: '{}',
          meta: {}
        }
      default:
        return {
          raw: '',
          display: '',
          meta: {}
        }
    }
  }

  /**
   * 🔥 从原始数据加载为 FieldValue 格式（静态方法，支持多态）
   * 
   * 每个组件负责自己的数据转换逻辑：
   * - 基础组件（Input/Select/Number 等）：直接转换
   * - 容器组件（Table/Form 等）：递归调用子组件的 loadFromRawData()
   * 
   * 这样符合开闭原则：新增组件类型不需要修改已有代码
   * 
   * @param rawValue 原始数据（可能来自后端、父组件、缓存等）
   * @param field 字段配置
   * @returns FieldValue 格式的数据
   */
  static loadFromRawData(rawValue: any, field: FieldConfig): FieldValue {
    // 🔥 如果已经是 FieldValue 格式，直接返回
    if (rawValue && typeof rawValue === 'object' && 'raw' in rawValue && 'display' in rawValue) {
      return rawValue
    }
    
    // 🔥 空值处理：返回默认值
    if (rawValue === null || rawValue === undefined) {
      return this.getDefaultValue(field)
    }
    
    // 🔥 基础类型：直接转换
    return {
      raw: rawValue,
      display: rawValue !== null && rawValue !== undefined ? String(rawValue) : '',
      meta: {}
    }
  }

  constructor(props: WidgetRenderProps) {
    this.field = props.field
    this.fieldPath = props.currentFieldPath
    this.fieldCode = props.field.code
    this.value = ref(props.value)
    this.formManager = props.formManager
    this.formRenderer = props.formRenderer
    this.depth = props.depth || 0
    this.onChange = props.onChange

    // 深度检查
    if (this.depth > BaseWidget.MAX_DEPTH) {
      Logger.error('BaseWidget', `嵌套深度超过限制: ${this.depth}，字段: ${this.fieldPath}`)
      throw new Error(`最大嵌套深度为 ${BaseWidget.MAX_DEPTH}`)
    }
  }

  /**
   * 获取当前值
   */
  protected getValue(): FieldValue {
    const value = this.value.value
    // 如果值不存在，返回默认值
    if (!value) {
      return {
        raw: '',
        display: '',
        meta: {}
      }
    }
    return value
  }
  
  /**
   * ✅ 获取当前值（用于提交，公开方法）
   * 注意：这个方法名和上面的 protected getValue 不同
   */
  getRawValueForSubmit(): any {
    return this.getValue().raw
  }

  /**
   * 设置值
   */
  protected setValue(newValue: FieldValue): void {
    this.value.value = newValue
    this.onChange(newValue)
    
    // ✅ 同步到 formManager（如果存在）
    if (this.formManager) {
      this.formManager.setValue(this.fieldPath, newValue)
    }
  }

  /**
   * 根据字段类型转换值
   */
  protected convertValueByType(value: any): any {
    // 🔥 空值统一返回 null（后端可以正确处理 null，但不能处理空字符串转数字）
    if (value === null || value === undefined || value === '') {
      return null
    }
    
    // 🔥 获取字段类型：优先使用 data.type，如果为空则使用 widget.type
    let fieldType = this.field.data?.type || ''
    if (!fieldType || fieldType.trim() === '') {
      fieldType = this.field.widget?.type || 'string'
    }
    
    // 根据类型转换
    switch (fieldType.toLowerCase()) {
      case 'int':
      case 'integer':
      case 'number':  // 🔥 widget.type 可能是 'number'
        const intValue = Number(value)
        return isNaN(intValue) ? null : intValue  // 🔥 转换失败返回 null
      
      case 'float':
      case 'double':
        const floatValue = Number(value)
        return isNaN(floatValue) ? null : floatValue  // 🔥 转换失败返回 null
      
      case 'bool':
      case 'boolean':
      case 'switch':  // 🔥 widget.type 可能是 'switch'
        if (typeof value === 'boolean') return value
        if (typeof value === 'string') {
          const lower = value.toLowerCase()
          return lower === 'true' || lower === '1' || lower === 'yes'
        }
        return Boolean(value)
      
      case 'string':
      case 'input':  // 🔥 widget.type 可能是 'input'
      case 'text':
      case 'textarea':
      case 'text_area':
      default:
        // 🔥 字符串类型：空值返回 null，有值返回字符串
        return value ? String(value) : null
    }
  }

  /**
   * 获取用于提交的原始值（已转换类型）
   */
  getRawValueForSubmit(): any {
    const raw = this.value.value.raw
    
    // 🔥 获取字段类型：优先使用 data.type，如果为空则使用 widget.type
    let fieldType = this.field.data?.type || ''
    if (!fieldType || fieldType.trim() === '') {
      fieldType = this.field.widget?.type || 'string'
    }
    
    // 对于嵌套结构（List/Struct），不做类型转换（由子组件处理）
    if (fieldType.includes('[]') || fieldType === 'struct' || 
        fieldType === 'table' || fieldType === 'form') {
      return raw
    }
    
    // 对于基础类型，转换类型
    return this.convertValueByType(raw)
  }

  /**
   * 更新原始值（保留 display 和 meta，自动类型转换）
   */
  protected updateRawValue(raw: any): void {
    // 转换类型（对于基础类型）
    const fieldType = this.field.data?.type || 'string'
    let convertedRaw = raw
    
    // 只有基础类型才转换，嵌套结构由子组件处理
    if (!fieldType.includes('[]') && fieldType !== 'struct') {
      convertedRaw = this.convertValueByType(raw)
    }
    
    this.setValue({
      ...this.value.value,
      raw: convertedRaw,
      display: String(raw)  // display 保持原样（用于显示）
    })
  }

  /**
   * 🔥 渲染表格单元格（用于 TableWidget）
   * 子类可以覆盖此方法来自定义表格展示
   * @param value 字段值
   * @returns VNode（Vue 虚拟节点）或 字符串
   * 
   * 注意：为了兼容 TableRenderer，如果返回字符串，TableRenderer 会用 span 包裹
   * 子类如果要返回 VNode，可以直接返回 h(...)
   */
  renderTableCell(value: FieldValue): any {
    // 需要导入 h，但为了保持简洁，返回字符串，由 TableRenderer 处理
    if (!value) return '-'
    
    // 🔥 优先使用 display 属性
    if (value.display) {
      return value.display
    }
    
    // 降级：格式化 raw 值
    const raw = value.raw
    if (raw === null || raw === undefined) return '-'
    
    // 根据字段类型格式化
    if (this.field.widget?.type === 'timestamp') {
      return this.formatTimestamp(raw)
    }
    
    if (Array.isArray(raw)) {
      return raw.join(', ')
    }
    
    return String(raw)
  }

  /**
   * 格式化时间戳（用于表格显示）
   */
  protected formatTimestamp(timestamp: number | string): string {
    if (!timestamp) return '-'
    
    const date = typeof timestamp === 'number' 
      ? new Date(timestamp * 1000)  // Unix 时间戳（秒）
      : new Date(timestamp)
    
    if (isNaN(date.getTime())) return String(timestamp)
    
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hours = String(date.getHours()).padStart(2, '0')
    const minutes = String(date.getMinutes()).padStart(2, '0')
    
    return `${year}-${month}-${day} ${hours}:${minutes}`
  }

  /**
   * 🔥 渲染搜索输入框（用于 TableRenderer）
   * 子类可以覆盖此方法来自定义搜索输入
   * @param searchType 搜索类型，如 'eq', 'like', 'gte,lte', 'in'
   * @returns VNode（Vue 虚拟节点）
   */
  renderSearchInput(searchType: string): any {
    // 根据搜索类型返回不同的输入组件
    if (searchType.includes('eq')) {
      return this.renderExactSearch()
    }
    if (searchType.includes('like')) {
      return this.renderFuzzySearch()
    }
    if (searchType.includes('gte') && searchType.includes('lte')) {
      return this.renderRangeSearch()
    }
    if (searchType.includes('in')) {
      return this.renderInSearch()
    }
    
    // 默认：精确搜索
    return this.renderExactSearch()
  }

  /**
   * 渲染精确搜索输入框（eq）
   * 子类可以覆盖
   */
  protected renderExactSearch(): any {
    // 默认实现：返回配置对象，由 TableRenderer 渲染
    return {
      component: 'ElInput',
      props: {
        placeholder: `请输入${this.field.name}`,
        clearable: true,
        style: { width: '200px' }
      }
    }
  }

  /**
   * 渲染模糊搜索输入框（like）
   * 子类可以覆盖
   */
  protected renderFuzzySearch(): any {
    // 默认实现：和精确搜索一样
    return {
      component: 'ElInput',
      props: {
        placeholder: `请输入${this.field.name}`,
        clearable: true,
        style: { width: '200px' }
      }
    }
  }

  /**
   * 渲染范围搜索输入框（gte, lte）
   * 子类应该覆盖此方法以提供类型特定的范围输入
   */
  protected renderRangeSearch(): any {
    // 默认实现：两个文本输入框
    return {
      component: 'RangeInput',
      props: {
        minPlaceholder: `最小${this.field.name}`,
        maxPlaceholder: `最大${this.field.name}`,
        inputType: 'text'
      }
    }
  }

  /**
   * 渲染包含搜索输入框（in）
   * 子类可以覆盖
   */
  protected renderInSearch(): any {
    // 默认实现：下拉选择（如果有 options）
    const options = this.field.widget?.config?.options || []
    
    return {
      component: 'ElSelect',
      props: {
        placeholder: `请选择${this.field.name}`,
        clearable: true,
        style: { width: '200px' },
        options: options
      }
    }
  }

  /**
   * 🔥 发出事件
   * @param eventType 事件类型，如 'field:search', 'field:change'
   * @param payload 事件数据
   */
  protected emit(eventType: string, payload: any = {}): void {
    // ✅ 如果是临时 Widget，不发射事件
    if (!this.formManager) {
      return
    }
    
    // 自动添加 fieldPath 到 payload
    const fullPayload = {
      ...payload,
      fieldPath: this.fieldPath,
      fieldCode: this.fieldCode
    }
    
    // 构建完整的事件名称：eventType:fieldPath
    const fullEventType = `${eventType}:${this.fieldPath}`
    
    // 发出事件
    this.formManager.emit(fullEventType, fullPayload)
  }

  /**
   * 捕获快照（默认实现）
   */
  captureSnapshot(): WidgetSnapshot {

    return {
      widget_type: this.field.widget.type,
      field_path: this.fieldPath,
      field_code: this.fieldCode,
      field_value: {
        raw: this.value.value.raw,
        display: this.value.value.display,
        meta: this.value.value.meta
      },
      component_data: this.captureComponentData()
    }
  }

  /**
   * 恢复快照（默认实现）
   */
  restoreSnapshot(snapshot: WidgetSnapshot): void {

    // 恢复 FieldValue
    this.setValue({
      raw: snapshot.field_value.raw,
      display: snapshot.field_value.display,
      meta: snapshot.field_value.meta
    })

    // 恢复组件特定数据
    if (snapshot.component_data) {
      this.restoreComponentData(snapshot.component_data)
    }
  }

  /**
   * 捕获组件特定数据（子类覆盖）
   */
  protected captureComponentData(): any {
    return null
  }

  /**
   * 恢复组件特定数据（子类覆盖）
   */
  protected restoreComponentData(data: any): void {
    // 默认无操作
  }

  /**
   * 渲染方法（子类必须实现）
   */
  abstract render(): any
}

