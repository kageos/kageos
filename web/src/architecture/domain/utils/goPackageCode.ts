import { pinyin } from 'pinyin-pro'
import { GO_PACKAGE_NAME_MAX_LENGTH } from './goPackageName'

const RESERVED_GO_PACKAGE_CODES = new Set([
  'break',
  'case',
  'chan',
  'const',
  'continue',
  'default',
  'defer',
  'else',
  'fallthrough',
  'for',
  'func',
  'go',
  'goto',
  'if',
  'import',
  'interface',
  'map',
  'package',
  'range',
  'return',
  'select',
  'struct',
  'switch',
  'type',
  'var'
])

export function createGoPackageCodeFromLabel(label: string, fallback = 'directory'): string {
  const fallbackCode = normalizeCodeCandidate(fallback) || 'directory'
  let code = normalizeCodeCandidate(tokenizeLabel(label).join('_'))

  if (!code) {
    code = fallbackCode
  }
  if (!/^[a-z]/.test(code)) {
    code = `${fallbackCode}_${code}`
  }
  if (RESERVED_GO_PACKAGE_CODES.has(code)) {
    code = `${fallbackCode}_${code}`
  }

  code = trimCode(code, GO_PACKAGE_NAME_MAX_LENGTH)
  if (!code || !/^[a-z]/.test(code)) {
    return fallbackCode
  }
  if (RESERVED_GO_PACKAGE_CODES.has(code)) {
    return trimCode(`${fallbackCode}_${code}`, GO_PACKAGE_NAME_MAX_LENGTH)
  }
  return code
}

export function buildUniqueGoPackageCode(baseCode: string, existingCodes: Iterable<string>): string {
  const used = new Set(Array.from(existingCodes, code => code.toLowerCase()).filter(Boolean))
  let candidate = createGoPackageCodeFromLabel(baseCode)
  let index = 2

  while (used.has(candidate.toLowerCase())) {
    candidate = withNumericSuffix(baseCode, index)
    index += 1
  }

  return candidate
}

function tokenizeLabel(label: string): string[] {
  const tokens: string[] = []
  let ascii = ''

  const flushASCII = () => {
    if (!ascii) return
    tokens.push(ascii)
    ascii = ''
  }

  for (const char of label.normalize('NFKD')) {
    if (/[\u0300-\u036f]/.test(char)) {
      continue
    }
    if (/^[a-z0-9]$/i.test(char)) {
      ascii += char.toLowerCase()
      continue
    }

    flushASCII()
    const converted = pinyin(char, { toneType: 'none', type: 'array' })
      .map(part => normalizeCodeCandidate(part))
      .filter(Boolean)
    tokens.push(...converted)
  }
  flushASCII()

  return tokens
}

function withNumericSuffix(baseCode: string, index: number): string {
  const base = createGoPackageCodeFromLabel(baseCode)
  if (index <= 1) {
    return base
  }

  const suffix = `_${index}`
  const maxBaseLength = Math.max(1, GO_PACKAGE_NAME_MAX_LENGTH - suffix.length)
  return `${trimCode(base, maxBaseLength)}${suffix}`
}

function normalizeCodeCandidate(value: string): string {
  return value
    .normalize('NFKD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/_+/g, '_')
    .replace(/^_+|_+$/g, '')
}

function trimCode(code: string, maxLength: number): string {
  return code.slice(0, maxLength).replace(/^_+|_+$/g, '')
}
