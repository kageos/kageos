/**
 * MultiSelectWidget 初始化器
 * 
 * 🔥 组件自治：MultiSelectWidget 自己负责自己的初始化逻辑
 * 
 * 功能：
 * - 检查是否需要初始化（是否有 OnSelectFuzzy 回调）
 * - 如果只有 raw 值（来自 URL），通过 by_values 查询获取 display 和 meta
 * - 如果已经有完整的 display 和 meta，则不需要初始化
 */

import type { IWidgetInitializer, WidgetInitContext } from '@/architecture/presentation/widgets/interfaces/IWidgetInitializer'
import type { FieldValue } from '@/architecture/domain/types'
import { selectFuzzy } from '@/api/function'
import { SelectFuzzyQueryType } from '@/core/constants/select'
import { DataType } from '@/core/constants/widget'
import { convertArrayType, convertFormDataToRequestByType, buildOptionMaps, getOptionLabelFromMap } from '@/architecture/presentation/widgets/utils/typeConverter'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { Logger } from '@/core/utils/logger'
import { FieldCallback } from '@/core/constants/field'
import { FieldValueMeta } from '@/core/constants/field'

/**
 * MultiSelectWidget 初始化器
 * 
 * 🔥 组件自治：MultiSelectWidget 自己负责自己的初始化逻辑
 */
