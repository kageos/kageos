import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFormDataStore } from './formData'
import { FormStateManager } from '@/architecture/infrastructure/stateManager/FormStateManager'

describe('formData store scoping', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('keeps stores isolated across different pinia instances', () => {
    const piniaA = createPinia()
    const piniaB = createPinia()

    const storeA = useFormDataStore(piniaA)
    const storeB = useFormDataStore(piniaB)

    storeA.setValue('name', { raw: 'Alice', display: 'Alice', meta: {} } as any)

    expect(storeA.getValue('name').raw).toBe('Alice')
    expect(storeB.getValue('name').raw).toBeNull()
  })

  it('keeps state managers isolated when bound to different stores', () => {
    const storeA = useFormDataStore(createPinia())
    const storeB = useFormDataStore(createPinia())

    const managerA = new FormStateManager(storeA)
    const managerB = new FormStateManager(storeB)

    managerA.setValue('profile.name', { raw: 'Alice', display: 'Alice', meta: {} } as any)
    managerB.setValue('profile.name', { raw: 'Bob', display: 'Bob', meta: {} } as any)

    expect(managerA.getValue('profile.name').raw).toBe('Alice')
    expect(managerB.getValue('profile.name').raw).toBe('Bob')
  })
})
