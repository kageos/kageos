/**
 * WidgetFactory - Widget 工厂
 * 根据 widget.type 动态创建组件
 */

import type { BaseWidget } from '../widgets/BaseWidget'
import { Logger } from '../utils/logger'
import { WidgetType } from '../constants/widget'
import { InputWidget } from '../widgets/InputWidget'
import { NumberWidget } from '../widgets/NumberWidget'
import { FloatWidget } from '../widgets/FloatWidget'
import { TextAreaWidget } from '../widgets/TextAreaWidget'
import { SelectWidget } from '../widgets/SelectWidget'
import { MultiSelectWidget } from '../widgets/MultiSelectWidget'
import { TableWidget } from '../widgets/TableWidget'
import { FormWidget } from '../widgets/FormWidget'
import { SwitchWidget } from '../widgets/SwitchWidget'
import { TimestampWidget } from '../widgets/TimestampWidget'
import { FilesWidget } from '../widgets/FilesWidget'
import { ResponseFormWidget } from '../widgets/ResponseFormWidget'
import { ResponseTableWidget } from '../widgets/ResponseTableWidget'

export class WidgetFactory {
  private widgetMap: Map<string, typeof BaseWidget>
  // 🔥 Response Widget 映射（用于响应参数渲染）
  private responseWidgetMap: Map<string, typeof BaseWidget>

  constructor() {
    this.widgetMap = new Map()
    this.responseWidgetMap = new Map()
    
    // 🔥 注册默认 Widget
    // 文本输入
    this.registerWidget(WidgetType.INPUT, InputWidget)
    this.registerWidget(WidgetType.TEXT, InputWidget)  // text 别名
    this.registerWidget(WidgetType.ID, InputWidget)  // ID 字段（通常禁用或只读）
    
    // 数字输入
    this.registerWidget(WidgetType.NUMBER, NumberWidget)  // 🔥 整数输入（独立组件）
    this.registerWidget(WidgetType.FLOAT, FloatWidget)    // 🔥 浮点数输入（独立组件）
    
    // 文本域
    this.registerWidget('textarea', TextAreaWidget)  // 兼容旧命名
    this.registerWidget(WidgetType.TEXT_AREA, TextAreaWidget)
    
    // 选择器
    this.registerWidget(WidgetType.SELECT, SelectWidget)        // 🔥 单选
    this.registerWidget(WidgetType.MULTI_SELECT, MultiSelectWidget)  // 🔥 多选（独立组件）
    
    // 开关
    this.registerWidget(WidgetType.SWITCH, SwitchWidget)  // 🔥 开关组件
    
    // 时间选择器
    this.registerWidget(WidgetType.TIMESTAMP, TimestampWidget)  // 🔥 时间戳组件
    
    // 文件上传
    this.registerWidget(WidgetType.FILES, FilesWidget)  // 🔥 文件上传组件
    
    // 容器组件
    this.registerWidget(WidgetType.TABLE, TableWidget)  // table 表格组件
    this.registerWidget(WidgetType.FORM, FormWidget)   // form 组件（用于 data.type="struct"）
    
    // 🔥 注册 Response Widget（用于响应参数渲染）
    this.registerResponseWidget(WidgetType.FORM, ResponseFormWidget)
    this.registerResponseWidget(WidgetType.TABLE, ResponseTableWidget)
  }

  /**
   * 注册 Widget
   */
  registerWidget(type: string, WidgetClass: typeof BaseWidget): void {
    this.widgetMap.set(type, WidgetClass)
  }

  /**
   * 🔥 注册 Response Widget（用于响应参数渲染）
   * 某些组件在响应参数中需要特殊的只读渲染（如 Form、Table）
   */
  registerResponseWidget(type: string, ResponseWidgetClass: typeof BaseWidget): void {
    this.responseWidgetMap.set(type, ResponseWidgetClass)
  }

  /**
   * 获取 Widget 类
   */
  getWidgetClass(type: string): typeof BaseWidget {
    const WidgetClass = this.widgetMap.get(type)
    if (!WidgetClass) {
      Logger.warn('WidgetFactory', `未知的 widget 类型: ${type}，使用 InputWidget`)
      return InputWidget
    }
    return WidgetClass
  }

  /**
   * 🔥 获取 Response Widget 类（用于响应参数渲染）
   * 如果该类型有对应的 Response Widget，返回它；否则返回 null
   */
  getResponseWidgetClass(type: string): typeof BaseWidget | null {
    return this.responseWidgetMap.get(type) || null
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

