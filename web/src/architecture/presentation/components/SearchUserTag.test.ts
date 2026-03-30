import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SearchUserTag from './SearchUserTag.vue'

describe('SearchUserTag', () => {
  it('renders the label and emits remove when close is clicked', async () => {
    const wrapper = mount(SearchUserTag, {
      props: {
        label: 'Alice',
        initial: 'A'
      }
    })

    expect(wrapper.text()).toContain('Alice')

    await wrapper.get('.user-tag-close').trigger('click')

    expect(wrapper.emitted('remove')).toEqual([[]])
  })
})
