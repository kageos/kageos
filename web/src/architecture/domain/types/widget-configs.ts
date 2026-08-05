/**
 * Widget 配置类型定义
 * 🔥 100% 对齐后端 kageos-sdk/agent-app/widget/ 中的结构体定义
 * 
 * 设计原则：
 * - 每个组件都有独立的 Config 接口
 * - 所有字段都是可选的（omitempty）
 * - 字段命名与后端 JSON 标签完全一致（snake_case）
 * - 添加详细注释说明每个字段的用途和示例
 */

export interface SelectOptionConfig {
  label: string
  value: any
  disabled?: boolean
  displayInfo?: any
  display_info?: any
  icon?: string
  rich_text?: string
  files?: string
}

/**
 * Input Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/input.go
 */
export interface InputWidgetConfig {
  /** 占位符文本 */
  placeholder?: string

  /** 是否禁用 */
  disabled?: boolean
  
  /** 是否为密码框（true 时输入内容会被隐藏，且不会同步到 URL） */
  password?: boolean
  
  /** 输入框前置内容（如：￥、http://） */
  prepend?: string
  
  /** 输入框后置内容（如：.com、元） */
  append?: string
  
  /** 前端渲染默认值 */
  render_default?: string

}

/**
 * Select Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/select.go
 */
export interface SelectWidgetConfig {
  /** 选项列表（静态选项） */
  options?: Array<string | SelectOptionConfig>
  
  /** 
   * 选项的颜色配置
   * 仅支持 6 位十六进制 RRGGBB，不带 #
   * 每个颜色可以重复使用
   * 未识别颜色会降级为中性灰
   * 示例：["67C23A", "E6A23C", "F56C6C"]
   */
  options_colors?: string[]
  
  /** 占位符文本 */
  placeholder?: string

  /** 是否禁用 */
  disabled?: boolean
  
  /** 前端渲染默认选中的值 */
  render_default?: string | number | boolean | null

  
  /** 是否支持创建新选项（用户可以在下拉框中输入新值） */
  creatable?: boolean
}

/**
 * MultiSelect Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/multiselect.go
 */
export interface MultiSelectWidgetConfig {
  /** 选项列表（静态选项） */
  options?: Array<string | SelectOptionConfig>
  
  /** 
   * 选项的颜色配置
   * 仅支持 6 位十六进制 RRGGBB，不带 #
   * 每个颜色可以重复使用
   * 未识别颜色会降级为中性灰
   */
  options_colors?: string[]
  
  /** 占位符文本 */
  placeholder?: string

  /** 是否禁用 */
  disabled?: boolean
  
  /** 前端渲染默认选中的值（多个，逗号分隔） */
  render_default?: Array<string | number | boolean> | string

  
  /** 最大选择数量（0 表示不限制） */
  max_count?: number
  
  /** 是否支持创建新选项 */
  creatable?: boolean
}

/**
 * List Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/list.go
 */
export interface ListWidgetConfig {
  /** 元素类型：number 表示数字列表，text 表示文本列表 */
  item_type?: 'number' | 'text'

  /** 输入分隔符，默认逗号；组件也会兼容空白、换行和中文逗号 */
  separator?: string

  /** 占位符文本 */
  placeholder?: string

  /** 是否禁用 */
  disabled?: boolean

  /** 前端渲染默认值，如 "1,2,3" 或 "a,b,c" */
  render_default?: string | Array<string | number>

  /** 是否去重 */
  unique?: boolean

  /** 最大数量，0 表示不限制 */
  max_count?: number
}

/**
 * Integer Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/integer.go
 */
export interface IntegerWidgetConfig {
  /** 占位符文本 */
  placeholder?: string

  /** 是否禁用 */
  disabled?: boolean

  /** 最小值 */
  min?: number

  /** 最大值 */
  max?: number
  
  /** 步长（点击增减按钮的步进值，字符串或数字） */
  step?: string | number
  
  /** 前端渲染默认值（整数） */
  render_default?: number

  
  /** 单位（如：件、个、元、kg 等） */
  unit?: string
}

