/**
 * Widget 配置类型定义
 * 🔥 100% 对齐后端 sdk/agent-app/widget/ 中的结构体定义
 * 
 * 设计原则：
 * - 每个组件都有独立的 Config 接口
 * - 所有字段都是可选的（omitempty）
 * - 字段命名与后端 JSON 标签完全一致（snake_case）
 * - 添加详细注释说明每个字段的用途和示例
 */

/**
 * Input Widget 配置
 * 对应后端：sdk/agent-app/widget/input.go
 */
export interface InputWidgetConfig {
  /** 占位符文本 */
  placeholder?: string
  
  /** 是否为密码框（true 时输入内容会被隐藏，且不会同步到 URL） */
  password?: boolean
  
  /** 输入框前置内容（如：￥、http://） */
  prepend?: string
  
  /** 输入框后置内容（如：.com、元） */
  append?: string
  
  /** 默认值 */
  default?: string
}

/**
 * Select Widget 配置
 * 对应后端：sdk/agent-app/widget/select.go
 */
export interface SelectWidgetConfig {
  /** 选项列表（静态选项，逗号分隔） */
  options?: string[]
  
  /** 
   * 选项的颜色配置
   * 支持标准颜色：warning、info、success、danger、primary
   * 支持自定义颜色：如 #FF9800（橙色）、#9C27B0（紫色）
   * 每个颜色可以重复使用
   * 示例：["success", "warning", "#FF9800"]
   */
  options_colors?: string[]
  
  /** 占位符文本 */
  placeholder?: string
  
  /** 默认选中的值 */
  default?: string
  
  /** 是否支持创建新选项（用户可以在下拉框中输入新值） */
  creatable?: boolean
}

/**
 * MultiSelect Widget 配置
 * 对应后端：sdk/agent-app/widget/multiselect.go
 */
export interface MultiSelectWidgetConfig {
  /** 选项列表（静态选项，逗号分隔） */
  options?: string[]
  
  /** 
   * 选项的颜色配置
   * 支持标准颜色：warning、info、success、danger、primary
   * 支持自定义颜色：如 #FF9800（橙色）、#9C27B0（紫色）
   * 每个颜色可以重复使用
   */
  options_colors?: string[]
  
  /** 占位符文本 */
  placeholder?: string
  
  /** 默认选中的值（多个，逗号分隔） */
  default?: string[]
  
  /** 最大选择数量（0 表示不限制） */
  max_count?: number
  
  /** 是否支持创建新选项 */
  creatable?: boolean
}

/**
 * Number Widget 配置
 * 对应后端：sdk/agent-app/widget/number.go
 */
export interface NumberWidgetConfig {
  /** 占位符文本 */
  placeholder?: string
  
  /** 步长（点击增减按钮的步进值，字符串或数字） */
  step?: string | number
  
  /** 默认值（整数） */
  default?: number
  
  /** 单位（如：件、个、元、kg 等） */
  unit?: string
}

/**
 * Float Widget 配置
 * 对应后端：sdk/agent-app/widget/float.go
 */
export interface FloatWidgetConfig {
  /** 占位符文本 */
  placeholder?: string
  
  /** 小数位数（显示和输入精度，字符串或数字） */
  precision?: string | number
  
  /** 步长（点击增减按钮的步进值，字符串或数字） */
  step?: string | number
  
  /** 默认值（浮点数） */
  default?: number
  
  /** 单位（如：元、kg、% 等） */
  unit?: string
}

/**
 * TextArea Widget 配置
 * 对应后端：sdk/agent-app/widget/text_area.go
 */
export interface TextAreaWidgetConfig {
  /** 占位符文本 */
  placeholder?: string
  
  /** 默认值 */
  default?: string
}

/**
 * Switch Widget 配置
 * 对应后端：sdk/agent-app/widget/switch.go
 * 
 * 注意：当前 Switch 组件没有配置项（大道至简，MVP 产品）
 */
export interface SwitchWidgetConfig {
  // 当前无配置项
}

/**
 * Timestamp Widget 配置
 * 对应后端：sdk/agent-app/widget/timestamp.go
 * 
 * 功能：
 * - 支持日期时间选择
 * - 支持动态默认值：$now、$today、$tomorrow、$yesterday 等
 * 
 * 动态默认值说明：
 * - 基础时间：$now（当前时间）、$today（今天 00:00:00）、$tomorrow（明天 00:00:00）、$yesterday（昨天 00:00:00）
 * - 相对时间（小时）：$after_1h、$after_2h、$before_1h 等
 * - 相对时间（天）：$after_1d、$after_7d、$before_1d 等
 * - 相对时间（周/月/年）：$next_week、$last_month、$next_year 等
 */
export interface TimestampWidgetConfig {
  /** 
   * 日期格式
   * 示例：YYYY-MM-DD HH:mm:ss、YYYY-MM-DD
   */
  format?: string
  
  /** 是否禁用（只读模式） */
  disabled?: boolean
  
