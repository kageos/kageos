import { describe, expect, it } from 'vitest'
import { SUPPORTED_LOCALES, type SupportedLocale } from './index'
import enUS from './locales/en-US'
import zhCN from './locales/zh-CN'
import { additionalLocaleMessages } from './locales/additional'

function collectLeafKeys(value: unknown, prefix = ''): string[] {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return prefix ? [prefix] : []
  }

  return Object.entries(value)
    .flatMap(([key, child]) => collectLeafKeys(child, prefix ? `${prefix}.${key}` : key))
    .sort()
}

function collectUnknownOverrideKeys(
  overrides: Record<string, unknown>,
  base: Record<string, unknown>,
  prefix = ''
): string[] {
  const unknownKeys: string[] = []

  for (const [key, value] of Object.entries(overrides)) {
    const path = prefix ? `${prefix}.${key}` : key
    const baseValue = base[key]
    if (baseValue === undefined) {
      unknownKeys.push(path)
      continue
    }

    if (
      value &&
      typeof value === 'object' &&
      !Array.isArray(value) &&
      baseValue &&
      typeof baseValue === 'object' &&
      !Array.isArray(baseValue)
    ) {
      unknownKeys.push(
        ...collectUnknownOverrideKeys(
          value as Record<string, unknown>,
          baseValue as Record<string, unknown>,
          path
        )
      )
    }
  }

  return unknownKeys.sort()
}

describe('i18n message catalog', () => {
  it('keeps English and Chinese base locale keys aligned', () => {
    expect(collectLeafKeys(zhCN)).toEqual(collectLeafKeys(enUS))
  })

  it('only declares supported additional locales', () => {
    const supported = new Set(SUPPORTED_LOCALES)
    expect(Object.keys(additionalLocaleMessages).filter(locale => !supported.has(locale as SupportedLocale))).toEqual([])
  })

  it('has an override bucket for every non-base locale', () => {
    const baseLocales: SupportedLocale[] = ['en-US', 'zh-CN']
    const expectedAdditionalLocales = SUPPORTED_LOCALES.filter(locale => !baseLocales.includes(locale)).sort()
    expect(Object.keys(additionalLocaleMessages).sort()).toEqual(expectedAdditionalLocales)
  })

  it('keeps additional locale overrides within the base message shape', () => {
    const unknownKeys = Object.entries(additionalLocaleMessages)
      .flatMap(([locale, overrides]) => collectUnknownOverrideKeys(
        overrides as Record<string, unknown>,
        enUS as Record<string, unknown>
      ).map(key => `${locale}:${key}`))

    expect(unknownKeys).toEqual([])
  })
})
