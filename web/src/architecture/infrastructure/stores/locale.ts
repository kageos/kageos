import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { DEFAULT_LOCALE, getInitialLocale, LOCALE_META, setLocale, SUPPORTED_LOCALES, type SupportedLocale } from '@/architecture/shared/i18n'

export const useLocaleStore = defineStore('locale', () => {
  const currentLocale = ref<SupportedLocale>(getInitialLocale())

  const localeOptions = computed(() => SUPPORTED_LOCALES.map((value) => ({
    value,
    label: LOCALE_META[value].nativeLabel,
    nativeLabel: LOCALE_META[value].nativeLabel,
    englishLabel: LOCALE_META[value].label,
    flag: LOCALE_META[value].flag,
    dir: LOCALE_META[value].dir || 'ltr',
  })))

  function initLocale() {
    setAppLocale(currentLocale.value || DEFAULT_LOCALE)
  }

  function setAppLocale(locale: SupportedLocale) {
    currentLocale.value = locale
    setLocale(locale)
  }

  return {
    currentLocale,
    localeOptions,
    initLocale,
    setAppLocale,
  }
})
