/**
 * 条件渲染评估器
 *
 * 基于 validator/v10 的存在性规则，统一判断字段的显示、必填和排除状态。
 */

import type { FieldConfig } from '@/architecture/domain/types/field'
import type { FormValueReader } from '@/architecture/domain/validation'
import { getFieldPresenceState as resolveFieldPresenceState } from '@/architecture/domain/validation/utils/presenceRules'

export function getFieldPresenceState(
  field: FieldConfig,
  formManager: FormValueReader,
  allFields: FieldConfig[],
  fieldPath: string = field.code
) {
  return resolveFieldPresenceState(field, {
    formManager,
    allFields,
    fieldPath,
  })
}

export function shouldShowField(
  field: FieldConfig,
  formManager: FormValueReader,
  allFields: FieldConfig[],
  fieldPath: string = field.code
): boolean {
  return getFieldPresenceState(field, formManager, allFields, fieldPath).visible
}

export function isFieldCurrentlyRequired(
  field: FieldConfig,
  formManager: FormValueReader,
  allFields: FieldConfig[],
  fieldPath: string = field.code
): boolean {
  return getFieldPresenceState(field, formManager, allFields, fieldPath).required
}
