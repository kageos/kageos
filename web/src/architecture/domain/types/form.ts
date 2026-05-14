import type { FieldConfig, FieldValue } from './field'

export interface ValidationResult {
  valid: boolean
  message?: string
  field?: FieldConfig
}

export interface FormState {
  data: Map<string, FieldValue>
  errors: Map<string, ValidationResult[]>
  submitting: boolean
  response?: Record<string, any> | null
  metadata?: Record<string, any> | null
}
