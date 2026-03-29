/**
 * SelectWidget 初始化器
 * 
 * 🔥 组件自治：SelectWidget 自己负责自己的初始化逻辑
 * 
 * 功能：
 * - 检查是否需要初始化（是否有 OnSelectFuzzy 回调）
 * - 如果只有 raw 值（来自 URL），通过 by_value 查询获取 display 和 meta
 * - 如果已经有完整的 display 和 meta，则不需要初始化
 */

import type { IWidgetInitializer, WidgetInitContext } from '@/architecture/presentation/widgets/interfaces/IWidgetInitializer'
import type { FieldValue } from '@/architecture/domain/types'
import { selectFuzzy } from '@/api/function'
import { SelectFuzzyQueryType } from '@/core/constants/select'
import { DataType } from '@/core/constants/widget'
import { FieldCallback, FieldValueMeta } from '@/core/constants/field'
import { convertValueToType } from '@/architecture/presentation/widgets/utils/valueConverter'
import { convertBasicType, convertFormDataToRequestByType } from '@/architecture/presentation/widgets/utils/typeConverter'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { Logger } from '@/core/utils/logger'

/**
 * SelectWidget 初始化器
 * 
 * 🔥 组件自治：SelectWidget 自己负责自己的初始化逻辑
 */
export class SelectWidgetInitializer implements IWidgetInitializer {
  /**
   * 初始化 SelectWidget
   * 
   * @param context 初始化上下文
   * @returns 初始化后的 FieldValue，如果不需要初始化则返回 null
   */
  async initialize(context: WidgetInitContext): Promise<FieldValue | null> {
    const { field, currentValue, functionDetail, allFormData } = context
    
    console.log(`🔍 [SelectWidgetInitializer] 开始初始化字段 ${field.code}`, {
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
      const fieldType = field.data?.type || DataType.STRING
      
      console.log(`🔍 [SelectWidgetInitializer] 字段 ${field.code} 来自 URL，进行类型转换`, {
        originalValue,
        fieldType,
        currentRaw: currentValue.raw
      })
      
      // 🔥 使用统一的类型转换工具（避免硬编码）
      const convertedRaw = convertBasicType(originalValue, fieldType)
      
      processedValue = {
        raw: convertedRaw,
        display: String(originalValue),  // display 暂时使用原始字符串，后续通过回调获取
        meta: {
          ...currentValue.meta,
          [FieldValueMeta.CONVERTED]: true  // 标记已转换
        }
      }
      
      console.log(`✅ [SelectWidgetInitializer] 字段 ${field.code} 类型转换完成`, {
        originalValue,
        convertedRaw,
        fieldType
      })
    }
    
    // 1. 检查是否需要初始化
    // 如果字段没有 OnSelectFuzzy 回调，则不需要初始化（但已转换的值需要返回）
    if (!field.callbacks?.includes(FieldCallback.ON_SELECT_FUZZY)) {
      console.log(`🔍 [SelectWidgetInitializer] 字段 ${field.code} 没有 ${FieldCallback.ON_SELECT_FUZZY} 回调，跳过初始化`)
      // 🔥 如果进行了类型转换，返回转换后的值；否则返回 null
      return processedValue !== currentValue ? processedValue : null
    }
    
    // 2. 如果已经有完整的 display 和 meta，则不需要初始化
    // 🔥 优化：如果 display 存在且不等于 raw，说明已经有有意义的显示值，不需要初始化
    if (processedValue.display && 
        String(processedValue.display) !== String(processedValue.raw) && 
        processedValue.display !== '' &&
        processedValue.meta?.displayInfo) {
      console.log(`🔍 [SelectWidgetInitializer] 字段 ${field.code} 已有完整的 display 和 meta，跳过初始化`, {
        display: processedValue.display,
        raw: processedValue.raw,
        hasDisplayInfo: !!processedValue.meta?.displayInfo
      })
      return processedValue  // 返回处理后的值（可能包含类型转换）
    }
    
    // 🔥 如果 display 等于 raw 或为空，说明还没有有意义的显示值，需要初始化
    const displayEqualsRaw = processedValue.display && String(processedValue.display) === String(processedValue.raw)
    if (displayEqualsRaw || !processedValue.display || processedValue.display === '') {
      console.log(`🔍 [SelectWidgetInitializer] 字段 ${field.code} display 等于 raw 或为空，需要初始化`, {
        display: processedValue.display,
        raw: processedValue.raw,
        displayEqualsRaw
      })
    }
    
    // 3. 如果只有 raw 值（来自 URL 或默认值），需要通过 by_value 查询获取 display 和 meta
    if (processedValue.raw !== null && processedValue.raw !== undefined) {
      console.log(`🔍 [SelectWidgetInitializer] 字段 ${field.code} 只有 raw 值，需要通过 by_value 查询`, {
        rawValue: processedValue.raw
      })
      try {
        const valueType = field.data?.type || DataType.STRING
        let convertedValue: any = processedValue.raw
        
        // 类型转换：如果 raw 是字符串但类型不是 string，需要转换（可能已经在步骤 0 转换过了）
        if (typeof processedValue.raw === 'string' && valueType !== DataType.STRING && !processedValue.meta?.[FieldValueMeta.CONVERTED]) {
          convertedValue = convertValueToType(processedValue.raw, valueType, 'SelectWidgetInitializer')
        }
        
        // 🔥 构建请求参数（将 allFormData 转换为请求格式，并根据字段类型进行转换）
        // 使用统一的类型转换函数，确保所有字段都根据 field.data.type 正确转换
        const requestData = convertFormDataToRequestByType(allFormData, functionDetail)
        
        // 调用 OnSelectFuzzy 回调接口
        console.log(`🔍 [SelectWidgetInitializer] 调用 OnSelectFuzzy 回调接口`, {
          fieldCode: field.code,
          method: functionDetail.method || 'GET',
          router: functionDetail.router || '',
          convertedValue,
          valueType
        })
        
        const response = await selectFuzzy(
          functionDetail.method || 'GET',
          functionDetail.router || '',
          {
            code: field.code,
            type: SelectFuzzyQueryType.BY_VALUE,
            value: convertedValue,
            request: requestData,
            value_type: valueType
          }
        )
        
        console.log(`🔍 [SelectWidgetInitializer] OnSelectFuzzy 回调接口返回`, {
          fieldCode: field.code,
          hasError: !!response.error_msg,
          itemsCount: response.items?.length || 0
        })
        
        if (response.error_msg) {
          console.warn(`⚠️ [SelectWidgetInitializer] 字段 ${field.code} 回调接口返回错误`, {
            error: response.error_msg
          })
          return null  // 初始化失败，返回 null
        }
        
        // 找到匹配的选项
        if (response.items && Array.isArray(response.items) && response.items.length > 0) {
          const matchedItem = response.items.find((item: any) => {
            // 支持多种类型比较
            return item.value === processedValue.raw || 
                   String(item.value) === String(processedValue.raw)
          })
          
          if (matchedItem) {
            const initializedValue = createFieldValue(
              field,
              processedValue.raw,
              matchedItem.label || String(matchedItem.value),
              {
                ...processedValue.meta,
                displayInfo: matchedItem.display_info || matchedItem.displayInfo,
                statistics: response.statistics || {}
              }
            )
            
            console.log(`✅ [SelectWidgetInitializer] 字段 ${field.code} 初始化成功`, {
              raw: initializedValue.raw,
              display: initializedValue.display,
              hasDisplayInfo: !!initializedValue.meta?.displayInfo
            })
            
            // 构建初始化后的 FieldValue
            return initializedValue
          }
        }
        
        console.warn(`⚠️ [SelectWidgetInitializer] 字段 ${field.code} 未找到匹配的选项`, {
          rawValue: currentValue.raw,
          itemsCount: response.items?.length || 0
        })
        return null  // 未找到匹配的选项，返回 null
      } catch (error: any) {
        Logger.error('[SelectWidgetInitializer]', '初始化失败', {
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
