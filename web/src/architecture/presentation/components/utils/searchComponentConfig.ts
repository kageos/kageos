/**
 * 搜索组件配置生成工具
 * 🔥 统一生成不同组件类型的搜索配置，遵循单一职责原则和依赖倒置原则
 */

import { WidgetType } from '@/architecture/domain/constants/widget'
import { SearchType, SearchComponent, SearchConfig, hasSearchType, hasAllSearchTypes } from '@/architecture/domain/constants/search'
import { generatePlaceholder } from '@/architecture/domain/utils/stringUtils'
import type { FieldConfig } from '@/architecture/domain/types/field'

/**
 * 组件配置接口
 */
export interface ComponentConfig {
  component: string
  props?: Record<string, any>
  onRemoteMethod?: (query: string) => Promise<Array<{ label: string; value: any; userInfo?: any; departmentInfo?: any }>>
  onInitOptions?: (value: any) => Promise<Array<{ label: string; value: any; userInfo?: any; departmentInfo?: any }>>
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

  // 多用户组件
  if (widgetType === WidgetType.USERS) {
    return createUsersComponentConfig(field, searchType)
  }

  // 组织架构组件
  if (widgetType === WidgetType.DEPARTMENT) {
    return createDepartmentComponentConfig(field, searchType)
  }

  // 多组织架构组件
  if (widgetType === WidgetType.DEPARTMENTS) {
    return createDepartmentsComponentConfig(field, searchType)
  }

  // 时间组件
  if (widgetType === WidgetType.DATETIME) {
    return createDateTimeComponentConfig(field, searchType)
  }

  // 选择组件
  if (widgetType === WidgetType.SELECT) {
    return createSelectComponentConfig(field, searchType, widgetConfig, functionMethod, functionRouter)
  }

  // 单选组件在搜索栏中也统一走下拉逻辑，保证筛选区交互一致。
  if (widgetType === WidgetType.RADIO) {
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
  // 多选筛选使用统一的远程选择器兜底
  if (hasSearchType(searchType, SearchType.IN) || hasSearchType(searchType, SearchType.EQ)) {
    const multiple = hasSearchType(searchType, SearchType.IN)
    return {
      component: SearchComponent.EL_SELECT,
      props: {
        placeholder: generatePlaceholder(field.name, 'select'),
        clearable: true,
        filterable: true,
        remote: true,
        multiple,
        style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
        collapseTags: multiple,
        maxCollapseTags: SearchConfig.MAX_COLLAPSE_TAGS,
        popperClass: 'user-select-dropdown-popper'
      },
      onRemoteMethod: createUserRemoteMethod(),
      onInitOptions: createUsersInitOptions()
    }
  }

  // 文本筛选渲染普通文本输入框
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
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
      popperClass: 'user-select-dropdown-popper'
    },
    onRemoteMethod: createUserRemoteMethod(),
    onInitOptions: createUsersInitOptions()
  }
}

/**
 * 创建多用户组件配置
 */
function createUsersComponentConfig(field: FieldConfig, searchType: string | undefined): ComponentConfig {
  // 多用户组件默认支持多选搜索（contains/in）
  // 多值筛选使用多选用户搜索
  if (hasSearchType(searchType, SearchType.CONTAINS) || hasSearchType(searchType, SearchType.IN)) {
    return {
      component: SearchComponent.EL_SELECT,
      props: {
        placeholder: generatePlaceholder(field.name, 'select'),
        clearable: true,
        filterable: true,
        remote: true,
        multiple: true,
        style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
        collapseTags: true,
        maxCollapseTags: SearchConfig.MAX_COLLAPSE_TAGS,
        popperClass: 'user-select-dropdown-popper' // 🔥 使用用户选择器的样式
      },
      onRemoteMethod: createUserRemoteMethod(),
      onInitOptions: createUsersInitOptions() // 🔥 支持初始化已选中的用户
    }
  }

  // 文本筛选渲染普通文本输入框
  if (hasSearchType(searchType, SearchType.LIKE)) {
    return createDefaultInputConfig(field)
  }

  // 默认：使用多选搜索（contains），渲染多用户选择器
  return {
    component: SearchComponent.EL_SELECT,
    props: {
      placeholder: generatePlaceholder(field.name, 'select'),
      clearable: true,
      filterable: true,
      remote: true,
      multiple: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
      collapseTags: true,
      maxCollapseTags: SearchConfig.MAX_COLLAPSE_TAGS,
      popperClass: 'user-select-dropdown-popper' // 🔥 使用用户选择器的样式
    },
    onRemoteMethod: createUserRemoteMethod(),
    onInitOptions: createUsersInitOptions() // 🔥 支持初始化已选中的用户
  }
}

/**
 * 创建日期时间组件配置
 */
