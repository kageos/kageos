<script setup lang="ts">
import { shallowRef, watch } from 'vue'
import { RouterView } from 'vue-router'
import { useLocaleStore } from '@/architecture/presentation/context/appStoresContext'
import type { SupportedLocale } from '@/architecture/shared/i18n'

const localeStore = useLocaleStore()

type ElementLocale = unknown
type ElementLocaleModule = { default: ElementLocale }

const elementLocaleLoaders: Record<SupportedLocale, () => Promise<ElementLocaleModule>> = {
  'en-US': () => import('element-plus/es/locale/lang/en'),
  'zh-CN': () => import('element-plus/es/locale/lang/zh-cn'),
  'ja-JP': () => import('element-plus/es/locale/lang/ja'),
  'ko-KR': () => import('element-plus/es/locale/lang/ko'),
  'es-ES': () => import('element-plus/es/locale/lang/es'),
  'ar-SA': () => import('element-plus/es/locale/lang/ar'),
  'ru-RU': () => import('element-plus/es/locale/lang/ru'),
  'fr-FR': () => import('element-plus/es/locale/lang/fr'),
  'de-DE': () => import('element-plus/es/locale/lang/de'),
  'pt-BR': () => import('element-plus/es/locale/lang/pt-br'),
  'it-IT': () => import('element-plus/es/locale/lang/it'),
  'id-ID': () => import('element-plus/es/locale/lang/id'),
  'tr-TR': () => import('element-plus/es/locale/lang/tr'),
  'hi-IN': () => import('element-plus/es/locale/lang/hi'),
  'nl-NL': () => import('element-plus/es/locale/lang/nl'),
  'pl-PL': () => import('element-plus/es/locale/lang/pl'),
  'vi-VN': () => import('element-plus/es/locale/lang/vi'),
  'th-TH': () => import('element-plus/es/locale/lang/th'),
}

const elementLocale = shallowRef<ElementLocale>()
let localeRequestId = 0

watch(
  () => localeStore.currentLocale,
  async (locale) => {
    const requestId = ++localeRequestId
    const loader = elementLocaleLoaders[locale] || elementLocaleLoaders['en-US']
    const module = await loader()
    if (requestId === localeRequestId) {
      elementLocale.value = module.default
    }
  },
  { immediate: true }
)
</script>

<template>
  <el-config-provider :locale="elementLocale">
    <!-- 🔥 移除 MainLayout，所有页面自己管理布局 -->
    <RouterView />
  </el-config-provider>
</template>

<style>
/* 全局样式 */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html,
body {
  height: 100%;
  overflow-y: auto;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Helvetica Neue', sans-serif;
}

#app {
  min-height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Helvetica Neue', sans-serif;
}
</style>
