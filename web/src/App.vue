<script setup lang="ts">
import { computed } from 'vue'
import { RouterView } from 'vue-router'
import en from 'element-plus/es/locale/lang/en'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import ja from 'element-plus/es/locale/lang/ja'
import ko from 'element-plus/es/locale/lang/ko'
import es from 'element-plus/es/locale/lang/es'
import ar from 'element-plus/es/locale/lang/ar'
import ru from 'element-plus/es/locale/lang/ru'
import fr from 'element-plus/es/locale/lang/fr'
import de from 'element-plus/es/locale/lang/de'
import ptBr from 'element-plus/es/locale/lang/pt-br'
import it from 'element-plus/es/locale/lang/it'
import id from 'element-plus/es/locale/lang/id'
import tr from 'element-plus/es/locale/lang/tr'
import hi from 'element-plus/es/locale/lang/hi'
import nl from 'element-plus/es/locale/lang/nl'
import pl from 'element-plus/es/locale/lang/pl'
import vi from 'element-plus/es/locale/lang/vi'
import th from 'element-plus/es/locale/lang/th'
import { useLocaleStore } from '@/architecture/presentation/context/appStoresContext'
import type { SupportedLocale } from '@/architecture/shared/i18n'

const localeStore = useLocaleStore()
const elementLocales: Record<SupportedLocale, typeof en> = {
  'en-US': en,
  'zh-CN': zhCn,
  'ja-JP': ja,
  'ko-KR': ko,
  'es-ES': es,
  'ar-SA': ar,
  'ru-RU': ru,
  'fr-FR': fr,
  'de-DE': de,
  'pt-BR': ptBr,
  'it-IT': it,
  'id-ID': id,
  'tr-TR': tr,
  'hi-IN': hi,
  'nl-NL': nl,
  'pl-PL': pl,
  'vi-VN': vi,
  'th-TH': th,
}
const elementLocale = computed(() => elementLocales[localeStore.currentLocale] || en)
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
