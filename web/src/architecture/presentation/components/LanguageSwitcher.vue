<template>
  <el-dropdown
    trigger="click"
    placement="bottom-end"
    popper-class="language-switcher-popper"
    @command="handleChange"
  >
    <button class="language-switcher" type="button" :aria-label="t('common.language')">
      <span class="language-switcher__flag" aria-hidden="true">{{ currentOption?.flag }}</span>
      <span class="language-switcher__copy">
        <span class="language-switcher__name">{{ currentOption?.nativeLabel }}</span>
        <span class="language-switcher__code">{{ localeStore.currentLocale }}</span>
      </span>
      <el-icon class="language-switcher__arrow"><ArrowDown /></el-icon>
    </button>

    <template #dropdown>
      <el-dropdown-menu class="language-menu">
        <el-dropdown-item
          v-for="option in localeStore.localeOptions"
          :key="option.value"
          :command="option.value"
          class="language-menu__item"
          :class="{ 'is-current': option.value === localeStore.currentLocale }"
        >
          <span class="language-option" :dir="option.dir">
            <span class="language-option__flag" aria-hidden="true">{{ option.flag }}</span>
            <span class="language-option__copy">
              <span class="language-option__native">{{ option.nativeLabel }}</span>
              <span class="language-option__english">{{ option.englishLabel }}</span>
            </span>
            <span v-if="option.value === localeStore.currentLocale" class="language-option__check">
              <el-icon><Check /></el-icon>
            </span>
          </span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowDown, Check } from '@element-plus/icons-vue'
import { ElDropdown, ElDropdownItem, ElDropdownMenu, ElIcon } from 'element-plus'
import { useLocaleStore } from '@/architecture/presentation/context/appStoresContext'
import type { SupportedLocale } from '@/architecture/shared/i18n'

const { t } = useI18n()
const localeStore = useLocaleStore()

const currentOption = computed(() => localeStore.localeOptions.find((option) => option.value === localeStore.currentLocale))

function handleChange(command: string | number | object) {
  if (typeof command !== 'string') return
  localeStore.setAppLocale(command as SupportedLocale)
}
</script>

<style scoped lang="scss">
.language-switcher {
  display: inline-flex;
  width: 168px;
  max-width: min(46vw, 188px);
  min-height: 38px;
  align-items: center;
  gap: 9px;
  padding: 6px 10px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-light));
  border-radius: 12px;
  background: var(--app-shell-panel-muted-bg, color-mix(in srgb, var(--el-bg-color) 92%, transparent));
  color: var(--el-text-color-primary);
  cursor: pointer;
  font: inherit;
  text-align: left;
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.52));
  transition: border-color 0.18s ease, background 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
}

.language-switcher:hover,
.language-switcher:focus-visible {
  border-color: var(--app-shell-panel-border, var(--el-color-primary-light-5));
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  box-shadow: var(--app-shell-panel-shadow-soft, 0 8px 24px rgba(15, 23, 42, 0.08));
  transform: none;
  outline: none;
}

.language-switcher__flag {
  flex: 0 0 auto;
  font-size: 18px;
  line-height: 1;
}

.language-switcher__copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 1px;
}

.language-switcher__name,
.language-switcher__code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.language-switcher__name {
  font-size: 13px;
  font-weight: 700;
  line-height: 1.15;
  color: inherit;
}

.language-switcher__code {
  color: var(--el-text-color-secondary);
  font-size: 10px;
  line-height: 1.1;
}

.language-switcher__arrow {
  flex: 0 0 auto;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
</style>

<style lang="scss">
.language-switcher-popper.el-popper {
  border: 1px solid rgba(148, 163, 184, 0.24);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 22px 52px rgba(15, 23, 42, 0.16);
  backdrop-filter: blur(18px);
  overflow: hidden;
}

.language-switcher-popper .el-popper__arrow::before {
  border-color: rgba(148, 163, 184, 0.24);
  background: rgba(255, 255, 255, 0.96);
}

.language-menu {
  width: 268px;
  max-width: calc(100vw - 24px);
  max-height: min(360px, calc(100vh - 96px));
  padding: 6px;
  overflow-y: auto;
  scrollbar-width: thin;
}

.language-menu .el-dropdown-menu__item {
  display: block;
  height: auto;
  padding: 0;
  border-radius: 8px;
  color: #26364d;
  line-height: 1.2;
}

.language-menu .el-dropdown-menu__item:hover,
.language-menu .el-dropdown-menu__item:focus {
  background: rgba(22, 119, 255, 0.08);
  color: #1e3a5f;
}

.language-menu .el-dropdown-menu__item.is-current {
  background: rgba(22, 119, 255, 0.12);
  color: #1e3a5f;
}

.language-menu .el-dropdown-menu__item.is-current .language-option__native {
  color: #1e3a5f;
}

.language-menu .el-dropdown-menu__item.is-current .language-option__english,
.language-menu .el-dropdown-menu__item.is-current .language-option__check {
  color: #1677ff;
}

.language-option {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
}

.language-option__flag {
  flex: 0 0 auto;
  width: 24px;
  font-size: 18px;
  line-height: 1;
  text-align: center;
}

.language-option__copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.language-option__native,
.language-option__english {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.language-option__native {
  font-size: 13px;
  font-weight: 700;
}

.language-option__english {
  color: #718096;
  font-size: 11px;
}

.language-option__check {
  flex: 0 0 auto;
  font-size: 14px;
}
</style>
