import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import enUS from './en-US'
import zhCN from './zh-CN'

function messagePaths(value: unknown, prefix = ''): string[] {
  if (typeof value === 'string') {
    return [prefix]
  }
  if (!value || typeof value !== 'object') {
    return []
  }
  return Object.entries(value).flatMap(([key, child]) =>
    messagePaths(child, prefix ? `${prefix}.${key}` : key)
  )
}

describe('built-in locale message syntax', () => {
  it.each([
    ['zh-CN', zhCN],
    ['en-US', enUS],
  ] as const)('compiles every %s message', (locale, messages) => {
    const i18n = createI18n({
      legacy: false,
      locale,
      messages: { [locale]: messages },
    })

    for (const path of messagePaths(messages)) {
      expect(() => i18n.global.t(path), path).not.toThrow()
    }
  })

  it('renders the LLM JSON example with literal braces', () => {
    const i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      messages: { 'zh-CN': zhCN },
    })

    expect(i18n.global.t('llmManagement.extraConfigPlaceholder')).toContain(
      '{"reasoning_effort":"medium"'
    )
  })
})
