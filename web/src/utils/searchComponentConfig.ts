/**
 * 搜索组件配置生成工具
 * 🔥 统一生成不同组件类型的搜索配置，遵循单一职责原则和依赖倒置原则
 */

import { WidgetType } from '@/core/constants/widget'
import { SearchType, SearchComponent, SearchConfig, hasSearchType, hasAllSearchTypes } from '@/core/constants/search'
import { generatePlaceholder } from '@/utils/stringUtils'
import type { FieldConfig } from '@/core/types/field'

/**
 * 组件配置接口
 */
export interface ComponentConfig {
  component: string
  props?: Record<string, any>
  onRemoteMethod?: (query: string) => Promise<Array<{ label: string; value: any }>>
}

/**
 * 创建搜索组件配置
 * @param field 字段配置
 * @param searchType 搜索类型
 * @param functionMethod 函数 HTTP 方法（用于 OnSelectFuzzy 回调）
 * @param functionRouter 函数路由（用于 OnSelectFuzzy 回调）
 * @returns 组件配置
 */
export function createSearchComponentConfig(
  field: FieldConfig,
  searchType: string | undefined,
  functionMethod?: string,
  functionRouter?: string
): ComponentConfig {
  const widgetType = field.widget?.type || WidgetType.INPUT
  const widgetConfig = field.widget?.config || {}

  // 用户组件
  if (widgetType === WidgetType.USER) {
    return createUserComponentConfig(field, searchType)
  }

  // 时间戳组件
  if (widgetType === WidgetType.TIMESTAMP) {
    return createTimestampComponentConfig(field, searchType)
  }

  // 选择组件
  if (widgetType === WidgetType.SELECT) {
    return createSelectComponentConfig(field, searchType, widgetConfig, functionMethod, functionRouter)
  }

  // 多选组件
  if (widgetType === WidgetType.MULTI_SELECT) {
    return createMultiselectComponentConfig(field, widgetConfig)
  }

  // 开关组件
  if (widgetType === WidgetType.SWITCH) {
    return createSwitchComponentConfig(field, widgetConfig)
  }

  // Slider 组件（范围搜索）
  if (widgetType === WidgetType.SLIDER) {
    return createSliderComponentConfig(field, searchType, widgetConfig)
  }

  // Rate 组件（范围搜索，类似 Slider）
  if (widgetType === WidgetType.RATE) {
    return createRateComponentConfig(field, searchType, widgetConfig)
  }

  // Color 组件（文本搜索）
  if (widgetType === WidgetType.COLOR) {
    return createColorComponentConfig(field, searchType)
  }

  // RichText 组件（文本搜索，搜索 HTML 内容）
  if (widgetType === WidgetType.RICH_TEXT) {
    return createDefaultInputConfig(field)
  }

  // 文本范围搜索
  if (hasAllSearchTypes(searchType, [SearchType.GTE, SearchType.LTE])) {
    return createRangeInputConfig(field)
  }

  // 多选搜索（in，用于文本类型）
  if (hasSearchType(searchType, SearchType.IN) && widgetType !== WidgetType.MULTI_SELECT) {
    return createMultiSelectConfig(field)
  }

  // 默认：普通文本输入框
  return createDefaultInputConfig(field)
}

/**
 * 创建用户组件配置
 */
function createUserComponentConfig(field: FieldConfig, searchType: string | undefined): ComponentConfig {
  // 如果 search 标签是 "in" 或 "eq"，使用自定义的用户搜索组件
  if (hasSearchType(searchType, SearchType.IN) || hasSearchType(searchType, SearchType.EQ)) {
    return {
      component: SearchComponent.USER_SEARCH_INPUT,
      props: {
        placeholder: generatePlaceholder(field.name, 'search'),
        multiple: hasSearchType(searchType, SearchType.IN)
      }
    }
  }

  // 如果 search 标签是 "like"，渲染普通文本输入框
  if (hasSearchType(searchType, SearchType.LIKE)) {
    return createDefaultInputConfig(field)
  }

  // 默认：使用精确搜索（eq），渲染用户选择器
  return {
    component: SearchComponent.EL_SELECT,
    props: {
      placeholder: generatePlaceholder(field.name, 'select'),
      clearable: true,
      filterable: true,
      remote: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH }
    },
    onRemoteMethod: createUserRemoteMethod()
  }
}

/**
 * 创建时间戳组件配置
 */
