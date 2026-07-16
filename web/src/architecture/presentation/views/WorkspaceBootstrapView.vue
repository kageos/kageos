<template>
  <main class="workspace-bootstrap" data-testid="workspace-bootstrap">
    <section class="workspace-bootstrap__card" aria-live="polite">
      <template v-if="loading">
        <el-icon class="workspace-bootstrap__icon is-loading"><Loading /></el-icon>
        <h1>{{ t('workspace.bootstrapLoading') }}</h1>
        <p>{{ t('workspace.bootstrapLoadingDescription') }}</p>
      </template>
      <template v-else>
        <el-icon class="workspace-bootstrap__icon is-error"><WarningFilled /></el-icon>
        <h1>{{ t('workspace.bootstrapFailedTitle') }}</h1>
        <p>{{ errorMessage || t('workspace.bootstrapFailedDescription') }}</p>
        <el-button type="primary" :icon="Refresh" @click="bootstrap">{{ t('workspace.bootstrapRetry') }}</el-button>
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
import { getErrorMessage } from '@/architecture/shared/apiError'

const router = useRouter()
const { t } = useI18n()
const loading = ref(true)
const errorMessage = ref('')
let active = true

async function bootstrap() {
  loading.value = true
  errorMessage.value = ''
  try {
    const { app } = await bootstrapPersonalWorkspace()
    if (!app?.user?.trim() || !app?.code?.trim()) throw new Error(t('workspace.bootstrapInvalidResponse'))
    if (active) await router.replace(`/workspace/${encodeURIComponent(app.user)}/${encodeURIComponent(app.code)}`)
  } catch (error) {
    if (active) errorMessage.value = getErrorMessage(error, '')
  } finally {
    if (active) loading.value = false
  }
}

onMounted(() => void bootstrap())
onBeforeUnmount(() => {
  active = false
})
</script>

<style scoped lang="scss">
.workspace-bootstrap {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 24px;
  background: var(--el-bg-color-page);
}

.workspace-bootstrap__card {
  width: min(100%, 440px);
  box-sizing: border-box;
  padding: 42px 36px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 20px;
  background: var(--el-bg-color);
  text-align: center;
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.08);
}

.workspace-bootstrap__icon {
  width: 56px;
  height: 56px;
  margin-bottom: 18px;
  font-size: 30px;
  color: var(--el-color-primary);
}

.workspace-bootstrap__icon.is-error {
  color: var(--el-color-danger);
}

h1 {
  margin: 0;
  font-size: 22px;
}

p {
  margin: 12px 0 24px;
  color: var(--el-text-color-regular);
  line-height: 1.7;
}
</style>
