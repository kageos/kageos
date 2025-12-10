<template>
  <el-card shadow="hover" class="knowledge-card">
    <template #header>
      <div class="knowledge-card__header">
        <div class="knowledge-card__title">
          <span class="knowledge-card__name">{{ knowledge.name }}</span>
          <el-tag 
            :type="knowledge.visibility === 0 ? 'success' : 'info'" 
            size="small" 
            style="margin-left: 8px;"
          >
            {{ knowledge.visibility === 0 ? '公开' : '私有' }}
          </el-tag>
        </div>
        <el-tag :type="knowledge.status === 'active' ? 'success' : 'info'" size="small">
          {{ knowledge.status === 'active' ? '激活' : '停用' }}
        </el-tag>
      </div>
    </template>

    <div class="knowledge-card__content">
      <!-- 文档数量 -->
      <div class="knowledge-card__info-item">
        <el-icon class="knowledge-card__info-icon"><Document /></el-icon>
        <span>文档数量：{{ knowledge.document_count || 0 }} 篇</span>
      </div>

      <el-divider />

      <!-- 使用该知识库的智能体 -->
      <div class="knowledge-card__agents">
        <div class="knowledge-card__agents-title">
          <el-icon class="knowledge-card__info-icon"><Operation /></el-icon>
          <span>使用该知识库的智能体：</span>
        </div>
        <div v-if="agentList.length > 0" class="knowledge-card__agents-list">
          <div v-for="agent in agentList" :key="agent.id" class="knowledge-card__agent-item">
            <el-icon class="knowledge-card__agent-icon">
              <CircleCheck v-if="agent.enabled" />
              <CircleClose v-else />
            </el-icon>
            <span class="knowledge-card__agent-name">{{ agent.name }}</span>
            <el-tag :type="agent.enabled ? 'success' : 'danger'" size="small">
              {{ agent.enabled ? '已启用' : '已禁用' }}
            </el-tag>
          </div>
        </div>
        <div v-else class="knowledge-card__empty">暂无智能体使用</div>
      </div>

      <el-divider v-if="knowledge.description || knowledge.admin" />

      <!-- 权限信息 -->
      <div v-if="knowledge.admin" class="knowledge-card__permission">
        <div class="knowledge-card__info-item">
          <el-icon class="knowledge-card__info-icon"><User /></el-icon>
          <span>管理员：{{ knowledge.admin }}</span>
        </div>
      </div>

      <el-divider v-if="knowledge.description" />

      <!-- 描述 -->
      <div v-if="knowledge.description" class="knowledge-card__description">
        {{ knowledge.description }}
      </div>
    </div>

    <template #footer>
      <div class="knowledge-card__footer">
        <el-button size="small" type="primary" @click="handleEnter">进入</el-button>
        <el-button 
          v-if="knowledge.is_admin" 
          size="small" 
          @click="handleEdit"
        >
          编辑
        </el-button>
        <el-button 
          v-if="knowledge.is_admin" 
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
import { Document, Operation, CircleCheck, CircleClose, User } from '@element-plus/icons-vue'
import type { KnowledgeInfo, AgentInfo } from '@/api/agent'

interface Props {
  knowledge: KnowledgeInfo
  agents?: AgentInfo[] // 使用该知识库的智能体列表
}

const props = withDefaults(defineProps<Props>(), {
  agents: () => []
})

const emit = defineEmits<{
  enter: [knowledge: KnowledgeInfo]
  edit: [knowledge: KnowledgeInfo]
  delete: [knowledge: KnowledgeInfo]
}>()

// 使用传入的智能体列表
const agentList = computed(() => props.agents || [])

function handleEnter() {
  emit('enter', props.knowledge)
}

function handleEdit() {
  emit('edit', props.knowledge)
}

function handleDelete() {
  emit('delete', props.knowledge)
}
</script>

<style scoped lang="scss">
.knowledge-card {
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
    margin-bottom: 16px;
  }
  
  &__info-icon {
    font-size: 16px;
    color: var(--el-color-primary);
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