/**
 * Float Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/float.go
 */
export interface FloatWidgetConfig {
  /** 占位符文本 */
  placeholder?: string

  /** 是否禁用 */
  disabled?: boolean

  /** 最小值 */
  min?: number

  /** 最大值 */
  max?: number
  
  /** 小数位数（显示和输入精度，字符串或数字） */
  precision?: string | number
  
  /** 步长（点击增减按钮的步进值，字符串或数字） */
  step?: string | number
  
  /** 前端渲染默认值（浮点数） */
  render_default?: number

  
  /** 单位（如：元、kg、% 等） */
  unit?: string
}

/**
 * TextArea Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/text_area.go
 */
export interface TextAreaWidgetConfig {
  /** 占位符文本 */
  placeholder?: string

  /** 是否禁用 */
  disabled?: boolean
  
  /** 前端渲染默认值 */
  render_default?: string

  /** 文本域行数 */
  rows?: number | string

}

/**
 * Switch Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/switch.go
 * 
 * 注意：当前 Switch 组件没有配置项（大道至简，MVP 产品）
 */
export interface SwitchWidgetConfig {
  /** 前端渲染默认值；false 为默认空值时会被省略， */
  render_default?: boolean

}

/**
 * DateTime Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/datetime.go
 *
 * raw value 为 "YYYY-MM-DD HH:mm:ss" 字符串，后端推荐用 types.Time 写入数据库 datetime/time 列。
 */
export interface DateTimeWidgetConfig {
  /**
   * 日期格式
   * 示例：YYYY-MM-DD HH:mm:ss、YYYY-MM-DD
   */
  format?: string

  /** 是否禁用（只读模式） */
  disabled?: boolean

  /**
   * 前端渲染默认值
   * 推荐 SQL 风格白名单表达式：CURRENT_TIMESTAMP、CURRENT_DATE、DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 1 HOUR)
   */
  render_default?: string

}

/**
 * Files Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/files.go
 */
export interface FilesWidgetConfig {
  /** 
   * 文件类型限制
   * 支持多种格式（逗号分隔）：
   * - 扩展名：.pdf,.doc,.docx,.jpg,.png
   * - MIME类型：application/pdf,image/jpeg
   * - MIME通配符：image/*,video/*,audio/*
   * - 混合使用：.pdf,image/*,video/*
   * 示例：accept:.pdf,.doc,.docx,image/*,video/*
   * 为空则不限制类型
   */
  accept?: string
  
  /** 
   * 单个文件最大大小
   * 支持单位：B, KB, MB, GB
   * 示例：10MB、1024KB、1GB
   * 为空则使用系统默认值
   */
  max_size?: string
  
  /** 
   * 最大上传文件数量
   * 默认为 5
   * 示例：max_count:10
   */
  max_count?: number

  /** 是否禁用 */
  disabled?: boolean

  /** 是否在浏览器上传时生成图片缩略图或视频封面 */
  thumbnail?: boolean | 'true' | 'false' | '1' | '0' | 1 | 0

  /** 是否在表格列表单元格内联展示缩略图或封面 */
  list_preview?: boolean | 'true' | 'false' | '1' | '0' | 1 | 0
}

/**
 * Slider Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/slider.go
 */
export interface SliderWidgetConfig {
  /** 最小值（必需，默认 0） */
  min?: number
  
  /** 最大值（必需，默认 100） */
  max?: number
  
  /** 步长（可选，默认 1） */
  step?: number
  
  /** 前端渲染默认值（可选） */
  render_default?: number


  /** 是否禁用 */
  disabled?: boolean
  
  /** 单位（可选，如：%、元、kg 等） */
  unit?: string
}

/**
 * Rate Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/rate.go
 */
export interface RateWidgetConfig {
  /** 最大星级（默认 5） */
  max?: number
  
  /** 是否允许半星（默认 false） */
  allow_half?: boolean | 'true' | 'false'
  
  /** 前端渲染默认评分（可选） */
  render_default?: number

  
  /** 
   * 自定义文字数组
   * 示例：["很差", "差", "一般", "好", "很好"]
   * 如果配置了 texts，会自动显示文字；如果没有配置，则不显示文字
   */
  texts?: string[]
}

