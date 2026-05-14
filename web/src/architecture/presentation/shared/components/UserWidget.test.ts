import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it } from 'vitest'
import UserWidget from './UserWidget.vue'
import { WidgetType } from '@/architecture/domain/constants/widget'

const ElAvatarStub = defineComponent({
  name: 'ElAvatar',
  template: '<div class="el-avatar-stub"><slot /></div>'
})

const UserPickerDialogStub = defineComponent({
  name: 'UserPickerDialog',
  template: '<div data-testid="user-picker-dialog" />'
})

describe('UserWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('shows highlighted selected tag and clears value in search mode', async () => {
    const wrapper = mount(UserWidget, {
      props: {
        field: {
          code: 'owner',
          name: '负责人',
          widget: {
            type: WidgetType.USER,
            config: {}
          }
        } as any,
        value: {
          raw: 'alice',
          display: 'alice(艾丽丝)',
          meta: {
            userInfo: {
              username: 'alice',
              nickname: '艾丽丝',
              avatar: ''
            }
          }
        },
        mode: 'search',
        fieldPath: 'owner'
      },
      global: {
        stubs: {
          ElAvatar: ElAvatarStub,
          ElButton: true,
          ElIcon: true,
          UserPickerDialog: UserPickerDialogStub,
          UserDisplay: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.find('.search-selected-value').exists()).toBe(true)
    expect(wrapper.text()).toContain('alice(艾丽丝)')

    await wrapper.get('.selected-value-remove').trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted?.at(-1)?.[0]).toMatchObject({
      raw: null,
      display: ''
    })
  })
})
