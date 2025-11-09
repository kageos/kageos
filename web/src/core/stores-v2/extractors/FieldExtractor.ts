/**
 * 字段提取器接口（依赖倒置原则）
 * 🔥 抽象接口，不依赖具体实现
 * 
 * 功能：
 * - 定义字段值提取的抽象接口
 * - 支持不同类型的字段有自己的提取逻辑
 * - 方便扩展新的字段类型
 */

import type { FieldConfig, FieldValue } from '../../../types/field'

/**
 * 字段提取器接口
 */
export interface IFieldExtractor {
  /**
   * 提取字段值
   * @param field 字段配置
   * @param fieldPath 字段路径
   * @param getValue 获取字段值的函数
   * @param extractorRegistry 提取器注册表（用于递归调用）
   * @returns 提取的值
   */
  extract(
    field: FieldConfig,
    fieldPath: string,
    getValue: (path: string) => FieldValue | undefined,
    extractorRegistry: FieldExtractorRegistry
  ): any
}

/**
 * 提取器注册表接口
 */
export interface FieldExtractorRegistry {
  /**
   * 获取字段对应的提取器
   */
  getExtractor(field: FieldConfig): IFieldExtractor
  
  /**
   * 提取字段值（委托给对应的提取器）
   */
  extractField(field: FieldConfig, fieldPath: string, getValue: (path: string) => FieldValue | undefined): any
}

