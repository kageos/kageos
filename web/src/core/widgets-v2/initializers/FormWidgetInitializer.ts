/**
 * FormWidget 初始化器
 * 
 * 🔥 组件自治：FormWidget 自己负责自己的初始化逻辑
 * 
 * 功能：
 * - 处理来自 URL 的 JSON 字符串解析
 * - 递归处理嵌套字段的类型转换
 * - 调用子字段的初始化器
 */

import type { IWidgetInitializer, WidgetInitContext } from '../interfaces/IWidgetInitializer'
import type { FieldValue, FieldConfig } from '../../types/field'
import { widgetInitializerRegistry } from './WidgetInitializerRegistry'
import { convertBasicType } from '../utils/typeConverter'
import { Logger } from '../../utils/logger'
import { FieldValueMeta } from '../../constants/field'

/**
 * FormWidget 初始化器
 * 
 * 🔥 组件自治：FormWidget 自己负责自己的初始化逻辑
 */
export class FormWidgetInitializer implements IWidgetInitializer {
  /**
   * 初始化 FormWidget
   * 
   * @param context 初始化上下文
   * @returns 初始化后的 FieldValue，如果不需要初始化则返回 null
   */
  async initialize(context: WidgetInitContext): Promise<FieldValue | null> {
    const { field, currentValue, functionDetail } = context
    
    console.log(`🔍 [FormWidgetInitializer] 开始初始化字段 ${field.code}`, {
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
    
    // 🔥 暂不支持 form 类型的 URL 回显（太复杂，后续通过快链支持）
    if (currentValue.meta && currentValue.meta[FieldValueMeta.FROM_URL]) {
      console.log(`🔍 [FormWidgetInitializer] 字段 ${field.code} 来自 URL，暂不支持 form 类型的 URL 回显，后续通过快链支持`)
      return null
    }
    
    // 🔥 处理快链数据：需要递归初始化子字段
    if (currentValue.meta && currentValue.meta._fromQuickLink) {
      if (!currentValue.raw || typeof currentValue.raw !== 'object' || Array.isArray(currentValue.raw)) {
        return null
      }
      
      const subFields = field.children || []
      if (subFields.length === 0) {
        return null
      }
      
      // 🔥 需要将子字段的值保存到 formDataStore 的子路径中
      const { useFormDataStore } = await import('../../stores-v2/formData')
      const formDataStore = useFormDataStore()
      
      // 递归初始化所有子字段
      const initializedFormData: Record<string, any> = {}
      
      await Promise.all(subFields.map(async (subField) => {
        try {
          const subRawValue = currentValue.raw[subField.code]
          
          // 🔥 构建子字段的完整路径（支持嵌套 form）
          const basePath = context.fieldPath || field.code
          const subFieldPath = `${basePath}.${subField.code}`
          
          // 创建子字段的初始化上下文
          const subFieldContext: WidgetInitContext = {
            field: subField,
            currentValue: {
              raw: subRawValue,
              display: subRawValue !== null && subRawValue !== undefined ? String(subRawValue) : '',
              meta: {
                ...currentValue.meta,
                _fromQuickLink: true
              }
            },
            allFormData: context.allFormData,
            functionDetail,
            initSource: context.initSource,
            fieldPath: subFieldPath  // 🔥 传递完整路径给子字段
          }
          
          // 调用子字段的初始化器
          const initializedValue = await widgetInitializerRegistry.initialize(subFieldContext)
          
          // 🔥 将子字段的值保存到 formDataStore 的子路径中
          if (initializedValue) {
            formDataStore.setValue(subFieldPath, initializedValue)
            initializedFormData[subField.code] = initializedValue.raw
          } else {
            // 如果初始化器返回 null，使用基本类型转换
            const convertedValue = convertBasicType(subRawValue, subField.data?.type || 'string')
            formDataStore.setValue(subFieldPath, {
              raw: convertedValue,
              display: convertedValue !== null && convertedValue !== undefined ? String(convertedValue) : '',
              meta: {
                ...currentValue.meta,
                _fromQuickLink: true
              }
            })
            initializedFormData[subField.code] = convertedValue
          }
        } catch (error: any) {
          // 🔥 如果子字段初始化失败，记录错误但继续处理其他字段
          Logger.warn('[FormWidgetInitializer]', `子字段 ${subField.code} 初始化失败`, {
            fieldCode: field.code,
            subFieldCode: subField.code,
            error: error?.message || error
          })
          
          // 使用原始值作为降级方案
          const subRawValue = currentValue.raw[subField.code]
          // 🔥 构建子字段的完整路径（支持嵌套 form）
          const basePath = context.fieldPath || field.code
          const subFieldPath = `${basePath}.${subField.code}`
          const convertedValue = convertBasicType(subRawValue, subField.data?.type || 'string')
          formDataStore.setValue(subFieldPath, {
            raw: convertedValue,
            display: convertedValue !== null && convertedValue !== undefined ? String(convertedValue) : '',
            meta: {
              ...currentValue.meta,
              _fromQuickLink: true
            }
          })
          initializedFormData[subField.code] = convertedValue
        }
      }))
      
      // 返回初始化后的值
      return {
        raw: initializedFormData,
        display: JSON.stringify(initializedFormData),
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

