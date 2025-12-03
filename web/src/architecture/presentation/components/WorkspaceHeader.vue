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
      <!-- 🔥 开发工具：清理缓存按钮 -->
      <el-button
        v-if="isDevelopment"
        type="info"
        size="small"
        :icon="Delete"
        @click="handleClearCache"
        title="清理路由缓存（开发工具）"
      >
        清理缓存
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
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { ArrowDown, Delete } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { functionLoader } from '../../infrastructure/functionLoader'

const router = useRouter()
const authStore = useAuthStore()

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

// 🔥 开发工具：清理缓存
const isDevelopment = computed(() => {
  // 检查是否是开发环境（可以通过环境变量或 URL 参数判断）
  return import.meta.env.DEV || window.location.search.includes('dev=true')
})

const handleClearCache = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要清理所有路由缓存吗？这将清除函数详情缓存，需要重新加载。',
      '清理缓存',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    // 清理函数加载器缓存
    functionLoader.clearCache()
    
    ElMessage.success('缓存已清理')
    
    // 刷新当前页面以重新加载数据
    window.location.reload()
  } catch (error) {
    // 忽略取消操作
  }
}
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


