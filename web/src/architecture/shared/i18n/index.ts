import { createI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import enUS from './locales/en-US'
import zhCN from './locales/zh-CN'
import type { LocaleMessageOverrides } from './locales/additional'

type LocaleChangedHandler = (locale: SupportedLocale) => void

export const SUPPORTED_LOCALES = [
  'en-US',
  'zh-CN',
  'ja-JP',
  'ko-KR',
  'es-ES',
  'ar-SA',
  'ru-RU',
  'fr-FR',
  'de-DE',
  'pt-BR',
  'it-IT',
  'id-ID',
  'tr-TR',
  'hi-IN',
  'nl-NL',
  'pl-PL',
  'vi-VN',
  'th-TH',
] as const
export type SupportedLocale = typeof SUPPORTED_LOCALES[number]

export const DEFAULT_LOCALE: SupportedLocale = 'en-US'
export const LOCALE_STORAGE_KEY = 'kageos-locale'

export interface LocaleMeta {
  label: string
  nativeLabel: string
  flag: string
  dayjsLocale: string
  dir?: 'ltr' | 'rtl'
}

export const LOCALE_META: Record<SupportedLocale, LocaleMeta> = {
  'en-US': { label: 'English', nativeLabel: 'English', flag: '🇺🇸', dayjsLocale: 'en' },
  'zh-CN': { label: 'Chinese', nativeLabel: '简体中文', flag: '🇨🇳', dayjsLocale: 'zh-cn' },
  'ja-JP': { label: 'Japanese', nativeLabel: '日本語', flag: '🇯🇵', dayjsLocale: 'ja' },
  'ko-KR': { label: 'Korean', nativeLabel: '한국어', flag: '🇰🇷', dayjsLocale: 'ko' },
  'es-ES': { label: 'Spanish', nativeLabel: 'Español', flag: '🇪🇸', dayjsLocale: 'es' },
  'ar-SA': { label: 'Arabic', nativeLabel: 'العربية', flag: '🇸🇦', dayjsLocale: 'ar', dir: 'rtl' },
  'ru-RU': { label: 'Russian', nativeLabel: 'Русский', flag: '🇷🇺', dayjsLocale: 'ru' },
  'fr-FR': { label: 'French', nativeLabel: 'Français', flag: '🇫🇷', dayjsLocale: 'fr' },
  'de-DE': { label: 'German', nativeLabel: 'Deutsch', flag: '🇩🇪', dayjsLocale: 'de' },
  'pt-BR': { label: 'Portuguese', nativeLabel: 'Português', flag: '🇧🇷', dayjsLocale: 'pt-br' },
  'it-IT': { label: 'Italian', nativeLabel: 'Italiano', flag: '🇮🇹', dayjsLocale: 'it' },
  'id-ID': { label: 'Indonesian', nativeLabel: 'Bahasa Indonesia', flag: '🇮🇩', dayjsLocale: 'id' },
  'tr-TR': { label: 'Turkish', nativeLabel: 'Türkçe', flag: '🇹🇷', dayjsLocale: 'tr' },
  'hi-IN': { label: 'Hindi', nativeLabel: 'हिन्दी', flag: '🇮🇳', dayjsLocale: 'hi' },
  'nl-NL': { label: 'Dutch', nativeLabel: 'Nederlands', flag: '🇳🇱', dayjsLocale: 'nl' },
  'pl-PL': { label: 'Polish', nativeLabel: 'Polski', flag: '🇵🇱', dayjsLocale: 'pl' },
  'vi-VN': { label: 'Vietnamese', nativeLabel: 'Tiếng Việt', flag: '🇻🇳', dayjsLocale: 'vi' },
  'th-TH': { label: 'Thai', nativeLabel: 'ไทย', flag: '🇹🇭', dayjsLocale: 'th' },
}

const dayjsLocaleLoaders: Partial<Record<SupportedLocale, () => Promise<unknown>>> = {
  'zh-CN': () => import('dayjs/locale/zh-cn'),
  'ja-JP': () => import('dayjs/locale/ja'),
  'ko-KR': () => import('dayjs/locale/ko'),
  'es-ES': () => import('dayjs/locale/es'),
  'ar-SA': () => import('dayjs/locale/ar'),
  'ru-RU': () => import('dayjs/locale/ru'),
  'fr-FR': () => import('dayjs/locale/fr'),
  'de-DE': () => import('dayjs/locale/de'),
  'pt-BR': () => import('dayjs/locale/pt-br'),
  'it-IT': () => import('dayjs/locale/it'),
  'id-ID': () => import('dayjs/locale/id'),
  'tr-TR': () => import('dayjs/locale/tr'),
  'hi-IN': () => import('dayjs/locale/hi'),
  'nl-NL': () => import('dayjs/locale/nl'),
  'pl-PL': () => import('dayjs/locale/pl'),
  'vi-VN': () => import('dayjs/locale/vi'),
  'th-TH': () => import('dayjs/locale/th'),
}

const loadedDayjsLocales = new Set<string>(['en'])
let pendingDayjsLocale = LOCALE_META[DEFAULT_LOCALE].dayjsLocale
let additionalMessagesPromise: Promise<Record<string, LocaleMessageOverrides>> | null = null

function syncDayjsLocale(locale: SupportedLocale) {
  const dayjsLocale = LOCALE_META[locale].dayjsLocale
  pendingDayjsLocale = dayjsLocale

  const applyLocale = () => {
    if (pendingDayjsLocale === dayjsLocale) {
      dayjs.locale(dayjsLocale)
    }
  }

  if (loadedDayjsLocales.has(dayjsLocale)) {
    applyLocale()
    return
  }

  const loader = dayjsLocaleLoaders[locale]
  if (!loader) {
    applyLocale()
    return
  }

  loader()
    .then(() => {
      loadedDayjsLocales.add(dayjsLocale)
      applyLocale()
    })
    .catch(applyLocale)
}

function loadAdditionalMessages(): Promise<Record<string, LocaleMessageOverrides>> {
  if (!additionalMessagesPromise) {
    additionalMessagesPromise = import('./locales/additional').then(module => module.additionalLocaleMessages)
  }
  return additionalMessagesPromise
}

function mergeMessages<T extends Record<string, any>>(base: T, overrides: Record<string, any> | undefined): T {
  if (!overrides) return { ...base }
  const result: Record<string, any> = { ...base }
  for (const [key, value] of Object.entries(overrides)) {
    const baseValue = result[key]
    if (
      value &&
      typeof value === 'object' &&
      !Array.isArray(value) &&
      baseValue &&
      typeof baseValue === 'object' &&
      !Array.isArray(baseValue)
    ) {
      result[key] = mergeMessages(baseValue, value)
    } else {
      result[key] = value
    }
  }
  return result as T
}

export const messages = {
  'en-US': enUS,
  'zh-CN': zhCN,
} satisfies Partial<Record<SupportedLocale, Record<string, any>>>

const loadedMessageLocales = new Set<SupportedLocale>(['en-US', 'zh-CN'])

const localeChangedHandlers = new Set<LocaleChangedHandler>()

function normalizeLocale(locale: string | null | undefined): SupportedLocale {
  if (!locale) return DEFAULT_LOCALE
  const normalized = locale.toLowerCase()
  if (normalized.startsWith('zh')) return 'zh-CN'
  if (normalized.startsWith('ja')) return 'ja-JP'
  if (normalized.startsWith('ko')) return 'ko-KR'
  if (normalized.startsWith('es')) return 'es-ES'
  if (normalized.startsWith('ar')) return 'ar-SA'
  if (normalized.startsWith('ru')) return 'ru-RU'
  if (normalized.startsWith('fr')) return 'fr-FR'
  if (normalized.startsWith('de')) return 'de-DE'
  if (normalized.startsWith('pt')) return 'pt-BR'
  if (normalized.startsWith('it')) return 'it-IT'
  if (normalized.startsWith('id')) return 'id-ID'
  if (normalized.startsWith('tr')) return 'tr-TR'
  if (normalized.startsWith('hi')) return 'hi-IN'
  if (normalized.startsWith('nl')) return 'nl-NL'
  if (normalized.startsWith('pl')) return 'pl-PL'
  if (normalized.startsWith('vi')) return 'vi-VN'
  if (normalized.startsWith('th')) return 'th-TH'
  if (normalized.startsWith('en')) return 'en-US'
  return DEFAULT_LOCALE
}

export function getInitialLocale(): SupportedLocale {
  const savedLocale = localStorage.getItem(LOCALE_STORAGE_KEY)
  if (savedLocale) {
    return normalizeLocale(savedLocale)
  }
  return normalizeLocale(navigator.language)
}

export function syncRuntimeLocale(locale: SupportedLocale) {
  document.documentElement.setAttribute('lang', locale)
  document.documentElement.setAttribute('dir', LOCALE_META[locale].dir || 'ltr')
  localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  syncDayjsLocale(locale)
}

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: getInitialLocale(),
  fallbackLocale: DEFAULT_LOCALE,
  messages,
})

