<template>
  <el-card shadow="hover" class="agent-card">
    <template #header>
      <div class="agent-card__header">
        <div class="agent-card__title">
          <el-tag :type="agentTypeTagType" size="small">{{ agentTypeLabel }}</el-tag>
          <span class="agent-card__name">{{ agent.name }}</span>
          <el-tag 
            :type="agent.visibility === 0 ? 'success' : 'info'" 
            size="small" 
            style="margin-left: 8px;"
          >
            {{ agent.visibility === 0 ? '公开' : '私有' }}
          </el-tag>
        </div>
        <el-tag :type="agent.enabled ? 'success' : 'danger'" size="small">
          {{ agent.enabled ? '已启用' : '已禁用' }}
        </el-tag>
      </div>
    </template>

    <div class="agent-card__content">
      <!-- 关联信息 -->
      <div class="agent-card__relations">
        <div class="agent-card__relation-item">
          <el-icon class="agent-card__relation-icon"><Document /></el-icon>
          <span class="agent-card__relation-label">知识库：</span>
          <span class="agent-card__relation-value">{{ knowledgeBaseName }}</span>
        </div>
        <div v-if="agent.agent_type === 'plugin'" class="agent-card__relation-item">
          <el-icon class="agent-card__relation-icon"><Connection /></el-icon>
          <span class="agent-card__relation-label">插件：</span>
          <span class="agent-card__relation-value">{{ pluginName || '未关联' }}</span>
        </div>
        <div class="agent-card__relation-item">
          <el-icon class="agent-card__relation-icon"><Cpu /></el-icon>
          <span class="agent-card__relation-label">LLM：</span>
          <span class="agent-card__relation-value">{{ llmName }}</span>
        </div>
      </div>

      <el-divider />

      <!-- 其他信息 -->
      <div class="agent-card__info">
        <div class="agent-card__info-item">
          <el-icon class="agent-card__info-icon"><ChatDotRound /></el-icon>
          <span>聊天类型：{{ agent.chat_type }}</span>
        </div>
        <div class="agent-card__info-item">
          <el-icon class="agent-card__info-icon"><Timer /></el-icon>
          <span>超时：{{ agent.timeout }}秒</span>
        </div>
      </div>

      <el-divider v-if="agent.description || agent.admin" />

      <!-- 权限信息 -->
      <div v-if="agent.admin" class="agent-card__permission">
        <div class="agent-card__info-item">
          <el-icon class="agent-card__info-icon"><User /></el-icon>
          <span>管理员：{{ agent.admin }}</span>
        </div>
      </div>

      <el-divider v-if="agent.description" />

      <!-- 描述 -->
      <div v-if="agent.description" class="agent-card__description">
        {{ agent.description }}
      </div>
    </div>

    <template #footer>
      <div class="agent-card__footer">
        <el-button size="small" type="info" @click="handleDetail">详情</el-button>
        <el-button 
          v-if="agent.is_admin" 
          size="small" 
          @click="handleEdit"
        >
          编辑
        </el-button>
        <el-button
          v-if="agent.is_admin && agent.enabled"
          size="small"
          type="warning"
          @click="handleToggle"
        >
          禁用
        </el-button>
        <el-button
          v-if="agent.is_admin && !agent.enabled"
          size="small"
          type="success"
          @click="handleToggle"
        >
          启用
        </el-button>
        <el-button 
          v-if="agent.is_admin" 
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
import { Document, Connection, Cpu, ChatDotRound, Timer, User } from '@element-plus/icons-vue'
import type { AgentInfo } from '@/api/agent'

interface Props {
  agent: AgentInfo
}

const props = defineProps<Props>()

const emit = defineEmits<{
  detail: [agent: AgentInfo]
  edit: [agent: AgentInfo]
  toggle: [agent: AgentInfo]
  delete: [agent: AgentInfo]
}>()

const agentTypeLabel = computed(() => {
  return props.agent.agent_type === 'plugin' ? '插件类型' : '纯知识库'
})

const agentTypeTagType = computed(() => {
  return props.agent.agent_type === 'plugin' ? 'warning' : 'success'
})

const knowledgeBaseName = computed(() => {
  return props.agent.knowledge_base?.name || `ID: ${props.agent.knowledge_base_id}`
})

const pluginName = computed(() => {
  return props.agent.plugin?.name || '未关联'
})

const llmName = computed(() => {
  if (props.agent.llm_config) {
    return `${props.agent.llm_config.name}${props.agent.llm_config.is_default ? ' (默认)' : ''}`
  }
  return props.agent.llm_config_id === 0 ? '默认LLM' : `ID: ${props.agent.llm_config_id}`
})

function handleDetail() {
  emit('detail', props.agent)
}

function handleEdit() {
  emit('edit', props.agent)
}

function handleToggle() {
  emit('toggle', props.agent)
}

function handleDelete() {
  emit('delete', props.agent)
}
</script>

<style scoped lang="scss">
.agent-card {
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
  
  &__relations {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-bottom: 16px;
  }
  
  &__relation-item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
  }
  
  &__relation-icon {
    font-size: 16px;
    color: var(--el-color-primary);
  }
  
  &__relation-label {
    color: var(--el-text-color-regular);
    min-width: 60px;
  }
  
  &__relation-value {
    color: var(--el-text-color-primary);
    font-weight: 500;
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  
  &__info {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 16px;
  }
  
  &__info-item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    color: var(--el-text-color-regular);
  }
  
  &__info-icon {
    font-size: 14px;
    color: var(--el-text-color-secondary);
  }
  
  &__description {
    font-size: 14px;
    color: var(--el-text-color-regular);
    line-height: 1.6;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  
  &__footer {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }
}
</style>

