/**
 * widgetComponentFactory 实例导出
 * 
 * 🔥 单独文件导出，避免循环依赖
 * 
 * 原因：
 * - FormWidget 和 TableWidget 需要导入 widgetComponentFactory
 * - index.ts 需要导入 FormWidget 和 TableWidget 来注册它们
 * - 如果都在 index.ts 中，会导致循环依赖
 * 
 * 解决方案：
 * - 将 widgetComponentFactory 的创建和导出移到单独文件
 * - FormWidget 和 TableWidget 从这个文件导入
 * - index.ts 也从这里导入，然后注册组件
 */

import { WidgetComponentFactory } from './WidgetComponentFactory'

// 创建并导出工厂实例
export const widgetComponentFactory = new WidgetComponentFactory()