const i18nGlobal = i18n.global as unknown as {
  locale: { value: string }
  t: (key: string, params?: Record<string, unknown>) => string
  setLocaleMessage: (locale: string, message: Record<string, any>) => void
}

async function ensureLocaleMessages(locale: SupportedLocale): Promise<boolean> {
  if (loadedMessageLocales.has(locale)) {
    return false
  }

  const additionalLocaleMessages = await loadAdditionalMessages()
  i18nGlobal.setLocaleMessage(locale, mergeMessages(enUS, additionalLocaleMessages[locale]))
  loadedMessageLocales.add(locale)
  return true
}

syncRuntimeLocale(i18nGlobal.locale.value as SupportedLocale)
void ensureLocaleMessages(i18nGlobal.locale.value as SupportedLocale)

export function setLocale(locale: SupportedLocale) {
  i18nGlobal.locale.value = locale
  syncRuntimeLocale(locale)
  localeChangedHandlers.forEach((handler) => handler(locale))
  void ensureLocaleMessages(locale).then((loaded) => {
    if (loaded && i18nGlobal.locale.value === locale) {
      localeChangedHandlers.forEach((handler) => handler(locale))
    }
  })
}

export function getCurrentLocale(): SupportedLocale {
  return i18nGlobal.locale.value as SupportedLocale
}

export function translate(key: string, params?: Record<string, unknown>): string {
  return i18nGlobal.t(key, params || {})
}

export function onLocaleChanged(handler: LocaleChangedHandler): () => void {
  localeChangedHandlers.add(handler)
  return () => localeChangedHandlers.delete(handler)
}
