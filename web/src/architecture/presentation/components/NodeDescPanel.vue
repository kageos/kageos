<!--
  NodeDescPanel - 节点说明面板
  用于文档、目录等节点，展示名称、描述、路径等
-->
<template>
  <div class="node-desc-container">
    <div class="node-desc-header">
      <span class="node-type-tag">{{ typeLabel }}</span>
      <h2 class="node-name">{{ node?.name || t('nodeDesc.untitled') }}</h2>
    </div>

    <div v-if="node?.description" class="node-desc-section">
      <h3>{{ t('nodeDesc.description') }}</h3>
      <p class="node-description">{{ node.description }}</p>
    </div>

    <div v-if="node?.full_code_path" class="node-desc-section">
      <h3>{{ t('nodeDesc.path') }}</h3>
      <code class="node-path">{{ node.full_code_path }}</code>
    </div>

    <div v-if="node?.tags && node.tags.trim()" class="node-desc-section">
      <h3>{{ t('nodeDesc.tags') }}</h3>
      <div class="node-tags">
        <el-tag
          v-for="tag in tagsList"
          :key="tag"
          size="small"
          effect="plain"
          class="tag-item"
        >
          {{ tag }}
        </el-tag>
      </div>
    </div>

    <div v-if="!node?.description && !node?.full_code_path && !(node?.tags && node.tags.trim())" class="node-desc-empty">
      <el-text type="info">{{ t('nodeDesc.empty') }}</el-text>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ServiceTree } from '@/architecture/domain/types'

interface Props {
  node: ServiceTree | null | undefined
}

const props = defineProps<Props>()
const { t } = useI18n()

const typeLabel = computed(() => {
  const type = props.node?.type
  if (type === 'docs') return t('nodeDesc.docs')
  if (type === 'package') return t('nodeDesc.directory')
  return t('nodeDesc.node')
})

const tagsList = computed(() => {
  const raw = props.node?.tags?.trim() || ''
  return raw ? raw.split(',').map((s) => s.trim()).filter(Boolean) : []
})
</script>

<style lang="scss" scoped>
.node-desc-container {
  width: 100%;
  height: 100%;
  padding: 16px;
  overflow-y: auto;
  background-color: var(--el-bg-color);
}

.node-desc-header {
  margin-bottom: 16px;

  .node-type-tag {
    display: inline-block;
    font-size: 12px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    padding: 2px 8px;
    border-radius: 4px;
    margin-bottom: 8px;
  }

  .node-name {
    font-size: 18px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin: 0;
  }
}

.node-desc-section {
  margin-bottom: 16px;

  h3 {
    font-size: 14px;
    color: var(--el-text-color-secondary);
    margin: 0 0 8px 0;
    font-weight: 500;
  }

  .node-description {
    font-size: 14px;
    line-height: 1.6;
    color: var(--el-text-color-regular);
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .node-path {
    display: block;
    font-size: 12px;
    padding: 10px 12px;
    background: var(--el-fill-color-light);
    border-radius: 6px;
    word-break: break-all;
    color: var(--el-text-color-regular);
  }

  .node-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;

    .tag-item {
      font-size: 12px;
    }
  }
}

.node-desc-empty {
  padding: 24px 0;
  text-align: center;
}
</style>