function createTimestampComponentConfig(field: FieldConfig, searchType: string | undefined): ComponentConfig {
  // 范围搜索（gte/lte）
  if (hasAllSearchTypes(searchType, [SearchType.GTE, SearchType.LTE])) {
    return {
      component: SearchComponent.EL_DATE_PICKER,
      props: {
        type: 'datetimerange',
        rangeSeparator: '至',
        startPlaceholder: generatePlaceholder(field.name, 'start'),
        endPlaceholder: generatePlaceholder(field.name, 'end'),
        format: 'YYYY-MM-DD HH:mm:ss',
        valueFormat: 'x', // 毫秒级时间戳格式
        clearable: true,
        style: { width: SearchConfig.DEFAULT_RANGE_WIDTH },
        shortcuts: createDateShortcuts()
      }
    }
  }

  // 单个日期搜索
  return {
    component: SearchComponent.EL_DATE_PICKER,
    props: {
      type: 'datetime',
      placeholder: generatePlaceholder(field.name, 'select'),
      format: 'YYYY-MM-DD HH:mm:ss',
      valueFormat: 'x', // 毫秒级时间戳格式
      clearable: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH }
    }
  }
}

/**
 * 创建选择组件配置
 */
function createSelectComponentConfig(
  field: FieldConfig,
  searchType: string | undefined,
  widgetConfig: Record<string, any>,
  functionMethod?: string,
  functionRouter?: string
): ComponentConfig {
  const options = getWidgetOptions(widgetConfig)
  
  // 🔥 检查是否有 OnSelectFuzzy 回调
  const hasCallback = field.callbacks?.includes('OnSelectFuzzy') || false
  
  // 🔥 创建 onRemoteMethod（用于 by_keyword 搜索）
  const onRemoteMethod = hasCallback && functionMethod && functionRouter
    ? createSelectFuzzyRemoteMethod(field, functionMethod, functionRouter)
    : undefined
  
  // 🔥 创建 onInitOptions（用于 by_value 搜索，初始化已选中的值）
  const onInitOptions = hasCallback && functionMethod && functionRouter
    ? createSelectFuzzyInitOptions(field, functionMethod, functionRouter)
    : undefined

  // 多选搜索（in）
  if (hasSearchType(searchType, SearchType.IN)) {
    return {
      component: SearchComponent.EL_SELECT,
      props: {
        placeholder: generatePlaceholder(field.name, 'select'),
        clearable: true,
        filterable: true,
        multiple: true,
        remote: hasCallback, // 🔥 如果有回调，启用 remote 模式
        style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
        collapseTags: true,
        maxCollapseTags: SearchConfig.MAX_COLLAPSE_TAGS,
        options
      },
      onRemoteMethod,
      onInitOptions
    }
  }

  // 单选搜索（eq）
  return {
    component: SearchComponent.EL_SELECT,
    props: {
      placeholder: generatePlaceholder(field.name, 'select'),
      clearable: true,
      filterable: true,
      remote: hasCallback, // 🔥 如果有回调，启用 remote 模式
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
      options
    },
    onRemoteMethod,
    onInitOptions
  }
}

/**
 * 创建多选组件配置
 */
function createMultiselectComponentConfig(field: FieldConfig, widgetConfig: Record<string, any>): ComponentConfig {
  const options = getWidgetOptions(widgetConfig)

  return {
    component: SearchComponent.EL_SELECT,
    props: {
      placeholder: generatePlaceholder(field.name, 'select'),
      clearable: true,
      filterable: true,
      multiple: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
      collapseTags: true,
      maxCollapseTags: SearchConfig.MAX_COLLAPSE_TAGS,
      options
    }
  }
}

/**
 * 创建开关组件配置
 */
function createSwitchComponentConfig(field: FieldConfig, widgetConfig: Record<string, any>): ComponentConfig {
  const activeText = widgetConfig.activeText || '是'
  const inactiveText = widgetConfig.inactiveText || '否'

  return {
    component: SearchComponent.EL_SELECT,
    props: {
      placeholder: generatePlaceholder(field.name, 'select'),
      clearable: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
      options: [
        { label: activeText, value: true },
        { label: inactiveText, value: false }
      ]
    }
  }
}

/**
 * 创建 Slider 组件配置（范围搜索）
 */
function createSliderComponentConfig(
  field: FieldConfig,
  searchType: string | undefined,
  widgetConfig: Record<string, any>
): ComponentConfig {
  // Slider 组件默认支持范围搜索（gte/lte）
  const min = Number(widgetConfig.min) || 0
  const max = Number(widgetConfig.max) || 100
  const step = Number(widgetConfig.step) || 1
  
  // 计算步长的小数位数（用于 input-number 的 precision）
  const stepStr = String(step)
  const precision = stepStr.includes('.') ? stepStr.split('.')[1].length : 0

  return {
    component: SearchComponent.NUMBER_RANGE_INPUT,
    props: {
      minPlaceholder: generatePlaceholder(field.name, 'min'),
      maxPlaceholder: generatePlaceholder(field.name, 'max'),
      min: min,
      max: max,
      step: step,
      precision: precision,
      unit: widgetConfig.unit || ''
    }
  }
}

