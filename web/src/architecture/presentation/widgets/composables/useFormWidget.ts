/**
 * useFormWidget - FormWidget 组合式函数
 * 🔥 统一架构组件
 * 
 * 功能：
 * - 提取 FormWidget 的共享逻辑
 * - 处理子字段的递归渲染
 * - 处理条件渲染
 */

import { computed } from 'vue'
import type { WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/presentation/context/formRuntimeContext'
import { createAutoFieldValue, createEmptyRawFieldValue } from '@/architecture/domain/utils/createFieldValue'
import { getFieldPresenceState } from '@/architecture/domain/utils/conditionEvaluator'
import { syncFormContainerValue } from '@/architecture/domain/utils/containerValue'
import { clearScopedDependentFields } from '@/architecture/domain/utils/dependency'
import { applyScopedPresenceEffects } from '@/architecture/domain/utils/presenceEffects'
import type { FormValueReader } from '@/architecture/domain/validation'

function isPlainObject(value: unknown): value is Record<string, any> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function hasMeaningfulFieldValue(value: any): boolean {
  if (!value) {
    return false
  }
  return (value.raw !== null && value.raw !== undefined)
    || value.display !== ''
    || Object.keys(value.meta || {}).length > 0
}

function readObjectPath(source: Record<string, any>, path: string): any {
  if (!path) {
    return source
  }

  const segments = path.match(/([^[.\]]+)|\[(\d+)\]/g) || []
  let current: any = source

  for (const segment of segments) {
    if (current === null || current === undefined) {
      return undefined
    }

    if (segment.startsWith('[')) {
      const index = Number(segment.slice(1, -1))
      current = Array.isArray(current) ? current[index] : undefined
    } else {
      current = current[segment]
    }
  }

  return current
}

export function useFormWidget(props: WidgetComponentProps) {
  const formDataStore = useFormDataStore()
  const rawObject = computed(() => {
    if (formDataStore.data.has(props.fieldPath)) {
      const storeValue = formDataStore.getValue(props.fieldPath)
      if (isPlainObject(storeValue?.raw)) {
        return storeValue.raw
      }
    }

    return isPlainObject(props.value?.raw) ? props.value.raw : null
  })
  
  // 子字段列表
  const subFields = computed(() => {
    return props.field.children || []
  })

  function resolveScopedPath(fieldCodeOrPath: string): string {
    if (!fieldCodeOrPath) {
      return fieldCodeOrPath
    }

    if (fieldCodeOrPath === props.fieldPath) {
      return fieldCodeOrPath
    }

    if (fieldCodeOrPath.startsWith(`${props.fieldPath}.`) || fieldCodeOrPath.startsWith(`${props.fieldPath}[`)) {
      return fieldCodeOrPath
    }

    if (fieldCodeOrPath.includes('.') || fieldCodeOrPath.includes('[')) {
      return fieldCodeOrPath
    }

    return `${props.fieldPath}.${fieldCodeOrPath}`
  }

  function resolveRawRelativePath(fieldCodeOrPath: string): string {
    if (!fieldCodeOrPath) {
      return fieldCodeOrPath
    }

    if (fieldCodeOrPath.startsWith(`${props.fieldPath}.`)) {
      return fieldCodeOrPath.slice(props.fieldPath.length + 1)
    }

    if (fieldCodeOrPath.startsWith(`${props.fieldPath}[`)) {
      return fieldCodeOrPath.slice(props.fieldPath.length)
    }

    return fieldCodeOrPath
  }

  function getScopedFieldValue(fieldCodeOrPath: string): any {
    const scopedPath = resolveScopedPath(fieldCodeOrPath)
    const storeValue = formDataStore.getValue(scopedPath)
    const hasStoredValue = formDataStore.data.has(scopedPath)

    if (hasStoredValue || hasMeaningfulFieldValue(storeValue)) {
      return storeValue
    }

    if (rawObject.value) {
      const rawValue = readObjectPath(rawObject.value, resolveRawRelativePath(fieldCodeOrPath))
      if (rawValue !== undefined) {
        return createAutoFieldValue(rawValue)
      }
    }

    return storeValue
  }
  
  // 可见子字段（根据条件渲染规则过滤）
  const visibleSubFields = computed(() => {
    const scopedFormManager: FormValueReader = {
      getValue: (fieldCodeOrPath: string) => getScopedFieldValue(fieldCodeOrPath),
      hasValue: (fieldCodeOrPath: string) => formDataStore.data.has(resolveScopedPath(fieldCodeOrPath)),
    }

    return subFields.value.filter((subField) =>
      getFieldPresenceState(
        subField,
        scopedFormManager,
        subFields.value,
        `${props.fieldPath}.${subField.code}`
      ).visible
    )
  })

  function isSubFieldRequired(subField: any): boolean {
    const scopedFormManager: FormValueReader = {
      getValue: (fieldCodeOrPath: string) => getScopedFieldValue(fieldCodeOrPath),
      hasValue: (fieldCodeOrPath: string) => formDataStore.data.has(resolveScopedPath(fieldCodeOrPath)),
    }

    return getFieldPresenceState(
      subField,
      scopedFormManager,
      subFields.value,
      `${props.fieldPath}.${subField.code}`
    ).required
  }
  
  // 获取子字段的值
  function getSubFieldValue(subFieldCode: string): any {
    const isReadOnlyContext =
      props.mode === 'response'
      || (props.mode === 'table-cell' && props.parentMode === 'response')

    if (isReadOnlyContext && rawObject.value) {
      const rawValue = rawObject.value[subFieldCode]
      return rawValue !== undefined ? createAutoFieldValue(rawValue) : createEmptyRawFieldValue()
    }

    return getScopedFieldValue(subFieldCode)
  }
  
  // 更新子字段的值
  function updateSubFieldValue(subFieldCode: string, value: any): void {
    const subFieldPath = `${props.fieldPath}.${subFieldCode}`
    formDataStore.setValue(subFieldPath, value)
    props.formRenderer?.clearFieldErrors?.(subFieldPath)

    const clearedFieldPaths = clearScopedDependentFields({
      formDataStore,
      fields: subFields.value,
      changedFieldCode: subFieldCode,
      scopePath: props.fieldPath,
    })

    clearedFieldPaths.forEach((fieldPath) => {
      props.formRenderer?.clearFieldErrors?.(fieldPath, { includeSubtree: true })
    })

    applyScopedPresenceEffects({
      fields: subFields.value,
      formDataStore,
      scopePath: props.fieldPath,
      clearFieldErrors: props.formRenderer?.clearFieldErrors,
    })

    syncFormContainerValue(formDataStore, props.field, props.fieldPath, props.value)
    props.formRenderer?.clearFieldErrors?.(props.fieldPath)
  }
  
  return {
    subFields,
    visibleSubFields,
    isSubFieldRequired,
    getSubFieldValue,
    updateSubFieldValue
  }
}
