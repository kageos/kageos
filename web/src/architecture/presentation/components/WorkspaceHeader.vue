<!--
  WorkspaceHeader - 工作空间顶部导航栏组件
  
  职责：
  - 工作空间切换、消息、搜索与用户菜单
  - 高级入口由产品功能开关统一控制
-->

<template>
  <div class="workspace-header" data-testid="workspace-header">
    <div class="header-left" data-testid="workspace-header-left">
      <AppSwitcher
        ref="appSwitcherRef"
        compact
        :current-app="currentApp"
        :app-list="appList"
        :loading-apps="loadingApps"
        @switch-app="$emit('switch-app', $event)"
        @create-app="$emit('create-app')"
        @update-app="$emit('update-app', $event)"
        @delete-app="$emit('delete-app', $event)"
        @load-apps="$emit('load-apps')"
      />
    </div>
    <div class="header-right" data-testid="workspace-header-right">
      <el-button
        size="small"
        class="header-search-button"
        @click="showGlobalSearchDialog = true"
        :title="t('workspace.globalSearch')"
        data-testid="workspace-header-search"
      >
        <el-icon><Search /></el-icon>
        {{ t('workspace.search') }}
      </el-button>
      <WorkspaceInbox
        :service-tree="serviceTree || []"
        :current-app="currentApp"
        :app-list="appList"
      />

      <el-dropdown
        @command="handleUserCommand"
        trigger="click"
        popper-class="workspace-user-dropdown-popper"
      >
        <span class="user-profile" data-testid="workspace-header-user-menu">
          <UserAvatar :size="32" :src="userAvatar" :alt="userName" />
          <span class="username">{{ userName }}</span>
          <el-icon class="el-icon--right"><ArrowDown /></el-icon>
        </span>
        <template #dropdown>
          <el-dropdown-menu class="workspace-user-dropdown">
            <el-dropdown-item disabled class="user-dropdown-account">
              <span class="user-account-avatar">
                <UserAvatar :size="42" :src="userAvatar" :alt="userName" />
              </span>
              <span class="user-account-copy">
                <span class="user-account-name">{{ userName }}</span>
                <span class="user-account-meta">{{ userEmail || t('workspace.loggedInAccount') }}</span>
              </span>
              <span class="user-account-badge">MVP</span>
            </el-dropdown-item>

            <div
              v-if="companyName || companyCode"
              class="user-company-card"
              :title="companyTitle"
              data-testid="workspace-user-menu-company"
              @click.stop
            >
              <el-avatar :size="34" :src="companyLogo" class="user-company-logo">
                {{ companyInitials }}
              </el-avatar>
              <span class="user-company-copy">
                <span class="user-company-label">{{ t('workspace.company') }}</span>
                <span class="user-company-name">{{ companyTitle }}</span>
              </span>
            </div>

            <el-dropdown-item disabled class="user-dropdown-section-title">{{ t('workspace.account') }}</el-dropdown-item>
            <el-dropdown-item command="settings" class="user-dropdown-action">
              <span class="user-menu-icon user-menu-icon--profile">
                <el-icon><UserFilled /></el-icon>
              </span>
              <span class="user-menu-copy">
                <span class="user-menu-title">{{ t('workspace.profileSettings') }}</span>
                <span class="user-menu-desc">{{ t('workspace.profileDesc') }}</span>
              </span>
            </el-dropdown-item>
            <el-dropdown-item v-if="featureFlags.llmManagement" command="agent" class="user-dropdown-action">
              <span class="user-menu-icon user-menu-icon--llm">
                <el-icon><Cpu /></el-icon>
              </span>
              <span class="user-menu-copy">
                <span class="user-menu-title">{{ t('workspace.llmConfig') }}</span>
                <span class="user-menu-desc">{{ t('workspace.llmDesc') }}</span>
              </span>
            </el-dropdown-item>
            <el-dropdown-item v-if="isSystemUser" command="system-settings" class="user-dropdown-action">
              <span class="user-menu-icon user-menu-icon--profile">
                <el-icon><Setting /></el-icon>
              </span>
              <span class="user-menu-copy">
                <span class="user-menu-title">{{ t('route.systemSettings') }}</span>
                <span class="user-menu-desc">{{ t('workspace.systemSettingsDesc') }}</span>
              </span>
            </el-dropdown-item>
            <el-dropdown-item v-if="isSystemUser" command="login-settings" class="user-dropdown-action">
              <span class="user-menu-icon user-menu-icon--role">
                <el-icon><Key /></el-icon>
              </span>
              <span class="user-menu-copy">
                <span class="user-menu-title">{{ t('workspace.loginConfig') }}</span>
                <span class="user-menu-desc">{{ t('workspace.loginConfigDesc') }}</span>
              </span>
            </el-dropdown-item>
            <el-dropdown-item v-if="isSystemUser" command="connector-settings" class="user-dropdown-action">
              <span class="user-menu-icon user-menu-icon--connector">
                <el-icon><Connection /></el-icon>
              </span>
              <span class="user-menu-copy">
                <span class="user-menu-title">{{ t('workspace.connectorConfig') }}</span>
                <span class="user-menu-desc">{{ t('workspace.connectorConfigDesc') }}</span>
              </span>
            </el-dropdown-item>

            <el-dropdown-item disabled class="user-dropdown-section-title user-dropdown-section-title--divided">{{ t('workspace.help') }}</el-dropdown-item>
            <el-dropdown-item command="help-docs" class="user-dropdown-action">
              <span class="user-menu-icon user-menu-icon--docs">
                <el-icon><QuestionFilled /></el-icon>
              </span>
              <span class="user-menu-copy">
                <span class="user-menu-title">{{ t('workspace.helpDocs') }}</span>
                <span class="user-menu-desc">{{ t('workspace.helpDocsDesc') }}</span>
              </span>
            </el-dropdown-item>

            <el-dropdown-item disabled class="user-dropdown-section-title user-dropdown-section-title--divided">{{ t('workspace.preferences') }}</el-dropdown-item>
            <div class="user-preference-panel" @click.stop>
              <div class="user-preference-row">
                <span class="user-preference-label">{{ t('common.language') }}</span>
                <el-select
                  class="user-preference-select"
                  size="small"
                  :model-value="localeStore.currentLocale"
                  @change="handleLocaleChange"
                >
                  <el-option
                    v-for="option in localeStore.localeOptions"
                    :key="option.value"
                    :label="`${option.flag} ${option.nativeLabel}`"
                    :value="option.value"
                  />
                </el-select>
              </div>
              <div class="user-preference-row">
                <span class="user-preference-label">{{ t('workspace.theme') }}</span>
                <el-select
                  class="user-preference-select"
                  size="small"
                  :model-value="currentThemeName"
                  @change="handleThemeChange"
                >
                  <el-option
                    v-for="theme in availableThemes"
                    :key="theme.name"
                    :label="theme.label"
                    :value="theme.name"
                  />
                </el-select>
              </div>
            </div>

            <el-dropdown-item command="logout" class="user-dropdown-action user-dropdown-action--logout">
              <span class="user-menu-icon user-menu-icon--logout">
                <el-icon><SwitchButton /></el-icon>
              </span>
              <span class="user-menu-copy">
                <span class="user-menu-title">{{ t('workspace.logout') }}</span>
                <span class="user-menu-desc">{{ t('workspace.logoutDesc') }}</span>
              </span>
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <GlobalResourceSearchDialog
      v-if="showGlobalSearchDialog"
      v-model:visible="showGlobalSearchDialog"
    />

  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  ArrowDown,
  UserFilled,
  Connection,
  Cpu,
  Key,
  QuestionFilled,
  Setting,
  Search,
  SwitchButton
} from '@element-plus/icons-vue'
import AppSwitcher from '@/architecture/presentation/shared/components/AppSwitcher.vue'
import type { App, ServiceTree } from '@/architecture/domain/types'
import { ElMessageBox } from 'element-plus'
import { useAuthStore, useLocaleStore, useThemeStore } from '@/architecture/presentation/context/appStoresContext'
import WorkspaceInbox from './WorkspaceInbox.vue'
import { featureFlags } from '@/architecture/shared/config/features'
import { getKageosDocsURL, openExternalURL } from '@/architecture/shared/config/externalLinks'
import type { SupportedLocale } from '@/architecture/shared/i18n'
import defaultCompanyLogo from '@/architecture/presentation/assets/logo.svg'
import UserAvatar from '@/architecture/presentation/shared/components/UserAvatar.vue'

