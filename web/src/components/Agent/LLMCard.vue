<template>
  <el-card shadow="hover" class="llm-card">
    <template #header>
      <div class="llm-card__header">
        <div class="llm-card__title">
          <span class="llm-card__name">{{ llm.name }}</span>
          <el-tag v-if="llm.is_default" type="success" size="small" style="margin-left: 8px;">默认</el-tag>
          <el-tag 
            :type="llm.visibility === 0 ? 'success' : 'info'" 
            size="small" 
            style="margin-left: 8px;"
          >
            {{ llm.visibility === 0 ? '公开' : '私有' }}
          </el-tag>
        </div>
      </div>
    </template>

    <div class="llm-card__content">
      <!-- 提供商和模型 -->
      <div class="llm-card__info-item">
        <el-icon class="llm-card__info-icon"><Shop /></el-icon>
        <span>提供商：{{ llm.provider }}</span>
      </div>
      <div class="llm-card__info-item">
        <el-icon class="llm-card__info-icon"><Cpu /></el-icon>
        <span>模型：{{ llm.model }}</span>
      </div>

      <el-divider />

      <!-- 配置信息 -->
      <div class="llm-card__config">
        <div class="llm-card__config-item">
          <span class="llm-card__config-label">API地址：</span>
          <span class="llm-card__config-value">{{ llm.api_base || '-' }}</span>
        </div>
        <div class="llm-card__config-item">
          <span class="llm-card__config-label">超时：</span>
          <span class="llm-card__config-value">{{ llm.timeout }}秒</span>
        </div>
        <div class="llm-card__config-item">
          <span class="llm-card__config-label">最大Token：</span>
          <span class="llm-card__config-value">{{ llm.max_tokens }}</span>
        </div>
      </div>

      <el-divider />

      <!-- 权限信息 -->
      <div v-if="llm.admin" class="llm-card__permission">
        <div class="llm-card__info-item">
          <el-icon class="llm-card__info-icon"><User /></el-icon>
          <span>管理员：{{ llm.admin }}</span>
        </div>
      </div>

      <el-divider />

      <!-- 使用该LLM的智能体 -->
      <div class="llm-card__agents">
        <div class="llm-card__agents-title">
          <el-icon class="llm-card__info-icon"><Operation /></el-icon>
          <span>使用该LLM的智能体：</span>
        </div>
        <div v-if="agentList.length > 0" class="llm-card__agents-list">
          <div v-for="agent in agentList" :key="agent.id" class="llm-card__agent-item">
            <el-icon class="llm-card__agent-icon">
              <CircleCheck v-if="agent.enabled" />
              <CircleClose v-else />
            </el-icon>
            <span class="llm-card__agent-name">{{ agent.name }}</span>
            <el-tag :type="agent.enabled ? 'success' : 'danger'" size="small">
              {{ agent.enabled ? '已启用' : '已禁用' }}
            </el-tag>
          </div>
        </div>
        <div v-else class="llm-card__empty">暂无智能体使用</div>
      </div>
    </div>

    <template #footer>
      <div class="llm-card__footer">
        <el-button size="small" type="info" @click="handleDetail">详情</el-button>
        <el-button 
          v-if="llm.is_admin" 
          size="small" 
          @click="handleEdit"
        >
          编辑
        </el-button>
        <el-button
          v-if="llm.is_admin && !llm.is_default"
          size="small"
          type="primary"
          @click="handleSetDefault"
        >
          设为默认
        </el-button>
        <el-button 
          size="small" 
          type="info"
          @click="handleCopy"
        >
          复制
        </el-button>
        <el-button 
          v-if="llm.is_admin" 
          size="small" 
          type="danger" 
          @click="handleDelete"
        >
          删除
        </el-button>
      </div>
    </template>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Shop, Cpu, Operation, CircleCheck, CircleClose, User } from '@element-plus/icons-vue'
import type { LLMInfo, AgentInfo } from '@/api/agent'

interface Props {
  llm: LLMInfo
  agents?: AgentInfo[] // 使用该LLM的智能体列表
}

const props = withDefaults(defineProps<Props>(), {
  agents: () => []
})

const emit = defineEmits<{
  detail: [llm: LLMInfo]
  edit: [llm: LLMInfo]
  setDefault: [llm: LLMInfo]
  delete: [llm: LLMInfo]
}>()

// 使用传入的智能体列表
const agentList = computed(() => props.agents || [])

function handleDetail() {
  emit('detail', props.llm)
}

function handleEdit() {
  emit('edit', props.llm)
}

function handleSetDefault() {
  emit('setDefault', props.llm)
}

function handleDelete() {
  emit('delete', props.llm)
}

function handleCopy() {
  emit('copy', props.llm)
}
</script>

<style scoped lang="scss">
.llm-card {
  transition: all 0.3s;
  border: 1px solid var(--el-border-color-lighter);
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  }
  
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  
  &__title {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  &__name {
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  
  &__content {
    min-height: 200px;
  }
  
  &__info-item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    color: var(--el-text-color-regular);
    margin-bottom: 12px;
  }
  
  &__info-icon {
    font-size: 16px;
    color: var(--el-color-primary);
  }
  
  &__config {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 16px;
  }
  
  &__config-item {
    display: flex;
    font-size: 14px;
    color: var(--el-text-color-regular);
  }
  
  &__config-label {
    min-width: 80px;
    color: var(--el-text-color-secondary);
  }
  
  &__config-value {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--el-text-color-primary);
  }
  
  &__agents {
    margin-bottom: 16px;
  }
  
  &__agents-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    font-weight: 500;
    color: var(--el-text-color-primary);
    margin-bottom: 12px;
  }
  
  &__agents-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  &__agent-item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    padding: 4px 8px;
    background: var(--el-fill-color-lighter);
    border-radius: 4px;
  }
  
  &__agent-icon {
    font-size: 14px;
  }
  
  &__agent-name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--el-text-color-regular);
  }
  
  &__empty {
    font-size: 14px;
    color: var(--el-text-color-placeholder);
    text-align: center;
    padding: 16px 0;
  }
  
  &__footer {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }
}
</style>