  /** 
   * 默认值
   * 支持动态变量（以 $ 开头）或具体时间戳
   * 示例：$now、$today、$tomorrow、$yesterday
   */
  default?: string
}

/**
 * Files Widget 配置
 * 对应后端：sdk/agent-app/widget/files.go
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
}

/**
 * Slider Widget 配置
 * 对应后端：sdk/agent-app/widget/slider.go
 */
export interface SliderWidgetConfig {
  /** 最小值（必需，默认 0） */
  min?: number
  
  /** 最大值（必需，默认 100） */
  max?: number
  
  /** 步长（可选，默认 1） */
  step?: number
  
  /** 默认值（可选） */
  default?: number
  
  /** 单位（可选，如：%、元、kg 等） */
  unit?: string
}

/**
 * Rate Widget 配置
 * 对应后端：sdk/agent-app/widget/rate.go
 */
export interface RateWidgetConfig {
  /** 最大星级（默认 5） */
  max?: number
  
  /** 是否允许半星（默认 false） */
  allow_half?: boolean
  
  /** 默认评分（可选） */
  default?: number
  
  /** 
   * 自定义文字数组
   * 示例：["很差", "差", "一般", "好", "很好"]
   * 如果配置了 texts，会自动显示文字；如果没有配置，则不显示文字
   */
  texts?: string[]
}

/**
 * Color Widget 配置
 * 对应后端：sdk/agent-app/widget/color.go
 */
export interface ColorWidgetConfig {
  /** 
   * 颜色格式
   * 可选值：hex（默认）、rgb、rgba
   */
  format?: 'hex' | 'rgb' | 'rgba'
  
  /** 默认颜色（可选，如：#409EFF） */
  default?: string
  
  /** 
   * 是否显示透明度选择器
   * 默认 false，仅在 format 为 rgba 时有效
   * 如果启用透明度，会自动设置为 rgba 格式
   */
  show_alpha?: boolean
}

/**
 * RichText Widget 配置
 * 对应后端：sdk/agent-app/widget/richtext.go
 */
export interface RichTextWidgetConfig {
  /** 编辑器高度（单位：px，默认 300） */
  height?: number
}

/**
 * Link Widget 配置
 * 对应后端：sdk/agent-app/widget/link.go
 */
export interface LinkWidgetConfig {
  /** 链接文本（可选，如果不设置则使用字段名称） */
  text?: string
  
  /** 链接打开方式（_self, _blank，默认 _self） */
  target?: '_self' | '_blank'
  
  /** 链接类型（primary, success, warning, danger, info，默认 primary） */
  type?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  
  /** 链接图标（可选） */
  icon?: string
}

/**
 * Progress Widget 配置
 * 对应后端：sdk/agent-app/widget/progress.go
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
 * 对应后端：sdk/agent-app/widget/checkbox.go
 */
export interface CheckboxWidgetConfig {
  /** 选项列表（逗号分隔） */
  options?: string[]
  
  /** 默认选中项（逗号分隔） */
  default?: string[]
}

/**
 * Radio Widget 配置
 * 对应后端：sdk/agent-app/widget/radio.go
 */
export interface RadioWidgetConfig {
  /** 选项列表（逗号分隔） */
  options?: string[]
  
  /** 默认选中项 */
  default?: string
}

/**
 * User Widget 配置
 * 对应后端：sdk/agent-app/widget/user.go
 * 
 * 功能：
 * - 支持用户搜索和选择
 * - 支持动态默认值函数：me()（当前登录用户）
 */
export interface UserWidgetConfig {
  /** 
   * 默认值
   * 支持函数调用 me()（当前登录用户）
   * 适用于：预约人、创建人、负责人等字段
   */
  default?: string
}

/**
 * Text Widget 配置
 * 对应后端：sdk/agent-app/widget/text.go
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
 * 对应后端：sdk/agent-app/widget/id.go
 * 
 * 注意：ID 组件用于显示 ID 字段，通常不需要配置
 */
export interface IDWidgetConfig {
  // 当前无配置项（或根据实际需求添加）
}

/**
 * Table Widget 配置
 * 对应后端：sdk/agent-app/widget/table.go
 * 
 * 注意：Table 是容器组件，用于嵌套字段，通常不需要配置
 */
export interface TableWidgetConfig {
  // 当前无配置项（或根据实际需求添加）
}

/**
 * Form Widget 配置
 * 对应后端：sdk/agent-app/widget/form.go
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
  number: NumberWidgetConfig
  float: FloatWidgetConfig
  text_area: TextAreaWidgetConfig
  switch: SwitchWidgetConfig
  timestamp: TimestampWidgetConfig
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
  | NumberWidgetConfig
  | FloatWidgetConfig
  | TextAreaWidgetConfig
  | SwitchWidgetConfig
  | TimestampWidgetConfig
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
  | TextWidgetConfig
  | IDWidgetConfig
  | TableWidgetConfig
  | FormWidgetConfig