export class MultiSelectWidgetInitializer implements IWidgetInitializer {
  /**
   * 初始化 MultiSelectWidget
   * 
   * @param context 初始化上下文
   * @returns 初始化后的 FieldValue，如果不需要初始化则返回 null
   */
  async initialize(context: WidgetInitContext): Promise<FieldValue | null> {
    const { field, currentValue, functionDetail, allFormData } = context
    
    console.log(`🔍 [MultiSelectWidgetInitializer] 开始初始化字段 ${field.code}`, {
      hasCallback: field.callbacks?.includes(FieldCallback.ON_SELECT_FUZZY),
      currentValue: {
        raw: currentValue.raw,
        display: currentValue.display,
        hasDisplayInfo: !!currentValue.meta?.displayInfo,
        fromURL: !!currentValue.meta?.[FieldValueMeta.FROM_URL]
      },
      initSource: context.initSource
    })
    
    // 🔥 步骤 0：处理来自 URL 的类型转换（组件自治）
    let processedValue = currentValue
    if (currentValue.meta?.[FieldValueMeta.FROM_URL] && currentValue.meta?.[FieldValueMeta.ORIGINAL_VALUE] !== undefined) {
      const originalValue = currentValue.meta[FieldValueMeta.ORIGINAL_VALUE]
      const fieldType = field.data?.type || DataType.STRINGS
      
      console.log(`🔍 [MultiSelectWidgetInitializer] 字段 ${field.code} 来自 URL，进行类型转换`, {
        originalValue,
        fieldType,
        currentRaw: currentValue.raw
      })
      
      // 🔥 使用统一的类型转换工具（避免硬编码）
      const convertedRaw = convertArrayType(originalValue, fieldType)
      
      // 数组类型的 display 使用逗号分隔的字符串
      const display = Array.isArray(convertedRaw) 
        ? convertedRaw.map(v => String(v)).join(',')
        : String(originalValue)
      
      processedValue = {
        raw: convertedRaw,
        display,
        meta: {
          ...currentValue.meta,
          [FieldValueMeta.CONVERTED]: true  // 标记已转换
        }
      }
      
      console.log(`✅ [MultiSelectWidgetInitializer] 字段 ${field.code} 类型转换完成`, {
        originalValue,
        convertedRaw,
        display,
        fieldType
      })
    }
    
    // 1. 检查是否需要初始化
    // 如果字段没有 OnSelectFuzzy 回调，则不需要初始化（但已转换的值需要返回）
    if (!field.callbacks?.includes(FieldCallback.ON_SELECT_FUZZY)) {
      console.log(`🔍 [MultiSelectWidgetInitializer] 字段 ${field.code} 没有 ${FieldCallback.ON_SELECT_FUZZY} 回调，跳过初始化`)
      // 🔥 如果进行了类型转换，返回转换后的值；否则返回 null
      return processedValue !== currentValue ? processedValue : null
    }
    
    // 2. 如果已经有完整的 display 和 meta，则不需要初始化
    if (processedValue.display && processedValue.meta?.displayInfo) {
      console.log(`🔍 [MultiSelectWidgetInitializer] 字段 ${field.code} 已有完整的 display 和 meta，跳过初始化`, {
        display: processedValue.display,
        hasDisplayInfo: !!processedValue.meta?.displayInfo
      })
      return processedValue  // 返回处理后的值（可能包含类型转换）
    }
    
    // 3. 如果只有 raw 值（来自 URL 或默认值），需要通过 by_values 查询获取 display 和 meta
    if (processedValue.raw !== null && processedValue.raw !== undefined) {
      console.log(`🔍 [MultiSelectWidgetInitializer] 字段 ${field.code} 只有 raw 值，需要通过 by_values 查询`, {
        rawValue: processedValue.raw,
        isArray: Array.isArray(processedValue.raw)
      })
      try {
        // 确保 raw 是数组
        const rawArray = Array.isArray(processedValue.raw) ? processedValue.raw : [processedValue.raw]
        if (rawArray.length === 0) {
          return null  // 空数组，不需要初始化
        }
        
        // 类型转换：根据 value_type 转换数组元素类型（可能已经在步骤 0 转换过了）
        const valueType = field.data?.type || DataType.STRINGS
        let convertedValue: any = rawArray
        
        // 🔥 如果还没有转换过，使用统一的类型转换工具
        if (valueType.startsWith('[]') && !processedValue.meta?.[FieldValueMeta.CONVERTED]) {
          convertedValue = convertArrayType(rawArray, valueType)
        } else {
          convertedValue = rawArray
        }
        
        // 🔥 构建请求参数（将 allFormData 转换为请求格式，并根据字段类型进行转换）
        // 使用统一的类型转换函数，确保所有字段都根据 field.data.type 正确转换
        const requestData = convertFormDataToRequestByType(allFormData, functionDetail)
        
        // 调用 OnSelectFuzzy 回调接口（使用 by_values）
        console.log(`🔍 [MultiSelectWidgetInitializer] 调用 OnSelectFuzzy 回调接口`, {
          fieldCode: field.code,
          method: functionDetail.method || 'POST',
          router: functionDetail.router || '',
          convertedValue,
          valueType,
          valuesCount: Array.isArray(convertedValue) ? convertedValue.length : 1
        })
        
        const response = await selectFuzzy(
          functionDetail.method || 'POST',
          functionDetail.router || '',
          {
            code: field.code,
            type: SelectFuzzyQueryType.BY_VALUES,
            value: convertedValue,
            request: requestData,
            value_type: valueType
          }
        )
        
        console.log(`🔍 [MultiSelectWidgetInitializer] OnSelectFuzzy 回调接口返回`, {
          fieldCode: field.code,
          hasError: !!response.error_msg,
          itemsCount: response.items?.length || 0
        })
        
        if (response.error_msg) {
          console.warn(`⚠️ [MultiSelectWidgetInitializer] 字段 ${field.code} 回调接口返回错误`, {
            error: response.error_msg
          })
          return null  // 初始化失败，返回 null
        }
        
        // 🔥 构建选项映射（value -> label），使用统一工具函数
        const { optionMap, displayInfoMap } = buildOptionMaps(response.items || [])
        
        // 🔥 使用转换后的值（convertedValue）去匹配选项，确保类型一致
        const finalRawValue = Array.isArray(convertedValue) ? convertedValue : [convertedValue]
        
        // 构建 display 字符串（逗号分隔的标签），使用统一工具函数
        const displayLabels = finalRawValue.map((val: any) => {
          return getOptionLabelFromMap(optionMap, val)
        })
        const display = displayLabels.join(', ')
        
        // 构建 displayInfo（数组形式，每个元素对应一个值）
        const displayInfoArray = finalRawValue.map((val: any) => {
          return displayInfoMap.get(val) || null
        })
        
        const initializedValue = createFieldValue(
          field,
          convertedValue,  // 🔥 使用转换后的值作为 raw，确保类型正确
          display,
          {
            ...processedValue.meta,  // 🔥 使用 processedValue.meta，保留转换标记
            displayInfo: displayInfoArray.length > 0 ? displayInfoArray : undefined,
            statistics: response.statistics || {}
          }
        )
        
        console.log(`✅ [MultiSelectWidgetInitializer] 字段 ${field.code} 初始化成功`, {
          raw: initializedValue.raw,
          display: initializedValue.display,
          hasDisplayInfo: !!initializedValue.meta?.displayInfo,
          displayInfoCount: Array.isArray(initializedValue.meta?.displayInfo) ? initializedValue.meta.displayInfo.length : 0
        })
        
        // 构建初始化后的 FieldValue
        return initializedValue
      } catch (error: any) {
        Logger.error('[MultiSelectWidgetInitializer]', '初始化失败', {
          fieldCode: field.code,
          error: error?.message || error
        })
        return null  // 初始化失败，返回 null
      }
    }
    
    // 4. 没有 raw 值，不需要初始化
    return null
  }
  
}
