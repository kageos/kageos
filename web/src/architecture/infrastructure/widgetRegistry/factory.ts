/**
 * widgetComponentFactory 实例导出
 * 
 * ============================================
 * ⚠️ 重要：为什么需要单独文件？
 * ============================================
 * 
 * 🔥 问题：循环依赖（Circular Dependency）
 * 
 * 如果 widgetComponentFactory 在 index.ts 中创建和导出，会导致循环依赖：
 * 
 *   1. index.ts 在顶层导入 FormWidget 和 TableWidget（用于注册）
 *   2. FormWidget.vue 导入 widgetComponentFactory from 'widgetRegistry'
 *   3. TableWidget.vue 导入 widgetComponentFactory from 'widgetRegistry'
 *   4. widgetRegistry 指向 index.ts
 * 
 *   循环依赖链：
 *   index.ts → FormWidget.vue → widgetRegistry/index.ts → ...
 *   index.ts → TableWidget.vue → widgetRegistry/index.ts → ...
 * 
 *   结果：运行时错误 "Cannot access 'FormWidget' before initialization"
 * 
 * ============================================
 * ✅ 解决方案
 * ============================================
 * 
 * 1. **分离实例创建**：
 *    - 将 widgetComponentFactory 的创建移到单独的 factory.ts
 *    - factory.ts 只创建实例，不导入任何组件
 * 
 * 2. **延迟加载组件**：
 *    - index.ts 在函数内部使用动态 import() 延迟加载 FormWidget 和 TableWidget
 *    - 这样在导入时，widgetComponentFactory 已经初始化完成
 * 
 * 3. **重新导出**：
 *    - index.ts 从 factory.ts 导入并重新导出 widgetComponentFactory
 *    - FormWidget 和 TableWidget 通过 index.ts 的重新导出导入（保持导入路径不变）
 * 
 * ============================================
 * 📝 依赖关系图
 * ============================================
 * 
 *   factory.ts
 *     ↓ (创建实例)
 *   widgetComponentFactory
 *     ↑ (导入)
 *   index.ts → (重新导出) → FormWidget/TableWidget
 *     ↓ (动态导入)
 *   FormWidget.vue / TableWidget.vue
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **不要在这里导入任何组件**：
 *    - factory.ts 只能导入 WidgetComponentFactory 类
 *    - 不能导入任何 Vue 组件，否则会重新引入循环依赖
 * 
 * 2. **保持导入路径**：
 *    - FormWidget 和 TableWidget 仍然从 'widgetRegistry' 导入
 *    - 通过 index.ts 的重新导出，保持向后兼容
 * 
 * 3. **初始化时机**：
 *    - widgetComponentFactory 在模块加载时立即创建
 *    - 组件注册在 initializeWidgetComponentFactory() 中异步完成
 */

import { WidgetComponentFactory } from './WidgetComponentFactory'

// 创建并导出工厂实例
export const widgetComponentFactory = new WidgetComponentFactory()

