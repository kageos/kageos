import type { ReactiveFormDataManager } from '../../managers/ReactiveFormDataManager'
import type { FieldConfig, FieldValue } from '../../types/field'
import type { ValidationContext, ValidationRule } from '../types'
import {
  findFieldByCode,
  findFieldByPath,
  isEmpty,
  resolveReferencedFieldPath,
} from './fieldUtils'

// validator/v10 中与“字段是否出现、是否必填”相关的规则集合。
// 前端用这组规则同时驱动动态显示、动态必填和 excluded 提交清理。
export type PresenceRuleType =
  | 'required'
  | 'required_if'
  | 'required_unless'
  | 'required_with'
  | 'required_with_all'
  | 'required_without'
  | 'required_without_all'
  | 'excluded_if'
  | 'excluded_unless'
  | 'excluded_with'
  | 'excluded_with_all'
  | 'excluded_without'
  | 'excluded_without_all'

export interface ValidationRuleRef {
  field: string
  value?: string
}

export interface PresenceEvaluationContext {
  formManager: Pick<ReactiveFormDataManager, 'getValue'> & {
    hasValue?: (fieldPath: string) => boolean
  }
  fieldPath: string
  allFields: FieldConfig[]
}

export interface FieldPresenceState {
  visible: boolean
  required: boolean
  excluded: boolean
  activeRules: ValidationRule[]
}

const VALUE_PAIR_RULE_TYPES = new Set<PresenceRuleType>([
  'required_if',
  'required_unless',
  'excluded_if',
  'excluded_unless',
])

const FIELD_LIST_RULE_TYPES = new Set<PresenceRuleType>([
  'required_with',
  'required_with_all',
  'required_without',
  'required_without_all',
  'excluded_with',
  'excluded_with_all',
  'excluded_without',
  'excluded_without_all',
])

const CONDITIONAL_REQUIRED_RULE_TYPES = new Set<PresenceRuleType>([
  'required_if',
  'required_unless',
  'required_with',
  'required_with_all',
  'required_without',
  'required_without_all',
])

const EXCLUDED_RULE_TYPES = new Set<PresenceRuleType>([
  'excluded_if',
  'excluded_unless',
  'excluded_with',
  'excluded_with_all',
  'excluded_without',
  'excluded_without_all',
])

function toPresenceContext(context: PresenceEvaluationContext | ValidationContext): PresenceEvaluationContext {
  return {
    formManager: context.formManager,
    fieldPath: context.fieldPath,
    allFields: context.allFields,
  }
}

export function buildFieldNameMap(fields: FieldConfig[]): Map<string, string> {
  const map = new Map<string, string>()

  const walk = (fieldList: FieldConfig[]) => {
    fieldList.forEach((field) => {
      if (field.field_name && field.code) {
        map.set(field.field_name, field.code)
      }

      if (field.children?.length) {
        walk(field.children)
      }
    })
  }

  walk(fields)
  return map
}

export function isPresenceRuleType(type: string): type is PresenceRuleType {
  return type === 'required'
    || VALUE_PAIR_RULE_TYPES.has(type as PresenceRuleType)
    || FIELD_LIST_RULE_TYPES.has(type as PresenceRuleType)
}

export function isConditionalPresenceRuleType(type: string): boolean {
  return type !== 'required' && isPresenceRuleType(type)
}

export function isExcludedRuleType(type: string): boolean {
  return EXCLUDED_RULE_TYPES.has(type as PresenceRuleType)
}

export function isRequiredPresenceRuleType(type: string): boolean {
  return type === 'required' || CONDITIONAL_REQUIRED_RULE_TYPES.has(type as PresenceRuleType)
}

