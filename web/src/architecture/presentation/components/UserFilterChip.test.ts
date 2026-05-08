import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import UserFilterChip from './UserFilterChip.vue'

describe('UserFilterChip', () => {
  it('renders the label and emits remove when close is clicked', async () => {
    const wrapper = mount(UserFilterChip, {
      props: {
        label: 'Alice',
        initial: 'A'
      }
    })

    expect(wrapper.text()).toContain('Alice')

    await wrapper.get('.user-chip-close').trigger('click')

    expect(wrapper.emitted('remove')).toEqual([[]])
  })
})
