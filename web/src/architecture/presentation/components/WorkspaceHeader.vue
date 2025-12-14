<!--
  WorkspaceHeader - 工作空间顶部导航栏组件
  
  职责：
  - 显示 Logo
  - 主题切换
  - 用户信息展示和操作
-->

<template>
  <div class="workspace-header">
    <div class="header-left">
      <div class="logo">AI Agent OS</div>
    </div>
    <div class="header-right">
      <!-- 🔥 开发工具：Debug 弹窗按钮 -->
      <el-button
        v-if="isDevelopment"
        type="info"
        size="small"
        :icon="Delete"
        @click="showDebugDialog = true"
        title="开发调试工具"
      >
        Debug
      </el-button>
      
      <!-- Hub 和 Agent 路由链接 -->
      <el-button
        type="primary"
        size="small"
        @click="navigateToHub"
        title="应用中心"
      >
        应用中心
      </el-button>
      
      <!-- 升级企业版按钮 -->
      <el-button
        v-if="!licenseStore.isEnterprise"
        type="success"
        size="small"
        @click="showUpgradeDialog = true"
        title="升级企业版"
      >
        升级企业版
      </el-button>
      
      <!-- 企业版标识和注销按钮 -->
      <template v-else>
        <el-tag type="success" size="small">
          {{ licenseStore.edition }}
        </el-tag>
        <el-button
          type="warning"
          size="small"
          :icon="Delete"
          @click="handleDeactivate"
          title="注销 License（测试用）"
        >
          注销 License
        </el-button>
      </template>
      
      <el-button
        type="primary"
        size="small"
        @click="navigateToAgent"
        title="智能体管理"
      >
        智能体管理
      </el-button>
      
      <ThemeToggle />
      <el-dropdown @command="handleUserCommand">
        <span class="user-profile">
          <el-avatar :size="32" :src="userAvatar || undefined">{{ userInitials }}</el-avatar>
          <span class="username">{{ userName }}</span>
          <el-icon class="el-icon--right"><ArrowDown /></el-icon>
        </span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="settings">个人设置</el-dropdown-item>
            <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
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
import { ArrowDown, Delete } from '@element-plus/icons-vue'
import { ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useLicenseStore } from '@/stores/license'
import ThemeToggle from '@/components/ThemeToggle.vue'
import DebugDialog from './DebugDialog.vue'
import UpgradeEnterpriseDialog from '@/components/UpgradeEnterpriseDialog.vue'
import { navigateToHub as navigateToHubUtil } from '@/utils/hub-navigation'

const router = useRouter()
const authStore = useAuthStore()
const licenseStore = useLicenseStore()

// 用户相关
const userName = computed(() => authStore.userName || 'User')
const userAvatar = computed(() => authStore.user?.avatar || '')
const userInitials = computed(() => {
  const name = userName.value
  return name ? name.substring(0, 2).toUpperCase() : 'US'
})

const handleUserCommand = (command: string) => {
  if (command === 'logout') {
    handleLogout()
  } else if (command === 'settings') {
    router.push('/user/settings')
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

// 导航到 Hub
const navigateToHub = () => {
  navigateToHubUtil('/')
}

// 导航到 Agent
const navigateToAgent = () => {
  router.push('/agent')
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
onMounted(() => {
  // 从本地加载（不主动调用接口，依赖定时检查和激活时的保存）
  licenseStore.loadFromLocal()
  
  // 如果已有激活的 License，启动定时检查（如果还没启动的话）
  if (licenseStore.isEnterprise && !licenseStore.isExpired) {
    licenseStore.startPeriodicCheck()
  }
})
</script>

<style scoped lang="scss">
.workspace-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 60px;
  padding: 0 24px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.header-left {
  display: flex;
  align-items: center;
}

.logo {
  font-size: 20px;
  font-weight: 600;
  color: #6366f1; /* ✅ 与服务目录 fx 图标颜色一致（indigo-500） */
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


