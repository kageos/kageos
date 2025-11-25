/**
 * 字段提取器注册表
 * 🔥 使用策略模式，遵循依赖倒置原则
 * 
 * 功能：
 * - 注册和管理不同类型的字段提取器
 * - 根据字段类型返回对应的提取器
 * - 方便扩展新的字段类型
 */

import type { FieldConfig } from '../../../types/field'
import type { IFieldExtractor, FieldExtractorRegistry as IFieldExtractorRegistry } from './FieldExtractor'
import { BasicFieldExtractor } from './BasicFieldExtractor'
import { MultiSelectFieldExtractor } from './MultiSelectFieldExtractor'
import { FormFieldExtractor } from './FormFieldExtractor'
import { TableFieldExtractor } from './TableFieldExtractor'

export class FieldExtractorRegistry implements IFieldExtractorRegistry {
  private extractorMap: Map<string, IFieldExtractor> = new Map()
  private defaultExtractor: IFieldExtractor = new BasicFieldExtractor()
  
  constructor() {
    // 注册默认提取器
    this.registerExtractor('form', new FormFieldExtractor())
    this.registerExtractor('table', new TableFieldExtractor())
    this.registerExtractor('multiselect', new MultiSelectFieldExtractor())
  }
  
  /**
   * 注册提取器
   */
  registerExtractor(widgetType: string, extractor: IFieldExtractor): void {
    this.extractorMap.set(widgetType, extractor)
  }
  
  /**
   * 获取字段对应的提取器
   */
  getExtractor(field: FieldConfig): IFieldExtractor {
    // 优先根据 widget.type 判断
    if (field.widget?.type) {
      const extractor = this.extractorMap.get(field.widget.type)
      if (extractor) {
        return extractor
      }
    }
    
    // 其次根据 data.type 判断
    if (field.data?.type === '[]string') {
      return this.extractorMap.get('multiselect') || this.defaultExtractor
    }
    
    // 默认使用基础提取器
    return this.defaultExtractor
  }
  
  /**
   * 提取字段值（委托给对应的提取器）
   */
  extractField(
    field: FieldConfig,
    fieldPath: string,
    getValue: (path: string) => any
  ): any {
    const extractor = this.getExtractor(field)
    return extractor.extract(field, fieldPath, getValue, this)
  }
}

