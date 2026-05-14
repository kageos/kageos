/**
 * Table 搜索运行时门面
 *
 * 目的：
 * - 为表格领域层提供稳定的搜索/差异计算入口
 * - 收口历史上散落在 utils 下的表格搜索辅助函数
 * - 后续若要迁移实现，只需要调整这里，不必再改 Domain 层
 */

export { getChangedFields } from '@/architecture/runtime/utils/objectDiff'