/**
 * Color Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/color.go
 */
export interface ColorWidgetConfig {
  /** 
   * 颜色格式
   * 可选值：hex（默认）、rgb、rgba
   */
  format?: 'hex' | 'rgb' | 'rgba'
  
  /** 前端渲染默认颜色（可选，如：#409EFF） */
  render_default?: string

  
  /** 
   * 是否显示透明度选择器
   * 默认 false，仅在 format 为 rgba 时有效
   * 如果启用透明度，会自动设置为 rgba 格式
   */
  show_alpha?: boolean | 'true' | 'false'
}

/**
 * RichText Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/richtext.go
 */
export interface RichTextWidgetConfig {
  /** 编辑器高度（单位：px，默认 300） */
  height?: number
}

/**
 * Link Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/link.go
 */
export interface LinkWidgetConfig {
  /** 链接文本（可选，如果不设置则使用字段名称） */
  text?: string
  
  /** 链接打开方式（_self, _blank，默认 _self） */
  target?: '_self' | '_blank'
  
  /** 链接类型（primary, success, warning, danger, info，默认 primary） */
  type?: 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'link'
  
  /** 链接图标（可选） */
  icon?: string
}

/**
 * Progress Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/progress.go
 */
export interface ProgressWidgetConfig {
  /** 最小值（默认 0） */
  min?: number
  
  /** 最大值（默认 100） */
  max?: number
  
  /** 单位（如：%、人、次等，默认 %） */
  unit?: string
}

/**
 * Checkbox Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/checkbox.go
 */
export interface CheckboxWidgetConfig {
  /** 选项列表（逗号分隔） */
  options?: string[]
  
  /** 前端渲染默认选中项（逗号分隔） */
  render_default?: string[] | string

}

/**
 * Radio Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/radio.go
 */
export interface RadioWidgetConfig {
  /** 选项列表（逗号分隔） */
  options?: string[]
  
  /** 前端渲染默认选中项 */
  render_default?: string

}

/**
 * User Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/user.go
 * 
 * 功能：
 * - 支持用户搜索和选择
 * - 支持动态默认值函数：Me()（当前登录用户）、MyLeader()（当前用户的上级领导）
 */
export interface UserWidgetConfig {
  /** 
   * 前端渲染默认值
   * 支持函数调用：
   * - Me()：当前登录用户，适用于预约人、创建人、负责人等字段
   * - MyLeader()：当前用户的上级领导，适用于审批人、抄送人、上级领导等字段
   */
  render_default?: string

  
  /** 是否禁用（只读模式，Form 中展示但不可编辑） */
  disabled?: boolean
}

/**
 * Users Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/users.go
 * 
 * 功能：
 * - 支持多个用户搜索和选择
 * - 支持动态默认值函数：Me()（当前登录用户）、MyLeader()（当前用户的上级领导）
 * - 值使用逗号分隔的字符串格式存储（如 "user1,user2"）
 */
export interface UsersWidgetConfig {
  /** 
   * 前端渲染默认值
   * 支持函数调用：
   * - Me()：当前登录用户
   * - MyLeader()：当前用户的上级领导
   * 多个值用逗号分隔，如 "Me(),MyLeader(),user2"
   */
  render_default?: string

  
  /** 最大选择数量，0表示不限制 */
  max_count?: number
  
  /** 详情模式最多显示的头像数量（默认 5 个） */
  max_display_count?: number
}

/**
 * Department Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/department.go
 * 
 * 功能：
 * - 支持组织架构搜索和选择
 * - 支持动态默认值函数：MyDepartment()（当前用户所在部门）
 */
export interface DepartmentWidgetConfig {
  /** 
   * 前端渲染默认值
   * 支持函数调用 MyDepartment()（当前用户所在部门）
   * 适用于：所属部门、创建部门等字段
   */
  render_default?: string

}

