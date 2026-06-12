import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import RoleHandoffCard from './RoleHandoffCard.vue'

const TagStub = {
  props: ['type', 'size'],
  template: '<span class="el-tag-stub"><slot /></span>',
}

function mountCard(resultData: Record<string, unknown>, args: Record<string, unknown> = {}) {
  return mount(RoleHandoffCard, {
    props: {
      defaultCollapsed: false,
      toolCall: {
        id: 'call-change-role',
        index: 0,
        round: 0,
        name: 'change_role',
        status: 'ok',
        arguments: JSON.stringify({
          target_role: 'qa_engineer',
          execute_directory: '/system/x_world/vote',
          reset_context: true,
          ...args,
        }),
        result: '',
        result_data: {
          role_id: 'qa_engineer',
          display_name: '测试工程师',
          execute_directory: '/system/x_world/vote',
          handoff: {
            execute_directory: '/system/x_world/vote',
            task_context: ['build 已通过'],
            key_information: ['重点验证投票提交'],
            references: ['/system/prompt/roles/qa-engineer'],
          },
          runtime_contract: {
            hooks: [
              {
                id: 'qa.before_enter_schema',
                stage: 'before_enter',
                purpose: '收集目标应用函数和 schema 摘要',
                implementation_status: 'implemented',
                produces: ['verification_plan'],
              },
              {
                id: 'qa.after_run',
                stage: 'after_tool',
                purpose: '运行失败后归因',
                implementation_status: 'planned',
              },
            ],
          },
          role_definition: {
            id: 'qa_engineer',
            display_name: '测试工程师',
            responsibility: '测试工程师确认 schema 后按实际功能顺序验证 Table/Form/Chart；不直接改代码。',
            allowed_tools: ['change_role', 'read_doc', 'search', 'run_form_submit'],
            forbidden_tools: ['write_go_file', 'build_workspace'],
            runtime_contract: {
              sop: ['固定 execute_directory', '确认 schema', '调用 run_* 工具验证'],
              done_when: ['核心 Table/Form/Chart 已验证并给出结论'],
            },
          },
          ...resultData,
        },
      },
    },
    global: {
      stubs: {
        ElTag: TagStub,
      },
    },
  })
}

describe('RoleHandoffCard', () => {
  it('renders model context policy observability', () => {
    const wrapper = mountCard({
      switched: true,
      context_policy: '已切换身份；旧上下文只作背景，优先携带标准四块交接并按当前身份文档包执行。执行目录固定为 /system/x_world/vote；所有读取、构建、测试、运行都必须限定在该目录或该目录下函数。',
    })

    expect(wrapper.text()).toContain('模型上下文策略')
    expect(wrapper.text()).toContain('角色已切换')
    expect(wrapper.text()).toContain('旧细节已裁剪')
    expect(wrapper.text()).toContain('旧上下文仅作背景')
    expect(wrapper.text()).toContain('四块交接生效')
    expect(wrapper.text()).toContain('目录已固定')
    expect(wrapper.text()).toContain('执行目录固定为 /system/x_world/vote')
    expect(wrapper.text()).toContain('qa.before_enter_schema')
    expect(wrapper.text()).toContain('已实现')
    expect(wrapper.text()).toContain('qa.after_run')
    expect(wrapper.text()).toContain('计划中')
    expect(wrapper.text()).toContain('角色协议')
    expect(wrapper.text()).toContain('测试工程师确认 schema 后按实际功能顺序验证')
    expect(wrapper.text()).toContain('run_form_submit')
    expect(wrapper.text()).toContain('write_go_file')
  })

  it('prefers typed handoff packet over legacy handoff blocks', () => {
    const wrapper = mountCard({
      handoff_packet: {
        version: 'role_handoff.v1',
        target_role: 'qa_engineer',
        execute_directory: '/system/x_world/vote_packet',
        task_context: ['packet task context'],
        key_information: ['packet key information'],
        references: ['packet-reference.table'],
        context_policy: '已切换身份；旧上下文只作背景，优先携带标准四块交接。执行目录固定为 /system/x_world/vote_packet。',
        executed_hooks: [
          {
            id: 'qa.before_enter_schema',
            stage: 'before_enter',
            status: 'ok',
            produced: ['schema_snapshot'],
          },
        ],
        build_diagnostics: {
          status: 'error',
          workspace_path: '/system/x_world/vote_packet',
          error_summary: 'router /vote_packet/list.table schema failed',
          categories: ['schema_validation'],
          routers: ['/vote_packet/list.table'],
          required_docs: ['/system/prompt/sdk/reference/build-validation'],
          repair_policy: ['先读构建校验文档'],
        },
        validation: {
          status: 'warning',
          warnings: ['references 包含无效或泛化引用'],
          repaired: ['已移除无效 references 项：当前目录'],
        },
      },
    })

    expect(wrapper.text()).toContain('交接协议 role_handoff.v1')
    expect(wrapper.text()).toContain('交接协议校验')
    expect(wrapper.text()).toContain('字段已修正')
    expect(wrapper.text()).toContain('警告：references 包含无效或泛化引用')
    expect(wrapper.text()).toContain('已修正：已移除无效 references 项：当前目录')
    expect(wrapper.text()).toContain('/system/x_world/vote_packet')
    expect(wrapper.text()).toContain('packet task context')
    expect(wrapper.text()).toContain('packet key information')
    expect(wrapper.text()).toContain('packet-reference.table')
    expect(wrapper.text()).toContain('schema_snapshot')
    expect(wrapper.text()).toContain('router /vote_packet/list.table schema failed')
    expect(wrapper.text()).not.toContain('重点验证投票提交')
  })
})