/**
 * 创建 Rate 组件配置（范围搜索，类似 Slider）
 */
function createRateComponentConfig(
  field: FieldConfig,
  searchType: string | undefined,
  widgetConfig: Record<string, any>
): ComponentConfig {
  // Rate 组件默认支持范围搜索（gte/lte）
  const max = Number(widgetConfig.max) || 5
  const allowHalf = widgetConfig.allow_half === true || widgetConfig.allow_half === 'true'
  const step = allowHalf ? 0.5 : 1
  const precision = allowHalf ? 1 : 0

  return {
    component: SearchComponent.NUMBER_RANGE_INPUT,
    props: {
      minPlaceholder: generatePlaceholder(field.name, 'min'),
      maxPlaceholder: generatePlaceholder(field.name, 'max'),
      min: 0,
      max: max,
      step: step,
      precision: precision
    }
  }
}

/**
 * 创建 Color 组件配置（文本搜索）
 */
function createColorComponentConfig(
  field: FieldConfig,
  searchType: string | undefined
): ComponentConfig {
  // Color 组件使用文本输入搜索
  return {
    component: SearchComponent.EL_INPUT,
    props: {
      placeholder: generatePlaceholder(field.name, 'search'),
      clearable: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH }
    }
  }
}

/**
 * 创建范围输入配置
 */
function createRangeInputConfig(field: FieldConfig): ComponentConfig {
  return {
    component: SearchComponent.RANGE_INPUT,
    props: {
      minPlaceholder: generatePlaceholder(field.name, 'min'),
      maxPlaceholder: generatePlaceholder(field.name, 'max')
    }
  }
}

/**
 * 创建多选配置（用于文本类型）
 */
function createMultiSelectConfig(field: FieldConfig): ComponentConfig {
  return {
    component: SearchComponent.EL_SELECT,
    props: {
      placeholder: generatePlaceholder(field.name, 'select'),
      clearable: true,
      filterable: true,
      multiple: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
      collapseTags: true,
      maxCollapseTags: SearchConfig.MAX_COLLAPSE_TAGS
    }
  }
}

/**
 * 创建默认输入框配置
 */
function createDefaultInputConfig(field: FieldConfig): ComponentConfig {
  return {
    component: SearchComponent.EL_INPUT,
    props: {
      placeholder: generatePlaceholder(field.name, 'input'),
      clearable: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH }
    }
  }
}

/**
 * 从 widget 配置获取选项
 * 兼容字符串数组和对象数组
 */
function getWidgetOptions(widgetConfig: Record<string, any>): Array<{ label: string; value: any }> {
  const opts = widgetConfig.options || []
  
  if (opts.length === 0) {
    return []
  }

  // 兼容字符串数组和对象数组
  if (typeof opts[0] === 'string') {
    return opts.map((opt: string) => ({ label: opt, value: opt }))
  }

  return opts.map((opt: any) => {
    if (typeof opt === 'object' && opt !== null) {
      return { label: opt.label || opt.value || String(opt), value: opt.value || opt }
    }
    return { label: String(opt), value: opt }
  })
}

/**
 * 创建用户远程搜索方法
 */
function createUserRemoteMethod(): (query: string) => Promise<Array<{ label: string; value: any }>> {
  return async (query: string) => {
    if (!query || query.trim() === '') {
      return []
    }

    try {
      const { searchUsersFuzzy } = await import('@/api/user')
      const response = await searchUsersFuzzy(query.trim(), SearchConfig.DEFAULT_PAGE_SIZE)
      const users = response.users || []

      return users.map((user: any) => ({
        label: user.nickname ? `${user.username}(${user.nickname})` : user.username,
        value: user.username
      }))
    } catch (error) {
      console.error('[SearchInput] 搜索用户失败', error)
      return []
    }
  }
}

/**
 * 创建 OnSelectFuzzy 回调的远程搜索方法（by_keyword）
 */