export function parsePresenceRule(
  type: string,
  value: string,
  fieldNameMap: Map<string, string>
): Pick<ValidationRule, 'field' | 'value' | 'fields' | 'refs'> | null {
  if (!isConditionalPresenceRuleType(type)) {
    return null
  }

  const tokens = value.split(/\s+/).map((token) => token.trim()).filter(Boolean)
  if (tokens.length === 0) {
    return null
  }

  if (VALUE_PAIR_RULE_TYPES.has(type as PresenceRuleType)) {
    // required_if / excluded_if 这类规则使用 "Field value" 成对参数；
    // 多条件时按 "Field1 value1 Field2 value2" 继续展开。
    if (tokens.length % 2 !== 0) {
      return null
    }

    const refs: ValidationRuleRef[] = []
    for (let index = 0; index < tokens.length; index += 2) {
      const fieldName = tokens[index]
      const expectedValue = tokens[index + 1]
      if (!fieldName || expectedValue === undefined) {
        return null
      }

      refs.push({
        field: fieldNameMap.get(fieldName) || fieldName,
        value: expectedValue,
      })
    }

    return {
      field: refs[0]?.field,
      value: refs[0]?.value,
      refs,
    }
  }

  const fields = tokens.map((fieldName) => fieldNameMap.get(fieldName) || fieldName)
  return {
    field: fields[0],
    fields,
    refs: fields.map((field) => ({ field })),
  }
}

export function getRuleRefs(rule: ValidationRule): ValidationRuleRef[] {
  if (rule.refs?.length) {
    return rule.refs
  }

  if (rule.fields?.length) {
    return rule.fields.map((field) => ({ field }))
  }

  if (rule.field) {
    return [{
      field: rule.field,
      value: typeof rule.value === 'number' || typeof rule.value === 'string'
        ? String(rule.value)
        : undefined,
    }]
  }

  return []
}

function createRuleContext(
  context: PresenceEvaluationContext,
  ruleField: string
): ValidationContext {
  return {
    formManager: context.formManager as ReactiveFormDataManager,
    fieldPath: context.fieldPath,
    allFields: context.allFields,
  }
}

function resolveReferencedField(
  ref: ValidationRuleRef,
  rawContext: PresenceEvaluationContext | ValidationContext
): { fieldPath: string; field: FieldConfig | null; value: FieldValue } {
  const context = toPresenceContext(rawContext)
  const validationContext = createRuleContext(context, ref.field)
  const fieldPath = resolveReferencedFieldPath(validationContext, ref.field)
  const field = findFieldByPath(context.allFields, fieldPath) || findFieldByCode(context.allFields, ref.field)
  const value = context.formManager.getValue(fieldPath)

  return { fieldPath, field, value }
}

export function isReferencedFieldPresent(
  ref: ValidationRuleRef,
  context: PresenceEvaluationContext | ValidationContext
): boolean {
  const resolved = resolveReferencedField(ref, context)
  return !isEmpty(resolved.value, resolved.field || undefined)
}

export function isReferencedFieldEqual(
  ref: ValidationRuleRef,
  context: PresenceEvaluationContext | ValidationContext
): boolean {
  if (ref.value === undefined) {
    return false
  }

  const resolved = resolveReferencedField(ref, context)
  const actualValue = resolved.value.raw
  const expectedValue = ref.value

  if (actualValue === null || actualValue === undefined) {
    return false
  }

  if (typeof actualValue === 'boolean') {
    return String(actualValue) === expectedValue || actualValue === (expectedValue === 'true')
  }

  if (typeof actualValue === 'number') {
    const expectedNum = Number(expectedValue)
    return !Number.isNaN(expectedNum) && actualValue === expectedNum
  }

  return String(actualValue) === expectedValue
}

export function evaluatePresenceRule(
  rule: ValidationRule,
  rawContext: PresenceEvaluationContext | ValidationContext
): boolean {
  const refs = getRuleRefs(rule)
  if (rule.type === 'required') {
    return true
  }

  if (refs.length === 0) {
    return false
  }

  switch (rule.type) {
    case 'required_if':
    case 'excluded_if':
      return refs.every((ref) => isReferencedFieldEqual(ref, rawContext))
    case 'required_unless':
    case 'excluded_unless':
      return !refs.every((ref) => isReferencedFieldEqual(ref, rawContext))
    case 'required_with':
    case 'excluded_with':
      return refs.some((ref) => isReferencedFieldPresent(ref, rawContext))
    case 'required_with_all':
    case 'excluded_with_all':
      return refs.every((ref) => isReferencedFieldPresent(ref, rawContext))
    case 'required_without':
    case 'excluded_without':
      return refs.some((ref) => !isReferencedFieldPresent(ref, rawContext))
    case 'required_without_all':
    case 'excluded_without_all':
      return refs.every((ref) => !isReferencedFieldPresent(ref, rawContext))
    default:
      return false
  }
}

