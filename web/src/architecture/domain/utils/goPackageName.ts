export const GO_PACKAGE_NAME_MAX_LENGTH = 50

const goPackageRegex = /^[a-z][a-z0-9_]*$/

const goKeywords = new Set([
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

export interface GoPackageNameValidationOptions {
  minLength?: number
  maxLength?: number
}

export const normalizeGoPackageName = (code: string): string => code.trim()

export const validateGoPackageName = (
  code: string,
  label = '英文标识',
  options: GoPackageNameValidationOptions = {}
): string | null => {
  const minLength = options.minLength ?? 1
  const maxLength = options.maxLength ?? GO_PACKAGE_NAME_MAX_LENGTH

  if (!code) {
    return `${label}不能为空`
  }
  if (code.length < minLength || code.length > maxLength) {
    return `${label}长度须为 ${minLength}-${maxLength} 个字符`
  }
  if (!goPackageRegex.test(code)) {
    return `${label}需以小写英文字母开头，只能包含小写英文字母、数字和下划线，不能使用横线`
  }
  if (goKeywords.has(code)) {
    return `${label}「${code}」已被系统占用，请换一个`
  }
  return null
}
