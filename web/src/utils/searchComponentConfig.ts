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
 * @returns 组件配置
 */
export function createSearchComponentConfig(
  field: FieldConfig,
  searchType: string | undefined
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
    return createSelectComponentConfig(field, searchType, widgetConfig)
  }

  // 多选组件
  if (widgetType === WidgetType.MULTI_SELECT) {
    return createMultiselectComponentConfig(field, widgetConfig)
  }

  // 开关组件
  if (widgetType === WidgetType.SWITCH) {
    return createSwitchComponentConfig(field, widgetConfig)
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
  widgetConfig: Record<string, any>
): ComponentConfig {
  const options = getWidgetOptions(widgetConfig)

  // 多选搜索（in）
  if (hasSearchType(searchType, SearchType.IN)) {
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

  // 单选搜索（eq）
  return {
    component: SearchComponent.EL_SELECT,
    props: {
      placeholder: generatePlaceholder(field.name, 'select'),
      clearable: true,
      filterable: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
      options
    }
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

