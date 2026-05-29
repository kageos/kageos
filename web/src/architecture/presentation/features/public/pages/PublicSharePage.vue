<template>
  <main class="public-share-page workspace-container">
    <section class="public-share-shell">
      <div v-if="loading" class="public-share-state">
        <el-skeleton :rows="8" animated />
      </div>

      <el-result
        v-else-if="errorMessage"
        icon="warning"
        title="链接不可用"
        :sub-title="errorMessage"
        class="public-share-result"
      />

      <template v-else-if="functionDetail && gateway">
        <header class="public-share-header">
          <div>
            <p class="public-share-eyebrow">Public Form</p>
            <h1>{{ title }}</h1>
            <p v-if="description" class="public-share-description">{{ description }}</p>
          </div>
          <div v-if="metaText" class="public-share-meta">{{ metaText }}</div>
        </header>

        <div class="public-share-function-panel workspace-function-renderer public-share-renderer">
          <div class="function-runtime">
            <FormView
              :key="`public-form-${functionDetail.router}`"
              :function-detail="functionDetail"
              :form-gateway="gateway"
              :show-submit-button="true"
              :show-reset-button="true"
              :show-debug-button="false"
              response-display-mode="dialog"
            />
          </div>
        </div>
      </template>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import FormView from '@/architecture/presentation/views/FormView.vue'
import {
  PublicShareFormGateway,
  createPublicShareFunctionDetail,
  getPublicShareView,
  type PublicShareView,
} from '@/architecture/presentation/context/api/publicShare'
import { getErrorMessage } from '@/architecture/shared/apiError'
import type { FunctionDetail } from '@/architecture/domain/types'

const route = useRoute()
const loading = ref(true)
const errorMessage = ref('')
const view = ref<PublicShareView | null>(null)
const functionDetail = ref<FunctionDetail | null>(null)
const gateway = ref<PublicShareFormGateway | null>(null)

const shareId = computed(() => String(route.params.shareId || route.params.share_id || ''))
const title = computed(() => view.value?.title || '表单')
const description = computed(() => view.value?.description || '')
const metaText = computed(() => {
  const pieces: string[] = []
  if (view.value?.expires_at) {
    pieces.push(`有效期至 ${new Date(view.value.expires_at).toLocaleString()}`)
  }
  if (view.value?.remaining_uses !== undefined) {
    pieces.push(`剩余 ${view.value.remaining_uses} 次`)
  }
  return pieces.join(' · ')
})

async function loadShare() {
  loading.value = true
  errorMessage.value = ''
  try {
    const nextView = await getPublicShareView(shareId.value)
    view.value = nextView
    functionDetail.value = createPublicShareFunctionDetail(nextView)
    gateway.value = new PublicShareFormGateway(nextView.share_id)
    document.title = `${nextView.title} - Public Form`
  } catch (error) {
    errorMessage.value = getErrorMessage(error, '公开表单暂时不可用')
  } finally {
    loading.value = false
  }
}

onMounted(loadShare)
</script>

<style scoped lang="scss">
.public-share-page {
  height: 100vh;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  padding: 16px 18px 18px;
  overflow: hidden;
  background: var(--app-shell-bg, var(--el-bg-color-page));
  background-attachment: fixed;
  color: var(--el-text-color-primary);
}

.public-share-shell {
  flex: 1;
  min-height: 0;
  width: min(1180px, 100%);
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow: hidden;
}

.public-share-state,
.public-share-result {
  padding: 48px;
}

.public-share-header {
  flex: 0 0 auto;
  display: flex;
  justify-content: space-between;
  gap: 24px;
  padding: 28px 32px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 24px;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
  box-shadow: var(--app-shell-panel-shadow-soft, var(--box-shadow-base));
  position: relative;
  overflow: hidden;
}

.public-share-header::before,
.public-share-function-panel::before {
  content: '';
  position: absolute;
  top: 0;
  left: 28px;
  right: 28px;
  height: 1px;
  background: var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.7));
  opacity: 0.7;
  pointer-events: none;
}

.public-share-eyebrow {
  margin: 0 0 8px;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-color-primary);
  text-transform: uppercase;
}

.public-share-header h1 {
  margin: 0;
  font-size: 28px;
  line-height: 1.25;
  letter-spacing: 0;
  color: var(--el-text-color-primary);
}

.public-share-description {
  margin: 10px 0 0;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.public-share-meta {
  flex: 0 0 auto;
  max-width: 260px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
  text-align: right;
}

.public-share-renderer {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.public-share-function-panel {
  position: relative;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 24px;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
  box-shadow: var(--app-shell-panel-shadow, var(--box-shadow-base));
  overflow: hidden;
}

.public-share-renderer .function-runtime {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 28px;
  -webkit-overflow-scrolling: touch;
}

@media (max-width: 700px) {
  .public-share-page {
    height: auto;
    min-height: 100svh;
    align-items: center;
    padding: 18px 12px 28px;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .public-share-shell {
    width: min(460px, 100%);
    flex: 0 0 auto;
    gap: 12px;
    overflow: visible;
  }

  .public-share-header {
    display: block;
    padding: 8px 8px 4px;
    border: none;
    border-radius: 0;
    background: transparent;
    box-shadow: none;
    text-align: center;
  }

  .public-share-header::before,
  .public-share-function-panel::before {
    display: none;
  }

  .public-share-eyebrow {
    margin-bottom: 6px;
    font-size: 11px;
  }

  .public-share-header h1 {
    font-size: 22px;
    line-height: 1.35;
  }

  .public-share-description {
    margin-top: 8px;
    font-size: 14px;
    line-height: 1.55;
  }

  .public-share-meta {
    margin-top: 10px;
    max-width: none;
    text-align: center;
    font-size: 12px;
  }

  .public-share-renderer {
    min-height: 0;
  }

  .public-share-function-panel {
    border-radius: 18px;
    overflow: visible;
  }

  .public-share-renderer .function-runtime {
    overflow: visible;
    padding: 10px;
  }

  .public-share-renderer :deep(.form-view-main) {
    padding: 20px 18px 22px;
    border-radius: 16px;
  }

  .public-share-renderer :deep(.section-title) {
    margin-bottom: 18px;
    font-size: 18px;
    text-align: center;
  }

  .public-share-renderer :deep(.form-actions-row) {
    display: grid;
    grid-template-columns: 1fr;
  }

  .public-share-renderer :deep(.form-actions-row .el-button) {
    width: 100%;
    margin-left: 0;
  }
}
</style>
