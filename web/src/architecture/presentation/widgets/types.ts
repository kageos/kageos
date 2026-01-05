/**
 * Widget 组件类型定义
 * 🔥 完全新增，不依赖旧代码
 * 
 * 🔥 统一类型系统：使用 WidgetTypes 命名空间
 */

import type { WidgetTypes } from '@/architecture/domain/types'
import type { ReactiveFormDataManager, FormRendererContext } from '@/architecture/presentation/widgets/types/widget'

/**
 * Widget 渲染模式
 * 🔥 从 WidgetTypes 命名空间导入
 */
export type WidgetMode = WidgetTypes.WidgetMode

// 🔥 向后兼容：导出类型别名
export type FieldConfig = WidgetTypes.FieldConfig
export type FieldValue = WidgetTypes.FieldValue

/**
 * Widget 组件 Props 接口
 * 🔥 所有组件使用统一的 Props 接口
 */
export interface WidgetComponentProps {
  // ========== 必需属性 ==========
  /** 字段配置 */
  field: WidgetTypes.FieldConfig
  
  /** 字段值 */
  value: WidgetTypes.FieldValue
  
  /** 场景模式 */
  mode: WidgetTypes.WidgetMode
  
  /** 字段路径（如：'name', 'products[0].name'） */
  fieldPath: string
  
  // ========== 可选属性 ==========
  /** 表单数据管理器（编辑模式需要） */
  formManager?: ReactiveFormDataManager | null
  
  /** FormRenderer 上下文（编辑模式需要） */
  formRenderer?: FormRendererContext | null
  
  /** 嵌套深度（用于防止无限递归） */
  depth?: number
  
  // ========== 场景特定属性 ==========
  /** 搜索类型（用于 search 模式） */
  searchType?: string
  
  /** 行数据（用于 table-cell 模式） */
  rowData?: any
  
  /** 行索引（用于 table-cell 模式） */
  rowIndex?: number
  
  /** 函数名称（用于 FilesWidget 打包下载命名） */
  functionName?: string
  
  /** 记录ID（用于 FilesWidget 打包下载命名） */
  recordId?: string | number
  
  /** 父级模式（用于 table-cell 模式判断上下文） */
  parentMode?: WidgetTypes.WidgetMode
}

/**
 * Widget 组件 Emits 接口
 */
export interface WidgetComponentEmits {
  /** 更新字段值 */
  'update:modelValue': [value: WidgetTypes.FieldValue]
  
  /** 统计更新（用于 TableWidget） */
  'statistics:updated'?: [statistics: Record<string, any>]
  
  /** 抽屉状态变化（用于 TableWidget） */
  'drawer:change'?: [show: boolean]
}