/**
 * Departments Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/departments.go
 * 
 * 功能：
 * - 支持多个组织架构搜索和选择
 * - 支持动态默认值函数：MyDepartment()（当前用户所在部门）
 * - 值使用逗号分隔的字符串格式存储（如 "/dept1,/dept2"）
 */
export interface DepartmentsWidgetConfig {
  /** 
   * 前端渲染默认值
   * 支持函数调用 MyDepartment()（当前用户所在部门），多个值用逗号分隔
   */
  render_default?: string

  
  /** 最大选择数量，0表示不限制 */
  max_count?: number
}

/**
 * Text Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/text.go
 * 
 * 注意：Text 组件一般用于输出参数中，支持格式化显示
 */
export interface TextWidgetConfig {
  /** 
   * 格式化类型
   * 支持：json、yaml、xml、markdown、html、csv 等
   */
  format?: string
}

/**
 * ID Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/id.go
 * 
 * 注意：ID 组件用于显示 ID 字段，通常不需要配置
 */
export interface IDWidgetConfig {
  // 当前无配置项（或根据实际需求添加）
}

/**
 * Table Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/table.go
 * 
 * 注意：Table 是容器组件，用于嵌套字段，通常不需要配置
 */
export interface TableWidgetConfig {
  // 当前无配置项（或根据实际需求添加）
}

/**
 * Form Widget 配置
 * 对应后端：kageos-sdk/agent-app/widget/form.go
 * 
 * 注意：Form 是容器组件，用于嵌套字段，通常不需要配置
 */
export interface FormWidgetConfig {
  // 当前无配置项（或根据实际需求添加）
}

/**
 * Widget 配置类型映射
 * 根据 widget type 获取对应的 config 类型
 */
export type WidgetConfigMap = {
  input: InputWidgetConfig
  select: SelectWidgetConfig
  multiselect: MultiSelectWidgetConfig
  list: ListWidgetConfig
  integer: IntegerWidgetConfig
  float: FloatWidgetConfig
  text_area: TextAreaWidgetConfig
  switch: SwitchWidgetConfig
  datetime: DateTimeWidgetConfig
  files: FilesWidgetConfig
  slider: SliderWidgetConfig
  rate: RateWidgetConfig
  color: ColorWidgetConfig
  richtext: RichTextWidgetConfig
  link: LinkWidgetConfig
  progress: ProgressWidgetConfig
  checkbox: CheckboxWidgetConfig
  radio: RadioWidgetConfig
  user: UserWidgetConfig
  users: UsersWidgetConfig
  department: DepartmentWidgetConfig
  departments: DepartmentsWidgetConfig
  text: TextWidgetConfig
  ID: IDWidgetConfig
  table: TableWidgetConfig
  form: FormWidgetConfig
}

/**
 * 根据 widget type 获取对应的 config 类型
 * 
 * @example
 * type InputConfig = GetWidgetConfig<'input'>  // InputWidgetConfig
 * type SelectConfig = GetWidgetConfig<'select'>  // SelectWidgetConfig
 */
export type GetWidgetConfig<T extends string> = T extends keyof WidgetConfigMap
  ? WidgetConfigMap[T]
  : Record<string, any>  // 未知类型使用通用类型

/**
 * 所有 Widget Config 的联合类型
 */
export type AnyWidgetConfig = 
  | InputWidgetConfig
  | SelectWidgetConfig
  | MultiSelectWidgetConfig
  | ListWidgetConfig
  | IntegerWidgetConfig
  | FloatWidgetConfig
  | TextAreaWidgetConfig
  | SwitchWidgetConfig
  | DateTimeWidgetConfig
  | FilesWidgetConfig
  | SliderWidgetConfig
  | RateWidgetConfig
  | ColorWidgetConfig
  | RichTextWidgetConfig
  | LinkWidgetConfig
  | ProgressWidgetConfig
  | CheckboxWidgetConfig
  | RadioWidgetConfig
  | UserWidgetConfig
  | UsersWidgetConfig
  | DepartmentWidgetConfig
  | DepartmentsWidgetConfig
  | TextWidgetConfig
  | IDWidgetConfig
  | TableWidgetConfig
  | FormWidgetConfig
