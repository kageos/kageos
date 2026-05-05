import { describe, expect, it } from 'vitest'
import { findMiniComposerMentionQuery, replaceMiniComposerMention } from './miniComposerMention'

describe('miniComposerMention', () => {
  it('detects a user mention token at the cursor', () => {
    const text = '请 @ali 帮忙'
    const query = findMiniComposerMentionQuery(text, 6)

    expect(query).toEqual({
      kind: 'user',
      trigger: '@',
      query: 'ali',
      start: 2,
      end: 6
    })
  })

  it('detects a resource token with nested slashes', () => {
    const text = '查看 /demo/docs/guide'
    const query = findMiniComposerMentionQuery(text, text.length)

    expect(query?.kind).toBe('resource')
    expect(query?.query).toBe('demo/docs/guide')
  })

  it('does not detect trigger characters inside a normal token', () => {
    expect(findMiniComposerMentionQuery('邮箱 a@b.com', 9)).toBeNull()
  })

  it('replaces the active token and appends a spacer at the end', () => {
    const text = '请看 /doc'
    const query = findMiniComposerMentionQuery(text, text.length)

    expect(query).not.toBeNull()
    const result = replaceMiniComposerMention(text, query!, '/luobei/demo/help.docs')

    expect(result.value).toBe('请看 /luobei/demo/help.docs ')
    expect(result.cursor).toBe(result.value.length)
  })

  it('does not add an extra spacer before existing whitespace', () => {
    const text = '请 @ali 帮忙'
    const query = findMiniComposerMentionQuery(text, 6)

    expect(query).not.toBeNull()
    const result = replaceMiniComposerMention(text, query!, '@alice')

    expect(result.value).toBe('请 @alice 帮忙')
    expect(result.cursor).toBe(8)
  })
})