const GlobalResourceSearchDialog = defineAsyncComponent(() => import('./GlobalResourceSearchDialog.vue'))

defineProps<{
  currentApp: App | null
  appList: App[]
  loadingApps: boolean
  serviceTree?: ServiceTree[]
}>()

defineEmits<{
  (e: 'switch-app', app: App): void
  (e: 'create-app'): void
  (e: 'update-app', app: App): void
  (e: 'delete-app', app: App): void
  (e: 'load-apps'): void
}>()

const router = useRouter()
const authStore = useAuthStore()
const localeStore = useLocaleStore()
const themeStore = useThemeStore()
const { t } = useI18n()

// 用户相关
const userName = computed(() => authStore.userName || 'User')
const userEmail = computed(() => authStore.userEmail || '')
const isSystemUser = computed(() => userName.value === 'system')
const userAvatar = computed(() => authStore.user?.avatar || '')
const companyName = computed(() => authStore.user?.company_name || authStore.user?.company_code || '')
const companyCode = computed(() => authStore.user?.company_code || '')
const companyLogo = computed(() => authStore.user?.company_logo_url || defaultCompanyLogo)
const companyInitials = computed(() => {
  const source = companyName.value || companyCode.value || 'CO'
  return source.substring(0, 2).toUpperCase()
})
const companyTitle = computed(() => {
  if (!companyCode.value || companyCode.value === companyName.value) {
    return companyName.value
  }
  return `${companyName.value} (${companyCode.value})`
})
const availableThemes = computed(() => themeStore.getAvailableThemes())
const currentThemeName = computed(() => themeStore.currentTheme.name)

