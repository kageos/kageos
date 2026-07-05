import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import MiniWorkstationMessages from './MiniWorkstationMessages.vue'

vi.mock('@/architecture/presentation/context/appStoresContext', () => ({
  useAuthStore: () => ({
    user: { username: 'demo' },
    userName: 'demo',
  }),
}))

const UserDisplayStub = defineComponent({
  template: '<span class="user-display-stub" />',
})

function renderMarkdownForTest(text: string): string {
  return text
    .replace(/^# (.+)$/gm, '<h1>$1</h1>')
    .replace(/\n/g, '<br>')
}

function mountMessages(messages: any[] = [
  {
    role: 'user',
    user: 'demo',
    content: '# 处理订单\n请看 <./orders.table>',
    created_at: '2026-07-05T10:00:00Z',
  },
]) {
  return mount(MiniWorkstationMessages, {
    props: {
      messages,
      maximized: false,
      sending: false,
      fullCodePath: '/system/app',
      streamingDisplayLength: 0,
      renderMarkdown: renderMarkdownForTest,
      formatMessageTime: (value: string) => value,
      getFileGroupsFromCalls: () => [],
      getDisplayFieldsFromCalls: () => [],
    },
    global: {
      stubs: {
        UserDisplay: UserDisplayStub,
        MiniWorkstationResourceIdentity: true,
        MessageToolCalls: defineComponent({
          props: {
            toolCalls: { type: Array, default: () => [] },
          },
          template: '<div class="message-tool-calls-stub">tools {{ toolCalls.length }}</div>',
        }),
      },
    },
  })
}

describe('MiniWorkstationMessages', () => {
  it('renders user message markdown and workspace resource tokens', () => {
    const wrapper = mountMessages()

    expect(wrapper.find('.mini-msg-user-body h1').text()).toBe('处理订单')

    const token = wrapper.get('a.workspace-resource-token')
    expect(token.text()).toContain('orders.table')
    expect(token.attributes('data-full-code-path')).toBe('/system/app/orders.table')
    expect(token.attributes('href')).toBe('/workspace/system/app/orders.table')
  })

  it('labels collapsed assistant activity by tool count instead of display rounds', () => {
    const wrapper = mountMessages([
      {
        role: 'user',
        user: 'demo',
        content: '看一下系统',
        created_at: '2026-07-05T10:00:00Z',
      },
      {
        role: 'assistant',
        user: 'demo',
        content: '系统概览',
        created_at: '2026-07-05T10:00:01Z',
        model_context_plan: {
          protocol_version: 'workspace_model_context.v1',
          session_id: 'session-1',
          round: 2,
          role: { id: 'reviewer' },
          execution: { full_code_path: '/system/app', children_count: 0, files_count: 0, scope_policy: 'directory' },
          messages: { context_policy: 'include', source_history_policy: 'full', system_messages: 1, llm_messages: 2, total_stored_messages: 2, included_stored_messages: 2, excluded_stored_messages: 0, excluded_by_anchor: 0, excluded_display_only: 0 },
          docs: {},
          tools: {},
          cache_plan: {},
        },
        blocks: [
          { type: 'tool_calls', calls: [{ name: 'change_role', status: 'ok' }] },
          { type: 'tool_calls', calls: [{ name: 'read_dir', status: 'ok' }] },
          { type: 'content', text: '系统概览' },
        ],
      },
    ])

    expect(wrapper.text()).toContain('执行过程 2 个工具')
    expect(wrapper.text()).toContain('模型第 3 轮')
    expect(wrapper.text()).toContain('工具 2 个 · change_role · read_dir')
  })
})
