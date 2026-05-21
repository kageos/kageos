<template>
  <main class="public-share-page">
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

        <div class="public-share-form">
          <FormView
            :function-detail="functionDetail"
            :form-gateway="gateway"
            :show-submit-button="true"
            :show-reset-button="true"
            :show-debug-button="false"
          />
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
} from '@/architecture/infrastructure/api/publicShare'
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
  min-height: 100vh;
  background:
    linear-gradient(
      180deg,
      color-mix(in srgb, var(--el-color-primary) 10%, var(--app-shell-bg, var(--el-bg-color-page))) 0%,
      var(--app-shell-bg, var(--el-bg-color-page)) 320px
    );
  color: var(--el-text-color-primary);
  padding: 34px 16px 48px;
}

.public-share-shell {
  width: min(1120px, 100%);
  margin: 0 auto;
  background: transparent;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  overflow: visible;
}

.public-share-state,
.public-share-result {
  padding: 48px;
}

.public-share-header {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  padding: 28px 32px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 18px;
  background: color-mix(in srgb, var(--el-color-primary) 7%, var(--app-shell-panel-muted-bg, var(--el-fill-color-light)));
  box-shadow: var(--app-shell-panel-shadow-soft, var(--box-shadow-base));
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

.public-share-form {
  padding: 22px 0 0;
  background: transparent;
}

@media (max-width: 700px) {
  .public-share-page {
    padding: 12px;
    background: var(--app-shell-bg, var(--el-bg-color-page));
  }

  .public-share-shell {
    border: none;
    border-radius: 0;
    box-shadow: none;
  }

  .public-share-header {
    display: block;
    padding: 22px 18px;
    border-radius: 14px;
  }

  .public-share-header h1 {
    font-size: 23px;
  }

  .public-share-meta {
    margin-top: 12px;
    max-width: none;
    text-align: left;
  }

  .public-share-form {
    padding: 14px 0 0;
  }

}
</style>