function createSelectFuzzyRemoteMethod(
  field: FieldConfig,
  functionMethod: string,
  functionRouter: string
): (query: string) => Promise<Array<{ label: string; value: any }>> {
  return async (query: string) => {
    if (!query || query.trim() === '') {
      return []
    }

    try {
      const { selectFuzzy } = await import('@/api/function')
      const { SelectFuzzyQueryType } = await import('@/core/constants/select')
      
      const valueType = field.data?.type || 'string'
      const response = await selectFuzzy(functionMethod, functionRouter, {
        code: field.code,
        type: SelectFuzzyQueryType.BY_KEYWORD,
        value: query.trim(),
        request: {}, // 搜索模式下，request 为空
        value_type: valueType
      })

      if (response.error_msg) {
        console.error('[SearchInput] OnSelectFuzzy 回调错误:', response.error_msg)
        return []
      }

      // 转换响应格式
      const items = response.items || []
      return items.map((item: any) => ({
        label: item.label || String(item.value),
        value: item.value
      }))
    } catch (error) {
      console.error('[SearchInput] OnSelectFuzzy 回调失败', error)
      return []
    }
  }
}

/**
 * 创建 OnSelectFuzzy 回调的初始化选项方法（by_value）
 */
function createSelectFuzzyInitOptions(
  field: FieldConfig,
  functionMethod: string,
  functionRouter: string
): (value: any) => Promise<Array<{ label: string; value: any }>> {
  return async (value: any) => {
    if (!value || (Array.isArray(value) && value.length === 0)) {
      return []
    }

    try {
      const { selectFuzzy } = await import('@/api/function')
      const { SelectFuzzyQueryType } = await import('@/core/constants/select')
      
      const valueType = field.data?.type || 'string'
      
      // 🔥 判断是单个值还是多个值
      const isArray = Array.isArray(value)
      const values = isArray ? value : [value]
      
      // 🔥 类型转换：根据 value_type 将字符串转换为正确的类型
      let convertedValues: any[] = []
      for (const val of values) {
        let convertedValue: any = val
        // 🔥 处理字符串类型的值（可能来自 URL 参数）
        if (typeof val === 'string' && valueType !== 'string') {
          if (valueType === 'int' || valueType === 'integer') {
            convertedValue = parseInt(val, 10)
            if (isNaN(convertedValue)) {
              continue
            }
          } else if (valueType === 'float' || valueType === 'number') {
            convertedValue = parseFloat(val)
            if (isNaN(convertedValue)) {
              continue
            }
          }
        }
        convertedValues.push(convertedValue)
      }
      
      if (convertedValues.length === 0) {
        return []
      }
      
      // 🔥 如果只有一个值，使用 by_value；如果有多个值，使用 by_values
      const queryType = convertedValues.length === 1 
        ? SelectFuzzyQueryType.BY_VALUE 
        : SelectFuzzyQueryType.BY_VALUES
      const queryValue = convertedValues.length === 1 
        ? convertedValues[0] 
        : convertedValues
      
      const response = await selectFuzzy(functionMethod, functionRouter, {
        code: field.code,
        type: queryType,
        value: queryValue,
        request: {}, // 搜索模式下，request 为空
        value_type: valueType
      })

      if (response.error_msg) {
        console.error('[SearchInput] OnSelectFuzzy 回调错误:', response.error_msg)
        return []
      }

      // 转换响应格式
      const items = response.items || []
      return items.map((item: any) => ({
        label: item.label || String(item.value),
        value: item.value
      }))
    } catch (error) {
      console.error('[SearchInput] OnSelectFuzzy 回调失败', error)
      return []
    }
  }
}

/**
 * 创建日期快捷选项
 */
function createDateShortcuts(): Array<{ text: string; value: () => number[] }> {
  return [
    {
      text: '今天',
      value: () => {
        const start = new Date()
        start.setHours(0, 0, 0, 0)
        const end = new Date()
        end.setHours(23, 59, 59, 999)
        return [start.getTime(), end.getTime()]
      }
    },
    {
      text: '昨天',
      value: () => {
        const start = new Date()
        start.setDate(start.getDate() - 1)
        start.setHours(0, 0, 0, 0)
        const end = new Date()
        end.setDate(end.getDate() - 1)
        end.setHours(23, 59, 59, 999)
        return [start.getTime(), end.getTime()]
      }
    },
    {
      text: '最近7天',
      value: () => {
        const end = new Date()
        end.setHours(23, 59, 59, 999)
        const start = new Date()
        start.setDate(start.getDate() - 6)
        start.setHours(0, 0, 0, 0)
        return [start.getTime(), end.getTime()]
      }
    },
    {
      text: '最近30天',
      value: () => {
        const end = new Date()
        end.setHours(23, 59, 59, 999)
        const start = new Date()
        start.setDate(start.getDate() - 29)
        start.setHours(0, 0, 0, 0)
        return [start.getTime(), end.getTime()]
      }
    }
  ]
}