function createDateTimeComponentConfig(field: FieldConfig, searchType: string | undefined): ComponentConfig {
  const format = field.widget?.config?.format || 'YYYY-MM-DD HH:mm:ss'
  const valueFormat = format

  // 范围搜索（gte/lte）
  if (hasAllSearchTypes(searchType, [SearchType.GTE, SearchType.LTE])) {
    return {
      component: SearchComponent.EL_DATE_PICKER,
      props: {
        type: 'datetimerange',
        rangeSeparator: '至',
        startPlaceholder: generatePlaceholder(field.name, 'start'),
        endPlaceholder: generatePlaceholder(field.name, 'end'),
        format,
        valueFormat,
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
      format,
      valueFormat,
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
  const precision = stepStr.includes('.') ? (stepStr.split('.')[1]?.length ?? 0) : 0

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
function createUserRemoteMethod(): (query: string) => Promise<Array<{ label: string; value: any; userInfo?: any }>> {
  return async (query: string) => {
    if (!query || query.trim() === '') {
      return []
    }

    try {
      const { searchUsersFuzzy } = await import('@/architecture/presentation/context/api/user')
      const response = await searchUsersFuzzy(query.trim(), SearchConfig.DEFAULT_PAGE_SIZE)
      const users = response.users || []

      return users.map((user: any) => ({
        label: user.nickname ? `${user.username}(${user.nickname})` : user.username,
        value: user.username,
        userInfo: user // 🔥 保存用户信息，用于显示头像等
      }))
    } catch (error) {
      console.error('[SearchInput] 搜索用户失败', error)
      return []
    }
  }
}

/**
 * 创建组织架构组件配置
 */
function createDepartmentComponentConfig(field: FieldConfig, searchType: string | undefined): ComponentConfig {
  // 多选筛选使用组织架构搜索
  if (hasSearchType(searchType, SearchType.IN) || hasSearchType(searchType, SearchType.EQ)) {
    return {
      component: SearchComponent.EL_SELECT,
      props: {
        placeholder: generatePlaceholder(field.name, 'select'),
        clearable: true,
        filterable: true,
        remote: true,
        multiple: hasSearchType(searchType, SearchType.IN), // 🔥 支持 IN 查询时启用多选
        style: { width: SearchConfig.DEFAULT_INPUT_WIDTH }
      },
      onRemoteMethod: createDepartmentRemoteMethod()
    }
  }

  // 文本筛选渲染普通文本输入框
  if (hasSearchType(searchType, SearchType.LIKE)) {
    return createDefaultInputConfig(field)
  }

  // 默认：使用精确搜索（eq），渲染组织架构选择器
  return {
    component: SearchComponent.EL_SELECT,
    props: {
      placeholder: generatePlaceholder(field.name, 'select'),
      clearable: true,
      filterable: true,
      remote: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH }
    },
    onRemoteMethod: createDepartmentRemoteMethod()
  }
}

/**
 * 创建多组织架构组件配置
 */
function createDepartmentsComponentConfig(field: FieldConfig, searchType: string | undefined): ComponentConfig {
  // 多组织架构组件默认支持多选搜索（contains/in）
  // 多值筛选使用多选组织架构搜索
  if (hasSearchType(searchType, SearchType.CONTAINS) || hasSearchType(searchType, SearchType.IN)) {
    return {
      component: SearchComponent.EL_SELECT,
      props: {
        placeholder: generatePlaceholder(field.name, 'select'),
        clearable: true,
        filterable: true,
        remote: true,
        multiple: true,
        style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
        collapseTags: true,
        maxCollapseTags: SearchConfig.MAX_COLLAPSE_TAGS
      },
      onRemoteMethod: createDepartmentRemoteMethod(),
      onInitOptions: createDepartmentsInitOptions()
    }
  }

  // 文本筛选渲染普通文本输入框
  if (hasSearchType(searchType, SearchType.LIKE)) {
    return createDefaultInputConfig(field)
  }

  // 默认：使用多选搜索（contains），渲染多组织架构选择器
  return {
    component: SearchComponent.EL_SELECT,
    props: {
      placeholder: generatePlaceholder(field.name, 'select'),
      clearable: true,
      filterable: true,
      remote: true,
      multiple: true,
      style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
      collapseTags: true,
      maxCollapseTags: SearchConfig.MAX_COLLAPSE_TAGS
    },
    onRemoteMethod: createDepartmentRemoteMethod(),
    onInitOptions: createDepartmentsInitOptions()
  }
}

/**
 * 创建组织架构远程搜索方法
 */
function createDepartmentRemoteMethod(): (query: string) => Promise<Array<{ label: string; value: any; departmentInfo?: any }>> {
  // 缓存部门树，避免重复加载
  let cachedDepartmentTree: any[] | null = null
  
  return async (query: string) => {
    try {
      // 加载部门树（如果还没有缓存）
      if (!cachedDepartmentTree) {
        const { getDepartmentTree } = await import('@/architecture/presentation/context/api/department')
        const response = await getDepartmentTree()
        cachedDepartmentTree = response.departments || []
      }
      
      // 扁平化部门列表
      const flattenDepartments = (depts: any[]): any[] => {
        const result: any[] = []
        const traverse = (list: any[]) => {
          for (const dept of list) {
            result.push(dept)
            if (dept.children && dept.children.length > 0) {
              traverse(dept.children)
            }
          }
        }
        traverse(depts)
        return result
      }
      
      const allDepartments = flattenDepartments(cachedDepartmentTree || [])
      
      // 如果没有搜索关键词，返回所有部门（限制数量）
      if (!query || query.trim() === '') {
        return allDepartments.slice(0, SearchConfig.DEFAULT_PAGE_SIZE).map((dept: any) => ({
          label: dept.full_name_path || dept.name,
          value: dept.full_code_path,
          departmentInfo: dept
        }))
      }
      
      // 过滤部门
      const keyword = query.trim().toLowerCase()
      const filtered = allDepartments.filter((dept: any) => {
        return (
          dept.name.toLowerCase().includes(keyword) ||
          dept.full_code_path.toLowerCase().includes(keyword) ||
          (dept.full_name_path && dept.full_name_path.toLowerCase().includes(keyword)) ||
          (dept.code && dept.code.toLowerCase().includes(keyword))
        )
      })
      
      return filtered.slice(0, SearchConfig.DEFAULT_PAGE_SIZE).map((dept: any) => ({
        label: dept.full_name_path || dept.name,
        value: dept.full_code_path,
        departmentInfo: dept
      }))
    } catch (error) {
      console.error('[SearchInput] 搜索组织架构失败', error)
      return []
    }
  }
}

/**
 * 创建多组织架构初始化选项方法（用于初始化已选中的组织架构）
 */
function createDepartmentsInitOptions(): (value: any) => Promise<Array<{ label: string; value: any; departmentInfo?: any }>> {
  return async (value: any) => {
    if (!value) {
      return []
    }
    
    // 如果是数组，转换为逗号分隔的字符串
    const paths = Array.isArray(value) ? value : String(value).split(',').map(p => p.trim()).filter(p => p)
    if (paths.length === 0) {
      return []
    }
    
    try {
      const { getDepartmentTree } = await import('@/architecture/presentation/context/api/department')
      const response = await getDepartmentTree()
      const departmentTree = response.departments || []
      
      // 扁平化部门列表
      const flattenDepartments = (depts: any[]): any[] => {
        const result: any[] = []
        const traverse = (list: any[]) => {
          for (const dept of list) {
            result.push(dept)
            if (dept.children && dept.children.length > 0) {
              traverse(dept.children)
            }
          }
        }
        traverse(depts)
        return result
      }
      
      const allDepartments = flattenDepartments(departmentTree)
      
      // 根据路径查找部门
      const departments = paths
        .map(path => allDepartments.find((dept: any) => dept.full_code_path === path))
        .filter(Boolean)
      
      return departments.map((dept: any) => ({
        label: dept.full_name_path || dept.name,
        value: dept.full_code_path,
        departmentInfo: dept
      }))
    } catch (error) {
      console.error('[SearchInput] 初始化组织架构选项失败', error)
      return []
    }
  }
}

/**
 * 创建多用户初始化选项方法（用于初始化已选中的用户）
 */
function createUsersInitOptions(): (values: string | string[]) => Promise<Array<{ label: string; value: any; userInfo?: any }>> {
  return async (values: string | string[]) => {
    if (!values) {
      return []
    }

    try {
      const { getUsersByUsernames } = await import('@/architecture/presentation/context/api/user')
      // 处理值：如果是字符串，可能是逗号分隔的字符串
      const usernames = Array.isArray(values) 
        ? values 
        : (typeof values === 'string' ? values.split(',').map(u => u.trim()).filter(u => u) : [])
      
      if (usernames.length === 0) {
        return []
      }

      const response = await getUsersByUsernames(usernames)
      const users = response.users || []

      return users.map((user: any) => ({
        label: user.nickname ? `${user.username}(${user.nickname})` : user.username,
        value: user.username,
        userInfo: user // 🔥 保存用户信息，用于显示头像等
      }))
    } catch (error) {
      console.error('[SearchInput] 初始化用户选项失败', error)
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
      const { selectFuzzy } = await import('@/architecture/presentation/context/api/function')
      const { SelectFuzzyQueryType } = await import('@/architecture/domain/constants/select')
      
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
      const { selectFuzzy } = await import('@/architecture/presentation/context/api/function')
      const { SelectFuzzyQueryType } = await import('@/architecture/domain/constants/select')
      
      const valueType = field.data?.type || 'string'
      
      // 🔥 判断是单个值还是多个值
      const isArray = Array.isArray(value)
      const values = isArray ? value : [value]
      
      // 🔥 类型转换：根据 value_type 将字符串转换为正确的类型
      const convertedValues: any[] = []
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
