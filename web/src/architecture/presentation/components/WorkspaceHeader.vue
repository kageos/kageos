<!--
  WorkspaceHeader - 工作空间顶部导航栏组件
  
  职责：
  - 应用中心入口（唯一保留在栏上的按钮）
  - 其余入口（智能体、组织架构、角色管理、企业版、Debug、主题、退出）放入用户下拉
-->

<template>
  <div class="workspace-header">
    <div class="header-right">
      <!-- 仅保留应用中心在栏上 -->
      <el-button
        type="primary"
        size="small"
        @click="navigateToHub"
        title="应用中心"
      >
        应用中心
      </el-button>

      <el-dropdown @command="handleUserCommand" trigger="click">
        <span class="user-profile">
          <el-avatar :size="32" :src="userAvatar || undefined">{{ userInitials }}</el-avatar>
          <span class="username">{{ userName }}</span>
          <el-icon class="el-icon--right"><ArrowDown /></el-icon>
        </span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="settings">
              <el-icon><UserFilled /></el-icon>
              <span>个人设置</span>
            </el-dropdown-item>
            <el-dropdown-item divided command="agent">
              <el-icon><Cpu /></el-icon>
              <span>智能体管理</span>
            </el-dropdown-item>
            <el-dropdown-item command="organization">
              <el-icon><OfficeBuilding /></el-icon>
              <span>组织架构和用户管理</span>
            </el-dropdown-item>
            <el-dropdown-item command="roles">
              <el-icon><Key /></el-icon>
              <span>角色管理</span>
            </el-dropdown-item>
            <!-- 非企业版：升级企业版 -->
            <el-dropdown-item
              v-if="!licenseStore.isEnterprise"
              divided
              command="upgrade"
            >
              <el-icon><Promotion /></el-icon>
              <span>升级企业版</span>
            </el-dropdown-item>
            <!-- 企业版：标识 + 注销 -->
            <template v-else>
              <el-dropdown-item divided disabled>
                <el-tag type="success" size="small">{{ licenseStore.edition }}</el-tag>
              </el-dropdown-item>
              <el-dropdown-item command="deactivate">
                <el-icon><Delete /></el-icon>
                <span>注销 License</span>
              </el-dropdown-item>
            </template>
            <el-dropdown-item v-if="isDevelopment" divided command="debug">
              <el-icon><Setting /></el-icon>
              <span>开发调试 (Debug)</span>
            </el-dropdown-item>
            <el-dropdown-item divided command="theme">
              <el-icon><Sunny /></el-icon>
              <span>切换主题</span>
            </el-dropdown-item>
            <el-dropdown-item divided command="logout">
              <el-icon><SwitchButton /></el-icon>
              <span>退出登录</span>
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <!-- Debug 弹窗 -->
    <DebugDialog v-model="showDebugDialog" />

    <!-- 升级企业版对话框 -->
    <UpgradeEnterpriseDialog
      v-model="showUpgradeDialog"
      @activated="handleLicenseActivated"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowDown,
  Delete,
  OfficeBuilding,
  UserFilled,
  Cpu,
  Key,
  Promotion,
  Setting,
  Sunny,
  SwitchButton
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useLicenseStore } from '@/stores/license'
import { useThemeStore } from '@/stores/theme'
import DebugDialog from './DebugDialog.vue'
import UpgradeEnterpriseDialog from '@/components/UpgradeEnterpriseDialog.vue'
import { navigateToHub as navigateToHubUtil } from '@/utils/hub-navigation'

const router = useRouter()
const authStore = useAuthStore()
const licenseStore = useLicenseStore()
const themeStore = useThemeStore()

// 用户相关
const userName = computed(() => authStore.userName || 'User')
const userAvatar = computed(() => authStore.user?.avatar || '')
const userInitials = computed(() => {
  const name = userName.value
  return name ? name.substring(0, 2).toUpperCase() : 'US'
})

const handleUserCommand = (command: string) => {
  switch (command) {
    case 'logout':
      handleLogout()
      break
    case 'settings':
      router.push('/user/settings')
      break
    case 'agent':
      router.push('/agent')
      break
    case 'organization':
      router.push('/organization')
      break
    case 'roles':
      router.push('/permissions/roles')
      break
    case 'upgrade':
      showUpgradeDialog.value = true
      break
    case 'deactivate':
      handleDeactivate()
      break
    case 'debug':
      showDebugDialog.value = true
      break
    case 'theme':
      themeStore.toggleTheme()
      break
    default:
      break
  }
}

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await authStore.logout()
  } catch (error) {
    // 忽略取消操作
  }
}

// 🔥 开发工具：Debug 弹窗
const isDevelopment = computed(() => {
  // 检查是否是开发环境（可以通过环境变量或 URL 参数判断）
  return import.meta.env.DEV || window.location.search.includes('dev=true')
})

const showDebugDialog = ref(false)

// 打开智能工作台管理（纯粹管理，不带目录）；工作台对话在详情里由服务目录 ⋮ → 打开工作台 带入 full_code_path
const navigateToWorkstation = (fullCodePath?: string) => {
  const q = typeof fullCodePath === 'string' && fullCodePath.trim() ? fullCodePath.trim() : ''
  const url = window.location.origin + '/workspace/workstation' + (q ? '?full_code_path=' + encodeURIComponent(q) : '')
  window.open(url, '_blank')
}

// 导航到 Hub
const navigateToHub = () => {
  navigateToHubUtil('/')
}

// 导航到 Agent
const navigateToAgent = () => {
  router.push('/agent')
}

// 导航到组织架构和用户管理
const navigateToOrganization = () => {
  router.push('/organization')
}

// 导航到角色管理
const navigateToRoleManagement = () => {
  router.push('/permissions/roles')
}

// 升级企业版对话框
const showUpgradeDialog = ref(false)

// License 激活成功回调
const handleLicenseActivated = async () => {
  // 刷新 License 状态
  await licenseStore.fetchStatus()
}

// License 注销处理
const handleDeactivate = async () => {
  try {
    // 检查方法是否存在
    if (typeof licenseStore.deactivate !== 'function') {
      console.error('licenseStore.deactivate 不是函数', licenseStore)
      ElMessage.error('License Store 未正确初始化，请刷新页面')
      return
    }
    await licenseStore.deactivate()
    // 注销成功后，状态会自动更新（store 中已处理）
  } catch (error) {
    // 错误已在 store 中处理
    console.error('注销 License 失败:', error)
  }
}

// 组件挂载时加载 License 状态
onMounted(async () => {
  // ⭐ 先从本地加载（快速显示，避免闪烁）
  licenseStore.loadFromLocal()
  
  // ⭐ 如果 localStorage 不存在，从后端获取
  // 如果 localStorage 存在，直接使用（快速显示），定时检查会每小时更新
  const hasLocalLicense = licenseStore.license !== null
  if (!hasLocalLicense) {
    // localStorage 不存在，从后端获取
    try {
      await licenseStore.fetchStatus()
    } catch (error) {
      console.warn('[WorkspaceHeader] 获取 License 状态失败:', error)
    }
  }
  
  // ⭐ 启动定时检查（每小时重新获取一次，确保状态同步）
  if (licenseStore.isEnterprise && !licenseStore.isExpired) {
    licenseStore.startPeriodicCheck()
  }
})
</script>

<style scoped lang="scss">
.workspace-header {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  height: 60px;
  padding: 0 24px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-profile {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background-color 0.2s;

  &:hover {
    background-color: var(--el-fill-color-light);
  }
}

.username {
  font-size: 14px;
  color: var(--el-text-color-primary);
}
</style>


