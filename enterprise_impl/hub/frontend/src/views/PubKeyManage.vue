<template>
  <div class="pub-key-manage">
    <div class="page-header">
      <h2>发布密钥管理</h2>
      <el-button type="primary" @click="showGenerateDialog = true">+ 生成新密钥</el-button>
    </div>

    <el-alert type="info" :closable="false" style="margin-bottom: 20px">
      <template #title>发布密钥用于从其他环境跨站推送项目到本站 Hub。密钥仅在生成时显示一次，请妥善保管。</template>
    </el-alert>

    <el-card shadow="hover">
      <div v-if="loading" style="text-align: center; padding: 40px">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span style="margin-left: 8px">加载中...</span>
      </div>

      <div v-else-if="pubKeys.length === 0" style="text-align: center; padding: 40px; color: var(--el-text-color-secondary)">
        暂无密钥，点击上方按钮生成
      </div>

      <div v-else class="key-list">
        <div v-for="key in pubKeys" :key="key.id" class="key-item">
          <div class="key-main">
            <span class="key-name">{{ key.name }}</span>
            <code class="key-prefix">{{ key.key_prefix }}...</code>
          </div>
          <div class="key-meta">
            <span>创建于 {{ formatDate(key.created_at) }}</span>
            <span v-if="key.last_used_at">最后使用 {{ formatDate(key.last_used_at) }}</span>
          </div>
          <el-button type="danger" size="small" link @click="handleDelete(key)">删除</el-button>
        </div>
      </div>
    </el-card>

    <!-- 生成密钥对话框 -->
    <el-dialog v-model="showGenerateDialog" title="生成新密钥" width="480px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="密钥名称">
          <el-input v-model="newKeyName" placeholder="例如：我的笔记本、公司电脑" maxlength="50" />
        </el-form-item>
      </el-form>

      <div v-if="generatedKey" class="generated-section">
        <el-alert type="warning" :closable="false" show-icon>
          <template #title>密钥仅显示一次，请立即复制保存！</template>
        </el-alert>
        <div class="generated-box">
          <code>{{ generatedKey }}</code>
          <el-button size="small" @click="copyKey">复制</el-button>
        </div>
      </div>

      <template #footer>
        <el-button @click="closeDialog">{{ generatedKey ? '关闭' : '取消' }}</el-button>
        <el-button v-if="!generatedKey" type="primary" :loading="generating" @click="handleGenerate">生成</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { generatePubKey, listPubKeys, deletePubKey, type PubKeyItem } from '@/api/hub'

const pubKeys = ref<PubKeyItem[]>([])
const loading = ref(false)
const showGenerateDialog = ref(false)
const newKeyName = ref('')
const generatedKey = ref('')
const generating = ref(false)

async function loadKeys() {
  loading.value = true
  try {
    pubKeys.value = await listPubKeys() || []
  } catch (e) {
    console.error('加载密钥失败:', e)
  } finally {
    loading.value = false
  }
}

async function handleGenerate() {
  generating.value = true
  try {
    const resp = await generatePubKey(newKeyName.value || '默认密钥')
    generatedKey.value = resp.key
    await loadKeys()
    ElMessage.success('密钥生成成功')
  } catch (e: any) {
    ElMessage.error(e.message || '生成失败')
  } finally {
    generating.value = false
  }
}

function closeDialog() {
  showGenerateDialog.value = false
  newKeyName.value = ''
  generatedKey.value = ''
}

async function handleDelete(key: PubKeyItem) {
  try {
    await ElMessageBox.confirm(`确定删除密钥「${key.name}」吗？删除后使用该密钥的推送将不再生效。`, '删除确认', { type: 'warning' })
    await deletePubKey(key.id)
    ElMessage.success('已删除')
    await loadKeys()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.message || '删除失败')
  }
}

function copyKey() {
  navigator.clipboard.writeText(generatedKey.value)
  ElMessage.success('已复制到剪贴板')
}

function formatDate(s: string | null) {
  if (!s) return ''
  const d = new Date(s)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

onMounted(loadKeys)
</script>

<style scoped>
.pub-key-manage {
  max-width: 900px;
  margin: 0 auto;
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
}

.key-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.key-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.key-main {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.key-name {
  font-weight: 500;
}

.key-prefix {
  font-size: 12px;
  padding: 2px 8px;
  background: var(--el-fill-color);
  border-radius: 4px;
  color: var(--el-text-color-secondary);
}

.key-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.generated-section {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.generated-box {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
}

.generated-box code {
  flex: 1;
  font-size: 13px;
  word-break: break-all;
}
</style>
