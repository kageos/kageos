import { computed, defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import TableRowDetailDrawer from './TableRowDetailDrawer.vue'

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRouter: () => ({
      push: vi.fn()
    })
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: {
      username: 'tester'
    }
  })
}))

vi.mock('@/architecture/presentation/composables/useTableRowDetailTabs', () => ({
  useTableRowDetailTabs: () => ({
    activeTab: ref('detail'),
    showPermissionRequestTab: computed(() => false)
  })
}))

vi.mock('@/architecture/presentation/composables/useTableRowDetailLayout', () => ({
  useTableRowDetailLayout: () => ({
    RICH_TEXT_PREVIEW_HEIGHT: 240,
    useGroupedDetailLayout: ref(false),
    toggleDetailLayout: vi.fn(),
    linkFields: computed(() => []),
    groupedFields: computed(() => []),
    setRichTextContentRef: vi.fn(),
    isRichTextExpanded: vi.fn(() => false),
    isRichTextOverflow: vi.fn(() => false),
    toggleRichTextExpanded: vi.fn(),
    getFieldValue: vi.fn(() => ({
      raw: '',
      display: '',
      meta: {}
    }))
  })
}))

const DrawerStub = defineComponent({
  name: 'ElDrawer',
  props: {
    modelValue: {
      type: Boolean,
      default: false
    }
  },
  emits: ['update:modelValue', 'close'],
  setup(_props, { slots }) {
    return () =>
      h('div', { class: 'drawer-stub' }, [
        h('div', { class: 'drawer-header-slot' }, slots.header?.()),
        h('div', { class: 'drawer-default-slot' }, slots.default?.()),
        h('div', { class: 'drawer-footer-slot' }, slots.footer?.())
      ])
  }
})

const ButtonStub = defineComponent({
  name: 'ElButton',
  props: {
    disabled: {
      type: Boolean,
      default: false
    }
  },
  emits: ['click'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          disabled: props.disabled,
          onClick: () => emit('click')
        },
        slots.default?.()
      )
  }
})

const IconStub = defineComponent({
  name: 'ElIcon',
  setup(_props, { slots }) {
    return () => h('span', { class: 'icon-stub' }, slots.default?.())
  }
})

function mountDrawer(props: Record<string, unknown> = {}) {
  return mount(TableRowDetailDrawer, {
    props: {
      visible: true,
      title: '详情',
      mode: 'read',
      fields: [],
      rowData: { id: 1, name: 'Alice' },
      tableData: [{ id: 1, name: 'Alice' }],
      currentIndex: 0,
      currentFunctionDetail: {
        id: 1,
        router: '/workspace/demo/users',
        template_type: 'table',
        callbacks: []
      },
      ...props
    },
    global: {
      stubs: {
        ElDrawer: DrawerStub,
        ElButton: ButtonStub,
        ElIcon: IconStub,
        ElSkeleton: true,
        ElEmpty: true,
        TableRowDetailReadTabs: true,
        ScheduledTaskDialog: true,
        FormView: true
      }
    }
  })
}

describe('TableRowDetailDrawer', () => {
  it('shows unsupported update hint instead of edit permission button', () => {
    const wrapper = mountDrawer({
      supportsEdit: false,
      canEdit: false
    })

    expect(wrapper.text()).toContain('当前表格不支持更新')
    expect(wrapper.text()).not.toContain('编辑（需update权限）')
  })

  it('shows edit permission button only when update is supported', () => {
    const wrapper = mountDrawer({
      supportsEdit: true,
      canEdit: false
    })

    expect(wrapper.text()).toContain('编辑（需update权限）')
    expect(wrapper.text()).not.toContain('当前表格不支持更新')
  })
})
