import { describe, expect, it } from 'vitest'
import {
  buildWorkspaceInvocationSnippet,
  filterEmptyInvocationParams,
  parseWorkspaceInvocationBlocks,
  parseWorkspacePromptSegments,
  unwrapWorkspaceResourceToken,
  wrapWorkspaceResourcePath,
} from './workspaceInvocationSnippet'

describe('workspaceInvocationSnippet', () => {
  it('wraps and unwraps resource paths with the lightweight token syntax', () => {
    const token = wrapWorkspaceResourcePath('/system/demos/weixin/wechat_articles/search_articles.form')

    expect(token).toBe('</system/demos/weixin/wechat_articles/search_articles.form>')
    expect(unwrapWorkspaceResourceToken(token)).toBe('/system/demos/weixin/wechat_articles/search_articles.form')
  })

  it('splits prompt text into text and resource segments', () => {
    const segments = parseWorkspacePromptSegments('调用 </system/app/search.form> 后总结')

    expect(segments.map((segment) => segment.type)).toEqual(['text', 'resource', 'text'])
    expect(segments[1]).toMatchObject({
      path: '/system/app/search.form',
      text: '</system/app/search.form>',
    })
  })

  it('builds a workspace-readable invocation block', () => {
    const snippet = buildWorkspaceInvocationSnippet({
      tool: 'run_form_submit',
      resourcePath: '/system/app/search.form',
      params: {
        keyword: 'AI 最新热点',
        page_size: 5,
      },
    })

    expect(snippet).toContain('函数调用：')
    expect(snippet).toContain('用途：复制后粘贴到工作台')
    expect(snippet).toContain('工具：run_form_submit')
    expect(snippet).toContain('函数：</system/app/search.form>')
    expect(snippet).toContain('keyword = AI 最新热点')
    expect(snippet).toContain('page_size = 5')
  })

  it('parses invocation blocks back into a light structure for preview', () => {
    const snippet = buildWorkspaceInvocationSnippet({
      tool: 'run_table_update',
      resourcePath: '/system/app/orders.table',
      params: [
        { key: 'body', value: [{ id: 1, updates: { status: '已处理' } }] },
        { key: 'groupid', value: 1, fixed: true },
      ],
    })

    const blocks = parseWorkspaceInvocationBlocks(snippet)

    expect(blocks).toHaveLength(1)
    expect(blocks[0]).toMatchObject({
      tool: 'run_table_update',
      resourcePath: '/system/app/orders.table',
    })
    expect(blocks[0]?.params.map((param) => ({ key: param.key, fixed: param.fixed }))).toEqual([
      { key: 'body', fixed: false },
      { key: 'groupid', fixed: true },
    ])
  })

  it('filters empty params before copying partially filled forms', () => {
    expect(filterEmptyInvocationParams({
      keyword: 'AI',
      empty: '',
      none: null,
      list: [],
      config_id: 0,
    })).toEqual({
      keyword: 'AI',
      config_id: 0,
    })
  })
})
