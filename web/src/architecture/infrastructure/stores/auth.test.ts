import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from './auth'

describe('auth store persistence', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('recovers from corrupt persisted user data', () => {
    localStorage.setItem('token', 'token-1')
    localStorage.setItem('user', '{invalid json')

    const store = useAuthStore()

    expect(store.token).toBe('token-1')
    expect(store.user).toBeNull()
    expect(localStorage.getItem('user')).toBeNull()
  })

  it('ignores persisted user values that do not match the user shape', () => {
    localStorage.setItem('user', JSON.stringify({ unexpected: true }))

    const store = useAuthStore()

    expect(store.user).toBeNull()
    expect(localStorage.getItem('user')).toBeNull()
  })
})
