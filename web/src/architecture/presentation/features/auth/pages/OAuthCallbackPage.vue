<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'

const router = useRouter()
const authStore = useAuthStore()
const message = ref('正在完成授权登录...')

onMounted(async () => {
  const params = new URLSearchParams(window.location.hash.replace(/^#/, ''))
  const token = params.get('token') || ''
  const refreshToken = params.get('refresh_token') || ''
  const redirectAfter = params.get('redirect_after') || ''

  if (!token || !refreshToken) {
    ElMessage.error('授权登录结果无效，请重新登录')
    await router.replace('/login')
    return
  }

  try {
    await authStore.completeOAuthLogin(token, refreshToken, redirectAfter)
  } catch (error: any) {
    message.value = '授权登录失败'
    ElMessage.error(error?.response?.data?.msg || error?.message || '授权登录失败')
    await router.replace('/login')
  }
})
</script>

<template>
  <div class="oauth-callback-page">
    <div class="callback-panel">
      <el-icon class="callback-loading"><Loading /></el-icon>
      <div class="callback-title">{{ message }}</div>
    </div>
  </div>
</template>

<style scoped>
.oauth-callback-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #eef4ff 0%, #f7fafc 52%, #e9f7ef 100%);
}

.callback-panel {
  display: flex;
  min-width: 260px;
  align-items: center;
  gap: 12px;
  padding: 18px 20px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.86);
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.12);
}

.callback-loading {
  color: var(--el-color-primary);
  font-size: 22px;
  animation: callback-spin 1s linear infinite;
}

.callback-title {
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 700;
}

@keyframes callback-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
