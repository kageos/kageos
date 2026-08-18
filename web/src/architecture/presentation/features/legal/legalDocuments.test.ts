import { describe, expect, it } from 'vitest'
import { CURRENT_LEGAL_POLICY_VERSION, getLegalDocument } from './legalDocuments'

describe('legal documents', () => {
  it('publishes the current policy version', () => {
    expect(CURRENT_LEGAL_POLICY_VERSION).toBe('2026-08-18')
  })

  it('covers the required privacy disclosure topics in Chinese', () => {
    const privacy = getLegalDocument('privacy', 'zh-CN')
    const text = privacy.sections.flatMap(section => [section.title, ...section.paragraphs, ...(section.bullets || [])]).join('\n')

    for (const topic of ['个人信息处理者', '保存期限', '第三方', '跨境', '注销账号', '未成年人', '撤回同意']) {
      expect(text).toContain(topic)
    }
  })

  it('keeps terms and privacy available in both supported languages', () => {
    for (const locale of ['zh-CN', 'en-US']) {
      expect(getLegalDocument('terms', locale).sections.length).toBeGreaterThanOrEqual(5)
      expect(getLegalDocument('privacy', locale).sections.length).toBeGreaterThanOrEqual(5)
    }
  })

  it('uses the lowercase kageos brand spelling', () => {
    const documents = ['zh-CN', 'en-US'].flatMap(locale => [
      getLegalDocument('terms', locale),
      getLegalDocument('privacy', locale),
    ])

    expect(JSON.stringify(documents)).not.toContain('KageOS')
    expect(JSON.stringify(documents)).toContain('kageos')
  })
})
