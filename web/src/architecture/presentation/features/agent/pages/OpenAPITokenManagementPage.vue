<template>
  <div class="openapi-token-page">
    <el-card shadow="hover" class="page-card">
      <template #header>
        <div class="card-header">
          <div>
            <h2>OpenAPI 配置</h2>
            <p class="header-desc">管理用于 OpenAPI 调用的访问 Token。</p>
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" @click="loadOpenAPITokens">刷新</el-button>
            <el-button type="primary" :icon="Plus" @click="createTokenDialogVisible = true">
              创建 Token
            </el-button>
          </div>
        </div>
      </template>

      <div class="page-body">
        <div class="toolbar">
          <div class="toolbar-summary">
            共 {{ openapiTokens.length }} 条
            <span v-if="availableTokenCount > 0">，{{ availableTokenCount }} 条可用</span>
          </div>
        </div>

        <el-table
          :data="openapiTokens"
          v-loading="tokensLoading"
          stripe
          style="width: 100%"
          empty-text="暂无 OpenAPI Token"
        >
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column prop="token_prefix" label="前缀" width="150" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.revoked_at ? 'danger' : 'success'">
                {{ row.revoked_at ? '已吊销' : '可用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="last_used_at" label="最后使用" min-width="170">
            <template #default="{ row }">
              {{ row.last_used_at || '从未使用' }}
            </template>
          </el-table-column>
          <el-table-column prop="expires_at" label="过期时间" min-width="170">
            <template #default="{ row }">
              {{ row.expires_at || '永不过期' }}
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" min-width="170" />
          <el-table-column label="操作" width="110" fixed="right">
            <template #default="{ row }">
              <el-button
                link
                type="danger"
                :disabled="!!row.revoked_at"
                @click="handleRevokeToken(row.id)"
              >
                吊销
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-card>

    <el-dialog v-model="createTokenDialogVisible" title="创建 OpenAPI Token" width="520px">
      <el-form label-width="90px">
        <el-form-item label="名称">
          <el-input v-model="newTokenName" placeholder="kageos-hub-production" />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker
            v-model="newTokenExpiresAt"
            type="datetime"
            placeholder="留空表示永不过期"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createTokenDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creatingToken" @click="handleCreateToken">
          创建
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="createdTokenVisible" title="Token 只显示一次" width="640px">
      <p class="form-tip">请现在保存这个 Token，关闭弹窗后无法再次查看明文。</p>
      <el-input
        v-model="createdSecretToken"
        readonly
        type="textarea"
        :rows="3"
        class="token-secret"
      />
      <template #footer>
        <el-button @click="copyCreatedToken">复制</el-button>
        <el-button type="primary" @click="createdTokenVisible = false">完成</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import {
  createOpenAPIToken,
  listOpenAPITokens,
  revokeOpenAPIToken,
  type OpenAPITokenInfo,
} from '@/architecture/presentation/context/api/user'

const tokensLoading = ref(false)
const creatingToken = ref(false)
const createTokenDialogVisible = ref(false)
const createdTokenVisible = ref(false)
const newTokenName = ref('')
const newTokenExpiresAt = ref<Date | null>(null)
const createdSecretToken = ref('')
const openapiTokens = ref<OpenAPITokenInfo[]>([])

const availableTokenCount = computed(() => {
  return openapiTokens.value.filter((token) => !token.revoked_at).length
})

async function loadOpenAPITokens() {
  tokensLoading.value = true
  try {
    const resp = await listOpenAPITokens()
    openapiTokens.value = resp.tokens || []
  } catch (error: any) {
    ElMessage.error(error?.message || '加载 OpenAPI Token 失败')
  } finally {
    tokensLoading.value = false
  }
}

async function handleCreateToken() {
  if (!newTokenName.value.trim()) {
    ElMessage.warning('请输入 Token 名称')
    return
  }
  creatingToken.value = true
  try {
    const resp = await createOpenAPIToken({
      name: newTokenName.value.trim(),
      expires_at: newTokenExpiresAt.value ? newTokenExpiresAt.value.toISOString() : undefined,
    })
    createdSecretToken.value = resp.secret_token
    createdTokenVisible.value = true
    createTokenDialogVisible.value = false
    newTokenName.value = ''
    newTokenExpiresAt.value = null
    await loadOpenAPITokens()
  } catch (error: any) {
    ElMessage.error(error?.message || '创建 OpenAPI Token 失败')
  } finally {
    creatingToken.value = false
  }
}

async function handleRevokeToken(id: number) {
  try {
    await ElMessageBox.confirm('吊销后使用该 Token 的服务调用会立即失效，确定继续吗？', '吊销 Token', {
      type: 'warning',
      confirmButtonText: '吊销',
      cancelButtonText: '取消',
    })
    await revokeOpenAPIToken(id)
    ElMessage.success('已吊销')
    await loadOpenAPITokens()
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error?.message || '吊销失败')
    }
  }
}

async function copyCreatedToken() {
  await navigator.clipboard.writeText(createdSecretToken.value)
  ElMessage.success('已复制')
}

onMounted(loadOpenAPITokens)
</script>

<style scoped lang="scss">
.openapi-token-page {
  padding: 20px;

  .page-card {
    border-radius: 16px;
  }

  .card-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;

    h2 {
      margin: 0;
      font-size: 24px;
      font-weight: 700;
      color: var(--el-text-color-primary);
    }

    .header-desc {
      margin: 8px 0 0;
      color: var(--el-text-color-secondary);
      font-size: 14px;
    }
  }

  .header-actions {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
  }

  .page-body {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .toolbar {
    display: flex;
    justify-content: flex-end;
  }

  .toolbar-summary {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  .form-tip {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin: 0;
  }

  .token-secret {
    margin-top: 12px;
  }
}

@media (max-width: 768px) {
  .openapi-token-page {
    padding: 12px;

    .card-header {
      flex-direction: column;
      align-items: stretch;
    }

    .header-actions {
      justify-content: flex-start;
    }

    .toolbar {
      justify-content: flex-start;
    }
  }
}
</style>
