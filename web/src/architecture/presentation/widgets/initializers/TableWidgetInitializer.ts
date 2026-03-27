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

import type { IWidgetInitializer, WidgetInitContext } from '@/architecture/presentation/widgets/interfaces/IWidgetInitializer'
import type { FieldValue } from '@/architecture/domain/types'
import { widgetInitializerRegistry } from './WidgetInitializerRegistry'
import { convertBasicType } from '@/architecture/presentation/widgets/utils/typeConverter'
import { Logger } from '@/core/utils/logger'
import { FieldValueMeta } from '@/core/constants/field'
import { useFormDataStore } from '@/core/stores-v2/formData'

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
        fromURL: !!(currentValue.meta && currentValue.meta[FieldValueMeta.FROM_URL]),
        fromQuickLink: !!(currentValue.meta && currentValue.meta._fromQuickLink)
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
    
    // 🔥 处理快链数据：需要递归初始化子字段
    if (currentValue.meta && currentValue.meta._fromQuickLink) {
      if (!Array.isArray(currentValue.raw)) {
        return null
      }
      
      const itemFields = field.children || []
      if (itemFields.length === 0) {
        return null
      }
      
      // 过滤掉空行（所有字段都为 null/undefined 的行）
      const validRows = currentValue.raw.filter((row: any) => {
        if (!row || typeof row !== 'object') {
          return false
        }
        // 检查行中是否有任何非空字段
        return Object.values(row).some((val: any) => val !== null && val !== undefined && val !== '')
      })
      
      if (validRows.length === 0) {
        return null
      }
      
      // 🔥 需要将子字段的值保存到 formDataStore 的子路径中
      const formDataStore = useFormDataStore()
      
      // 递归初始化每一行的子字段
      const initializedRows = await Promise.all(validRows.map(async (row: any, rowIndex: number) => {
        const rowData: Record<string, any> = {}
        
        await Promise.all(itemFields.map(async (itemField) => {
          const itemRawValue = row[itemField.code]
          
          // 🔥 构建子字段的完整路径（支持嵌套 table）
          const basePath = context.fieldPath || field.code
          const itemFieldPath = `${basePath}[${rowIndex}].${itemField.code}`
          
          // 创建子字段的初始化上下文
          const subFieldContext: WidgetInitContext = {
            field: itemField,
            currentValue: {
              raw: itemRawValue,
              display: itemRawValue !== null && itemRawValue !== undefined ? String(itemRawValue) : '',
              meta: {
                ...currentValue.meta,
                _fromQuickLink: true
              }
            },
            allFormData: context.allFormData,
            functionDetail,
            initSource: context.initSource,
            fieldPath: itemFieldPath  // 🔥 传递完整路径给子字段
          }
          
          // 调用子字段的初始化器
          const initializedValue = await widgetInitializerRegistry.initialize(subFieldContext)
          
          // 🔥 将子字段的值保存到 formDataStore 的子路径中
          if (initializedValue) {
            formDataStore.setValue(itemFieldPath, initializedValue)
            rowData[itemField.code] = initializedValue.raw
          } else {
            // 如果初始化器返回 null，使用基本类型转换
            const convertedValue = convertBasicType(itemRawValue, itemField.data?.type || 'string')
            formDataStore.setValue(itemFieldPath, {
              raw: convertedValue,
              display: convertedValue !== null && convertedValue !== undefined ? String(convertedValue) : '',
              meta: {
                ...currentValue.meta,
                _fromQuickLink: true
              }
            })
            rowData[itemField.code] = convertedValue
          }
        }))
        
        return rowData
      }))
      
      // 返回初始化后的值
      return {
        raw: initializedRows,
        display: `共 ${initializedRows.length} 条`,
        meta: {
          ...currentValue.meta,
          _fromQuickLink: true
        }
      }
    }
    
    // 不是来自 URL 或快链，不需要初始化
    return null
  }
  
}
