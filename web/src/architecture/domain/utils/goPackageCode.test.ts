import { describe, expect, it } from 'vitest'
import { buildUniqueGoPackageCode, createGoPackageCodeFromLabel } from './goPackageCode'

describe('goPackageCode', () => {
  it('creates pinyin codes from Chinese labels', () => {
    expect(createGoPackageCodeFromLabel('用户管理')).toBe('yong_hu_guan_li')
  })

  it('keeps ascii words and numbers', () => {
    expect(createGoPackageCodeFromLabel('CRM客户2.0')).toBe('crm_ke_hu_2_0')
  })

  it('adds a fallback prefix when the label starts with a number', () => {
    expect(createGoPackageCodeFromLabel('2026')).toBe('directory_2026')
  })

  it('adds numeric suffixes for existing sibling codes', () => {
    expect(buildUniqueGoPackageCode('用户管理', ['yong_hu_guan_li', 'yong_hu_guan_li_2'])).toBe('yong_hu_guan_li_3')
  })
})