const handleUserCommand = (command: string) => {
  switch (command) {
    case 'logout':
      handleLogout()
      break
    case 'settings':
      router.push('/user/settings')
      break
    case 'agent':
      router.push('/agent/llm')
      break
    case 'system-settings':
      router.push('/system/settings')
      break
    case 'login-settings':
      router.push({ path: '/system/settings', query: { tab: 'login' } })
      break
    case 'connector-settings':
      router.push({ path: '/system/settings', query: { tab: 'connectors' } })
      break
    case 'help-docs':
      openExternalURL(getKageosDocsURL('docs', localeStore.currentLocale))
      break
    default:
      break
  }
}

const handleLocaleChange = (value: string | number | boolean | object) => {
  if (typeof value !== 'string') {
    return
  }
  localeStore.setAppLocale(value as SupportedLocale)
}

const handleThemeChange = (themeName: string | number | boolean | object) => {
  if (typeof themeName !== 'string') {
    return
  }
  const theme = themeStore.getAvailableThemes().find(item => item.name === themeName)
  if (theme) {
    themeStore.setTheme(theme)
  }
}

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm(t('workspace.logoutConfirm'), t('workspace.logoutConfirmTitle'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    await authStore.logout()
  } catch {
    // 忽略取消操作
  }
}

const showGlobalSearchDialog = ref(false)

const appSwitcherRef = ref<InstanceType<typeof AppSwitcher> | null>(null)

