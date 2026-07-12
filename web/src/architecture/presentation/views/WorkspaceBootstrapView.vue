<template>
  <main class="workspace-bootstrap" data-testid="workspace-bootstrap">
    <section class="workspace-bootstrap__card" aria-live="polite">
      <template v-if="isLoading">
        <div class="workspace-bootstrap__icon workspace-bootstrap__icon--loading">
          <el-icon><Loading /></el-icon>
        </div>
        <h1>{{ t('workspace.bootstrapLoading') }}</h1>
        <p>{{ t('workspace.bootstrapLoadingDescription') }}</p>
      </template>

      <template v-else>
        <div class="workspace-bootstrap__icon workspace-bootstrap__icon--error">
          <el-icon><WarningFilled /></el-icon>
        </div>
        <h1>{{ t('workspace.bootstrapFailedTitle') }}</h1>
        <p>{{ errorMessage || t('workspace.bootstrapFailedDescription') }}</p>
        <el-button
          type="primary"
          :icon="Refresh"
          data-testid="workspace-bootstrap-retry"
          @click="bootstrapWorkspace"
        >
          {{ t('workspace.bootstrapRetry') }}
        </el-button>
      </template>
    </section>
  </main>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Loading, Refresh, WarningFilled } from '@element-plus/icons-vue'
import { bootstrapPersonalWorkspace } from '@/architecture/presentation/context/api/app'

const router = useRouter()
const { t } = useI18n()
const isLoading = ref(true)
const errorMessage = ref('')
let isActive = true

function getErrorMessage(error: unknown): string {
  if (!error || typeof error !== 'object') {
    return ''
  }

  const candidate = error as {
    message?: unknown
    response?: { data?: { msg?: unknown; message?: unknown } }
  }
  const responseMessage = candidate.response?.data?.msg ?? candidate.response?.data?.message
  if (typeof responseMessage === 'string' && responseMessage.trim()) {
    return responseMessage.trim()
  }
  return typeof candidate.message === 'string' ? candidate.message.trim() : ''
}

async function bootstrapWorkspace() {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const { app } = await bootstrapPersonalWorkspace()
    const user = app?.user?.trim()
    const code = app?.code?.trim()
    if (!user || !code) {
      throw new Error(t('workspace.bootstrapInvalidResponse'))
    }

    if (!isActive) {
      return
    }

    await router.replace({
      path: `/workspace/${encodeURIComponent(user)}/${encodeURIComponent(code)}`
    })
  } catch (error) {
    if (isActive) {
      errorMessage.value = getErrorMessage(error)
    }
  } finally {
    if (isActive) {
      isLoading.value = false
    }
  }
}

onMounted(() => {
  void bootstrapWorkspace()
})

onBeforeUnmount(() => {
  isActive = false
})
</script>

<style scoped lang="scss">
.workspace-bootstrap {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 24px;
  box-sizing: border-box;
  background: var(--app-shell-bg, var(--el-bg-color-page));
}

.workspace-bootstrap__card {
  width: min(100%, 440px);
  padding: 42px 36px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-light));
  border-radius: 20px;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
  box-shadow: var(--app-shell-panel-shadow-soft, 0 18px 48px rgba(15, 23, 42, 0.08));
  text-align: center;
}

.workspace-bootstrap__icon {
  width: 56px;
  height: 56px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 18px;
  border-radius: 18px;
  font-size: 28px;
}

.workspace-bootstrap__icon--loading {
  color: var(--el-color-primary);
  background: rgba(var(--el-color-primary-rgb), 0.12);

  .el-icon {
    animation: workspace-bootstrap-spin 1s linear infinite;
  }
}

.workspace-bootstrap__icon--error {
  color: var(--el-color-danger);
  background: rgba(var(--el-color-danger-rgb), 0.1);
}

h1 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 22px;
  line-height: 1.35;
}

p {
  margin: 12px 0 24px;
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.7;
}

@keyframes workspace-bootstrap-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
