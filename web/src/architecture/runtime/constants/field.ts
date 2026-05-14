/**
 * 字段相关常量
 * 
 * 🔥 统一管理字段相关的常量，避免硬编码
 * 🔥 符合依赖倒置原则：使用常量而非硬编码字符串
 * 
 * 使用场景：
 * - FieldCallback：用于判断字段是否有特定回调（如 OnSelectFuzzy）
 * - FieldValueMeta：用于标记字段值的来源和状态（如 _fromURL、_converted）
 */

/**
 * 字段回调类型常量
 */
export const FieldCallback = {
  ON_SELECT_FUZZY: 'OnSelectFuzzy',
  // 未来可以添加其他回调类型
} as const

/**
 * FieldValue meta 字段常量
 * 🔥 用于标记字段值的来源和状态
 */
export const FieldValueMeta = {
  FROM_URL: '_fromURL',           // 标记值来自 URL
  ORIGINAL_VALUE: '_originalValue', // 保存原始值（用于类型转换）
  CONVERTED: '_converted',        // 标记已进行类型转换
  DISPLAY_INFO: 'displayInfo',    // 显示信息（如 label）
  STATISTICS: 'statistics',       // 统计信息
} as const

