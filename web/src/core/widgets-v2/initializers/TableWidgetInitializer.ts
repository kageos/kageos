/**
 * TableWidget 初始化器
 * 
 * 🔥 组件自治：TableWidget 自己负责自己的初始化逻辑
 * 
 * 功能：
 * - 处理来自 URL 的 JSON 字符串解析（表格数据是数组）
 * - 递归处理表格行的嵌套字段的类型转换
 * - 调用子字段的初始化器
 */

import type { IWidgetInitializer, WidgetInitContext } from '../interfaces/IWidgetInitializer'
import type { FieldValue } from '../../types/field'
import { widgetInitializerRegistry } from './WidgetInitializerRegistry'
import { convertBasicType } from '../utils/typeConverter'
import { Logger } from '../../utils/logger'
import { FieldValueMeta } from '../../constants/field'

/**
 * TableWidget 初始化器
 * 
 * 🔥 组件自治：TableWidget 自己负责自己的初始化逻辑
 */
export class TableWidgetInitializer implements IWidgetInitializer {
  /**
   * 初始化 TableWidget
   * 
   * @param context 初始化上下文
   * @returns 初始化后的 FieldValue，如果不需要初始化则返回 null
   */
  async initialize(context: WidgetInitContext): Promise<FieldValue | null> {
    const { field, currentValue, functionDetail } = context
    
    console.log(`🔍 [TableWidgetInitializer] 开始初始化字段 ${field.code}`, {
      currentValue: {
        raw: currentValue.raw,
        display: currentValue.display,
        fromURL: !!(currentValue.meta && currentValue.meta._fromURL)
      },
      hasChildren: !!(field.children && field.children.length > 0),
      childrenCount: field.children?.length || 0,
      initSource: context.initSource
    })
    
    // 🔥 暂不支持 table 类型的 URL 回显（太复杂，后续通过快链支持）
    if (currentValue.meta && currentValue.meta[FieldValueMeta.FROM_URL]) {
      console.log(`🔍 [TableWidgetInitializer] 字段 ${field.code} 来自 URL，暂不支持 table 类型的 URL 回显，后续通过快链支持`)
      return null
    }
    
    // 🔥 暂不支持 table 类型的 URL 回显（太复杂，后续通过快链支持）
    // 已移除 URL 回显相关代码，保留初始化器结构以便未来扩展
    
    // 不是来自 URL，不需要初始化
    return null
  }
  
}

