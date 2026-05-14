import type { FieldConfig, FieldValue, FormState } from '@/architecture/domain/types'
import type { IStateManager } from './IStateManager'

export interface FormValueStorePort {
  getValue(fieldPath: string): FieldValue
  hasValue(fieldPath: string): boolean
  setValue(fieldPath: string, value: FieldValue): void
  deleteValue(fieldPath: string): void
  getAllFieldPaths(): string[]
}

export interface IFormStateManager extends IStateManager<FormState>, FormValueStorePort {
  getDataSnapshot(): Map<string, FieldValue>
  getSubmitData(fields: FieldConfig[]): Record<string, any>
  setSubmitting(submitting: boolean): void
  setResponse(response: Record<string, any> | null): void
  setMetadata(metadata: Record<string, any> | null): void
}

export function isFormValueStorePort(value: unknown): value is FormValueStorePort {
  const candidate = value as Partial<FormValueStorePort> | null
  return !!candidate
    && typeof candidate.getValue === 'function'
    && typeof candidate.hasValue === 'function'
    && typeof candidate.setValue === 'function'
    && typeof candidate.deleteValue === 'function'
    && typeof candidate.getAllFieldPaths === 'function'
}

export function isFormStateManager(value: unknown): value is IFormStateManager {
  const candidate = value as Partial<IFormStateManager> | null
  return isFormValueStorePort(value)
    && typeof candidate?.getDataSnapshot === 'function'
    && typeof candidate.getSubmitData === 'function'
    && typeof candidate.setSubmitting === 'function'
    && typeof candidate.setResponse === 'function'
    && typeof candidate.setMetadata === 'function'
}
