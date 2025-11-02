/**
 * WidgetFactory - Widget 工厂
 * 根据 widget.type 动态创建组件
 */

import type { BaseWidget } from '../widgets/BaseWidget'
import { InputWidget } from '../widgets/InputWidget'
import { NumberWidget } from '../widgets/NumberWidget'
import { FloatWidget } from '../widgets/FloatWidget'
import { TextAreaWidget } from '../widgets/TextAreaWidget'
import { SelectWidget } from '../widgets/SelectWidget'
import { MultiSelectWidget } from '../widgets/MultiSelectWidget'
import { ListWidget } from '../widgets/ListWidget'
import { FormWidget } from '../widgets/FormWidget'
import { SwitchWidget } from '../widgets/SwitchWidget'
import { TimestampWidget } from '../widgets/TimestampWidget'

export class WidgetFactory {
  private widgetMap: Map<string, typeof BaseWidget>

  constructor() {
    this.widgetMap = new Map()
    
    // 🔥 注册默认 Widget
    // 文本输入
    this.registerWidget('input', InputWidget)
    this.registerWidget('text', InputWidget)  // text 别名
    this.registerWidget('ID', InputWidget)  // ID 字段（通常禁用或只读）
    
    // 数字输入
    this.registerWidget('number', NumberWidget)  // 🔥 整数输入（独立组件）
    this.registerWidget('float', FloatWidget)    // 🔥 浮点数输入（独立组件）
    
    // 文本域
    this.registerWidget('textarea', TextAreaWidget)
    this.registerWidget('text_area', TextAreaWidget)  // text_area 别名
    
    // 选择器
    this.registerWidget('select', SelectWidget)        // 🔥 单选
    this.registerWidget('multiselect', MultiSelectWidget)  // 🔥 多选（独立组件）
    
    // 开关
    this.registerWidget('switch', SwitchWidget)  // 🔥 开关组件
    
    // 时间选择器
    this.registerWidget('timestamp', TimestampWidget)  // 🔥 时间戳组件
    
    // 容器组件
    this.registerWidget('list', ListWidget)
    this.registerWidget('table', ListWidget)  // table 是 list 的别名
    this.registerWidget('form', FormWidget)   // form 组件（用于 data.type="struct"）
    
    console.log('[WidgetFactory] 初始化，已注册 Widget:', Array.from(this.widgetMap.keys()))
  }

  /**
   * 注册 Widget
   */
  registerWidget(type: string, WidgetClass: typeof BaseWidget): void {
    this.widgetMap.set(type, WidgetClass)
    console.log(`[WidgetFactory] 注册 Widget: ${type}`)
  }

  /**
   * 获取 Widget 类
   */
  getWidgetClass(type: string): typeof BaseWidget {
    const WidgetClass = this.widgetMap.get(type)
    if (!WidgetClass) {
      console.warn(`[WidgetFactory] 未知的 widget 类型: ${type}，使用 InputWidget`)
      return InputWidget
    }
    return WidgetClass
  }

  /**
   * 检查是否支持该类型
   */
  hasWidget(type: string): boolean {
    return this.widgetMap.has(type)
  }

  /**
   * 获取所有已注册的类型
   */
  getRegisteredTypes(): string[] {
    return Array.from(this.widgetMap.keys())
  }
}

// 导出单例
export const widgetFactory = new WidgetFactory()

