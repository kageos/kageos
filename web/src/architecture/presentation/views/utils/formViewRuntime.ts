import { createPinia } from 'pinia'
import { FormApplicationService } from '@/architecture/application/services/FormApplicationService'
import type { IEventBus } from '@/architecture/domain/interfaces/IEventBus'
import type { IFormGateway } from '@/architecture/domain/interfaces/IFormGateway'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { FormDomainService } from '@/architecture/domain/services/FormDomainService'
import { FormStateManager } from '@/architecture/infrastructure/stateManager/FormStateManager'
import { useFormDataStore, type FormDataStore } from '@/architecture/runtime/stores/formData'
import { useResponseDataStore } from '@/architecture/runtime/stores/responseData'
import { createEmptyFieldValue } from '@/architecture/runtime/utils/createFieldValue'
import { useAuthStore } from '@/architecture/infrastructure/stores/auth'

export function createFormViewRuntime(options: {
  eventBus: IEventBus
  formGateway: IFormGateway
}) {
  const scopedFormPinia = createPinia()
  const formDataStore = useFormDataStore(scopedFormPinia)
  const responseDataStore = useResponseDataStore(scopedFormPinia)
  const stateManager = new FormStateManager(formDataStore)
  const domainService = new FormDomainService(stateManager, options.eventBus, [], {
    getAuthStore: () => useAuthStore()
  })
  const applicationService = new FormApplicationService(domainService, options.eventBus, options.formGateway)

  return {
    scopedFormPinia,
    formDataStore,
    responseDataStore,
    stateManager,
    domainService,
    applicationService
  }
}

export function syncFormDataStoreToStateManager(options: {
  fields: FieldConfig[]
  formDataStore: Pick<FormDataStore, 'getValue'>
  stateManager: FormStateManager
}): void {
  const state = options.stateManager.getState()
  const newData = new Map<string, FieldValue>()

  options.fields.forEach((field: FieldConfig) => {
    const fieldValue = options.formDataStore.getValue(field.code)
    if (fieldValue) {
      newData.set(field.code, fieldValue)
    } else {
      newData.set(field.code, createEmptyFieldValue(field))
    }
  })

  options.stateManager.setState({
    ...state,
    data: newData
  })
}

export function buildInitialDataFromFormDataStore(options: {
  fields: FieldConfig[]
  formDataStore: Pick<FormDataStore, 'getValue'>
}): Record<string, any> {
  const initialData: Record<string, any> = {}

  options.fields.forEach((field: FieldConfig) => {
    const fieldValue = options.formDataStore.getValue(field.code)
    if (fieldValue) {
      initialData[field.code] = fieldValue.raw
    }
  })

  return initialData
}
