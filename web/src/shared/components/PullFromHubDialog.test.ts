import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'

import PullFromHubDialog from './PullFromHubDialog.vue'

vi.mock('@/api/hub', () => ({
  pullDirectoryFromHub: vi.fn(),
  importHubDirectoryBundle: vi.fn(),
}))

vi.mock('@/utils/hubBundle', () => ({
  parseHubDirectoryBundleJson: vi.fn(),
}))

const ElDialogStub = defineComponent({
  name: 'ElDialog',
  props: {
    modelValue: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    return () =>
      props.modelValue
        ? h('div', { 'data-testid': 'dialog-shell' }, slots.default?.())
        : null
  },
})

const ElFormStub = defineComponent({
  name: 'ElForm',
  setup(_, { attrs, slots }) {
    return () => h('form', attrs, slots.default?.())
  },
})

const ElFormItemStub = defineComponent({
  name: 'ElFormItem',
  setup(_, { attrs, slots }) {
    return () => h('div', attrs, slots.default?.())
  },
})

const ElInputStub = defineComponent({
  name: 'ElInput',
  props: {
    modelValue: {
      type: [String, Number, null],
      default: '',
    },
  },
  emits: ['update:modelValue', 'paste'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h('div', { class: 'el-input-stub' }, [
        slots.prepend?.(),
        h('input', {
          ...attrs,
          value: props.modelValue ?? '',
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
          onPaste: (event: ClipboardEvent) => emit('paste', event),
        }),
      ])
  },
})

const ElButtonStub = defineComponent({
  name: 'ElButton',
  setup(_, { attrs, slots }) {
    return () => h('button', attrs, slots.default?.())
  },
})

const ElRadioGroupStub = defineComponent({
  name: 'ElRadioGroup',
  setup(_, { attrs, slots }) {
    return () => h('div', attrs, slots.default?.())
  },
})

const ElRadioButtonStub = defineComponent({
  name: 'ElRadioButton',
  setup(_, { attrs, slots }) {
    return () => h('button', attrs, slots.default?.())
  },
})

const ElTextStub = defineComponent({
  name: 'ElText',
  setup(_, { attrs, slots }) {
    return () => h('span', attrs, slots.default?.())
  },
})

const ElAlertStub = defineComponent({
  name: 'ElAlert',
  setup(_, { attrs, slots }) {
    return () => h('div', attrs, slots.default?.())
  },
})

const ElUploadStub = defineComponent({
  name: 'ElUpload',
  setup(_, { attrs, slots }) {
    return () => h('div', attrs, slots.default?.())
  },
})

const ElIconStub = defineComponent({
  name: 'ElIcon',
  setup(_, { slots }) {
    return () => h('span', slots.default?.())
  },
})

describe('PullFromHubDialog', () => {
  it('hydrates initial hub link when mounted already visible', async () => {
    const wrapper = mount(PullFromHubDialog, {
      props: {
        modelValue: true,
        initialHubLink: 'hub://demo.example/workspace/sample_dir@1',
        currentApp: {
          id: 1,
          name: 'Demo',
        },
      },
      global: {
        stubs: {
          ElDialog: ElDialogStub,
          ElForm: ElFormStub,
          ElFormItem: ElFormItemStub,
          ElInput: ElInputStub,
          ElButton: ElButtonStub,
          ElRadioGroup: ElRadioGroupStub,
          ElRadioButton: ElRadioButtonStub,
          ElText: ElTextStub,
          ElAlert: ElAlertStub,
          ElUpload: ElUploadStub,
          ElIcon: ElIconStub,
          Link: true,
        },
      },
    })

    await nextTick()

    const input = wrapper.get('input[data-testid="pull-from-hub-link"]')
    expect((input.element as HTMLInputElement).value).toBe('hub://demo.example/workspace/sample_dir@1')
  })
})