function openWorkspaceListDialog(forceSelect = false) {
  appSwitcherRef.value?.openWorkspaceListDialog(forceSelect)
}

defineExpose({
  openWorkspaceListDialog
})
</script>

<style scoped lang="scss">
.workspace-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  min-height: 72px;
  padding: 14px 22px;
  background: var(--app-shell-panel-bg);
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 22px;
  box-shadow: var(--app-shell-panel-shadow);
  position: relative;
  overflow: visible;
  isolation: isolate;
}

.workspace-header::before {
  content: '';
  position: absolute;
  top: 0;
  left: 24px;
  right: 24px;
  height: 1px;
  background: var(--app-shell-panel-highlight);
  opacity: 0.7;
  pointer-events: none;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.user-profile {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 7px 12px;
  border-radius: 12px;
  border: 1px solid transparent;
  background: var(--app-shell-panel-muted-bg);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
  transition: all 0.2s ease;
  max-width: 220px;

  &:hover {
    background: var(--app-shell-panel-bg-strong);
    border-color: var(--app-shell-panel-border);
    box-shadow: var(--app-shell-panel-shadow-soft);
  }

  .el-avatar {
    flex-shrink: 0;
    border: 1px solid rgba(var(--el-color-primary-rgb), 0.18);
    box-shadow: 0 6px 18px rgba(var(--el-color-primary-rgb), 0.12);
  }

  .el-icon--right {
    flex-shrink: 0;
    color: var(--el-text-color-secondary);
    transition: transform 0.2s ease, color 0.2s ease;
  }

  &:hover .el-icon--right {
    color: var(--el-color-primary);
  }
}

.username {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.workspace-user-dropdown-popper.el-popper) {
  border: none;
  background: transparent;
  box-shadow: none;
}

:global(.workspace-user-dropdown-popper.el-popper.is-light) {
  border: none;
}

:global(.workspace-user-dropdown-popper .el-popper__arrow::before) {
  border-color: var(--app-shell-panel-border);
  background: var(--app-shell-panel-bg);
}

.workspace-user-dropdown {
  width: min(340px, calc(100vw - 24px));
  max-height: min(680px, calc(100vh - 116px));
  padding: 10px;
  overflow-y: auto;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 18px;
  background:
    linear-gradient(135deg, rgba(var(--el-color-primary-rgb), 0.08), transparent 42%),
    var(--app-shell-panel-bg);
  box-shadow: 0 24px 56px rgba(15, 23, 42, 0.2);
  backdrop-filter: blur(18px);
  scrollbar-gutter: stable;

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-thumb {
    border-radius: 999px;
    background: rgba(var(--el-color-primary-rgb), 0.22);
  }

  :deep(.el-dropdown-menu__item) {
    line-height: 1.25;
    white-space: normal;
  }

  :deep(.el-dropdown-menu__item.is-disabled) {
    cursor: default;
    color: inherit;
    opacity: 1;
  }
}

.user-dropdown-account {
  min-height: 60px;
  padding: 8px 9px !important;
  margin-bottom: 6px;
  border-radius: 14px;
  background:
    linear-gradient(135deg, rgba(var(--el-color-primary-rgb), 0.14), rgba(34, 197, 94, 0.08)),
    var(--app-shell-panel-bg-strong);
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.12);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
}

.user-account-avatar {
  display: inline-flex;
  flex-shrink: 0;

  .el-avatar {
    border: 1px solid rgba(var(--el-color-primary-rgb), 0.2);
    box-shadow: 0 10px 22px rgba(var(--el-color-primary-rgb), 0.14);
  }
}

.user-account-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 4px;
  margin-left: 10px;
}

.user-account-name,
.user-account-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-account-name {
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
}

