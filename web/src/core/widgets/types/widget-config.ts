/**
 * Widget 配置类型定义
 * 提供公共的配置接口，减少重复代码
 */

/**
 * 基础 Widget 配置 - 所有 Widget 都支持的基础配置
 */
export interface BaseWidgetConfig {
  placeholder?: string
  disabled?: boolean
  default?: any
}

/**
 * 输入框相关配置 - 支持前缀、后缀、清除等
 */
export interface InputLikeConfig extends BaseWidgetConfig {
  prepend?: string  // 前缀
  append?: string   // 后缀
  clearable?: boolean
}

/**
 * 数字输入配置 - 继承输入框配置，添加数字相关属性
 */
export interface NumberLikeConfig extends InputLikeConfig {
  min?: number
  max?: number
  step?: number
}

/**
 * 文本域配置
 */
export interface TextAreaConfig extends BaseWidgetConfig {
  rows?: number
  autosize?: boolean | { minRows?: number; maxRows?: number }
  maxlength?: number
  showWordLimit?: boolean
}

/**
 * 输入框专用配置 - 支持密码框等功能
 */
export interface InputConfig extends InputLikeConfig {
  maxlength?: number
  minlength?: number
  showWordLimit?: boolean
  password?: boolean  // 是否为密码框
  showPassword?: boolean  // 是否显示密码切换按钮
}

