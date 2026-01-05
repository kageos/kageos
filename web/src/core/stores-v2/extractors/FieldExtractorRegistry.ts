/**
 * 字段提取器注册表
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **字段值提取**：
 *    - 表单提交时需要提取所有字段的值
 *    - 不同字段类型有不同的提取逻辑
 *    - 支持嵌套结构（form、table）的递归提取
 * 
 * 2. **提取策略**：
 *    - 基础字段（input、select、number）：直接提取 `raw` 值
 *    - 多选字段（multiselect）：根据 `data.type` 决定返回格式（字符串或数组）
 *    - 表单字段（form/struct）：递归提取所有子字段的 `raw` 值
 *    - 表格字段（table）：提取数组，每个元素是对象
 * 
 * 3. **值格式规范**：
 *    - 所有字段值都是 `FieldValue` 格式：`{ raw, display, meta }`
 *    - 提交时只提取 `raw` 值（原始值）
 *    - `display` 值仅用于前端展示，不提交给后端
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **策略模式**：
 *    - 使用 `IFieldExtractor` 接口定义提取策略
 *    - 不同字段类型实现不同的提取器
 *    - 通过注册表管理提取器，支持扩展
 * 
 * 2. **依赖倒置原则**：
 *    - 注册表依赖 `IFieldExtractor` 接口，不依赖具体实现
 *    - 可以轻松替换或扩展提取器
 * 
 * 3. **递归提取**：
 *    - 嵌套结构（form、table）需要递归提取
 *    - 提取器可以调用注册表获取其他提取器
 *    - 支持任意深度的嵌套
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **getExtractor**：
 *    - 根据字段的 `widget.type` 获取对应的提取器
 *    - 如果没有匹配的提取器，使用默认提取器（BasicFieldExtractor）
 * 
 * 2. **extractField**：
 *    - 委托给对应的提取器提取字段值
 *    - 提取器可以递归调用注册表提取嵌套字段
 * 
 * 3. **注册提取器**：
 *    - 支持注册自定义提取器
 *    - 支持取消注册提取器
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **值提取规则**：
 *    - 只提取 `raw` 值，不提取 `display` 值
 *    - `null` 值也要包含在提交数据中（让后端验证必填字段）
 * 
 * 2. **嵌套结构**：
 *    - form 字段：递归提取所有子字段的 `raw` 值，组成对象
 *    - table 字段：提取数组，每个元素是对象，包含子字段的 `raw` 值
 * 
 * 3. **多选字段**：
 *    - 根据 `field.data.type` 决定返回格式
 *    - `type: "string"` → 返回逗号分隔的字符串
 *    - `type: "[]string"` → 返回数组
 * 
 * ============================================
 * 📚 相关文档
 * ============================================
 * 
 * - 表单值提取逻辑分析：`web/docs/表单值提取逻辑分析报告.md`
 * - 字段提取器接口：`web/src/core/stores-v2/extractors/FieldExtractor.ts`
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
  
  /**
   * 取消注册提取器
   * 
   * @param widgetType Widget 类型
   */
  unregisterExtractor(widgetType: string): void {
    this.extractorMap.delete(widgetType)
  }
}

// 导出全局单例（用于插件系统）
export const fieldExtractorRegistry = new FieldExtractorRegistry()