export function parsePresenceRules(
  validation: string,
  allFields: FieldConfig[]
): ValidationRule[] {
  const rules: ValidationRule[] = []
  const fieldNameMap = buildFieldNameMap(allFields)

  validation
    .split(',')
    .map((part) => part.trim())
    .forEach((part) => {
      if (!part || part === 'omitempty') {
        return
      }

      if (part === 'required') {
        rules.push({ type: 'required' })
        return
      }

      if (!part.includes('=')) {
        return
      }

      const [type, value] = part.split('=', 2)
      if (!type || value === undefined) {
        return
      }

      const typeTrimmed = type.trim()
      const valueTrimmed = value.trim()
      if (!isConditionalPresenceRuleType(typeTrimmed)) {
        return
      }

      const parsed = parsePresenceRule(typeTrimmed, valueTrimmed, fieldNameMap)
      if (!parsed) {
        return
      }

      rules.push({
        type: typeTrimmed,
        ...parsed,
      })
    })

  return rules
}

export function getFieldPresenceState(
  field: FieldConfig,
  context: PresenceEvaluationContext
): FieldPresenceState {
  if (!field.validation) {
    return {
      visible: true,
      required: false,
      excluded: false,
      activeRules: [],
    }
  }

  const rules = parsePresenceRules(field.validation, context.allFields)
  const activeRules = rules.filter((rule) => evaluatePresenceRule(rule, context))
  const hasUnconditionalRequired = rules.some((rule) => rule.type === 'required')
  const hasConditionalRequired = rules.some((rule) => CONDITIONAL_REQUIRED_RULE_TYPES.has(rule.type as PresenceRuleType))
  // excluded 的优先级高于 required：一旦字段在当前条件下被排除，
  // 前端应隐藏它，并在提交时把它从 payload 中剔除。
  const excluded = activeRules.some((rule) => isExcludedRuleType(rule.type))
  const required = !excluded && (hasUnconditionalRequired || activeRules.some((rule) => CONDITIONAL_REQUIRED_RULE_TYPES.has(rule.type as PresenceRuleType)))
  const visible = !excluded && (hasUnconditionalRequired || !hasConditionalRequired || required)

  return {
    visible,
    required,
    excluded,
    activeRules,
  }
}

export function sanitizeExcludedSubmitData(
  fields: FieldConfig[],
  submitData: Record<string, any>,
  context: Pick<PresenceEvaluationContext, 'formManager'>
): Record<string, any> {
  // 提交前再做一次递归剔除，避免“UI 已隐藏但 payload 还残留旧值”。
  return sanitizeScopedSubmitData(fields, submitData, {
    formManager: context.formManager,
    fieldPath: '',
    allFields: fields,
  })
}

function sanitizeScopedSubmitData(
  fields: FieldConfig[],
  submitData: Record<string, any>,
  context: PresenceEvaluationContext
): Record<string, any> {
  const sanitizedData: Record<string, any> = { ...submitData }

  fields.forEach((field) => {
    const nextFieldPath = context.fieldPath ? `${context.fieldPath}.${field.code}` : field.code
    const presenceState = getFieldPresenceState(field, {
      formManager: context.formManager,
      fieldPath: nextFieldPath,
      allFields: fields,
    })

    if (presenceState.excluded) {
      delete sanitizedData[field.code]
      return
    }

    const currentValue = sanitizedData[field.code]
    if (!field.children?.length || currentValue === null || currentValue === undefined) {
      return
    }

    if (field.widget?.type === 'form' && typeof currentValue === 'object' && !Array.isArray(currentValue)) {
      sanitizedData[field.code] = sanitizeScopedSubmitData(field.children, currentValue, {
        formManager: context.formManager,
        fieldPath: nextFieldPath,
        allFields: field.children,
      })
      return
    }

    if (field.widget?.type === 'table' && Array.isArray(currentValue)) {
      sanitizedData[field.code] = currentValue.map((row, rowIndex) => {
        if (!row || typeof row !== 'object' || Array.isArray(row)) {
          return row
        }

        return sanitizeScopedSubmitData(field.children || [], row, {
          formManager: context.formManager,
          fieldPath: `${nextFieldPath}[${rowIndex}]`,
          allFields: field.children || [],
        })
      })
    }
  })

  return sanitizedData
}