.user-account-meta {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.user-account-badge {
  flex-shrink: 0;
  padding: 4px 7px;
  border-radius: 999px;
  background: rgba(var(--el-color-primary-rgb), 0.1);
  color: var(--el-color-primary);
  font-size: 11px;
  font-weight: 700;
}

.user-company-card {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 8px 9px;
  margin-bottom: 6px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 14px;
  background: var(--app-shell-panel-muted-bg);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
}

.user-company-logo {
  flex-shrink: 0;
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.16);
  background: rgba(var(--el-color-primary-rgb), 0.1);
  color: var(--el-color-primary);
  font-size: 12px;
  font-weight: 800;
}

.user-company-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.user-company-label,
.user-company-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-company-label {
  color: var(--el-text-color-secondary);
  font-size: 11px;
  font-weight: 700;
}

.user-company-name {
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 700;
}

.user-dropdown-section-title {
  min-height: 22px;
  padding: 8px 8px 3px !important;
  color: var(--el-text-color-secondary) !important;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0;
}

.user-dropdown-section-title--divided {
  margin-top: 8px;
  border-top: 1px solid var(--app-shell-panel-border);
}

.user-dropdown-action {
  min-height: 48px;
  gap: 10px;
  padding: 7px 9px !important;
  margin: 2px 0;
  border-radius: 13px;
  color: var(--el-text-color-primary);
  transition: background 0.16s ease, color 0.16s ease, box-shadow 0.16s ease;

  &:hover,
  &:focus {
    background: rgba(var(--el-color-primary-rgb), 0.08) !important;
    color: var(--el-text-color-primary) !important;
    box-shadow: inset 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.12);
  }
}

.user-dropdown-action--logout {
  margin-top: 8px;
  border-top: 1px solid var(--app-shell-panel-border);
  border-radius: 0 0 13px 13px;
  color: var(--el-color-danger);

  &:hover,
  &:focus {
    background: rgba(239, 68, 68, 0.08) !important;
    color: var(--el-color-danger) !important;
    box-shadow: inset 0 0 0 1px rgba(239, 68, 68, 0.12);
  }
}

.user-menu-icon {
  display: inline-flex;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 11px;
  font-size: 17px;
}

.user-menu-icon--profile {
  background: rgba(var(--el-color-primary-rgb), 0.12);
  color: var(--el-color-primary);
}

.user-menu-icon--llm {
  background: rgba(14, 165, 233, 0.12);
  color: #0284c7;
}

.user-menu-icon--connector {
  background: rgba(22, 163, 74, 0.12);
  color: #15803d;
}

.user-menu-icon--role {
  background: rgba(99, 102, 241, 0.12);
  color: #4f46e5;
}

.user-menu-icon--docs {
  background: rgba(245, 158, 11, 0.12);
  color: #d97706;
}

.user-menu-icon--logout {
  background: rgba(239, 68, 68, 0.1);
  color: var(--el-color-danger);
}

.user-menu-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 4px;
}

.user-menu-title,
.user-menu-desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-menu-title {
  color: inherit;
  font-size: 13px;
  font-weight: 700;
}

.user-menu-desc {
  color: var(--el-text-color-secondary);
  font-size: 11px;
  font-weight: 500;
}

.header-search-button {
  height: 40px;
  padding: 0 14px;
  border-radius: 12px;
  background: var(--app-shell-panel-muted-bg);
  border-color: var(--app-shell-panel-border);
  color: var(--el-text-color-primary);
  font-weight: 600;
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);

  .el-icon {
    margin-right: 4px;
  }

  &:hover {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary-light-5);
    background: var(--el-color-primary-light-9);
  }
}

.user-preference-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 9px 10px;
  margin: 2px 0 4px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 14px;
  background: color-mix(in srgb, var(--app-shell-panel-muted-bg) 72%, transparent);
}

.user-preference-row {
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
}

.user-preference-label {
  min-width: 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 700;
}

.user-preference-select {
  width: 100%;
}

@media (max-width: 760px) {
  .username {
    display: none;
  }

  .user-profile {
    max-width: none;
    padding: 7px 9px;
  }
}
</style>
