<template>
  <div v-if="children.length > 0" class="children-section">
    <div class="section-header">
      <h3 class="section-title">
        <el-icon class="section-icon"><Grid /></el-icon>
        资源列表
      </h3>
      <el-tag class="section-badge" type="primary" size="small">
        {{ children.length }}
      </el-tag>
    </div>

    <div class="children-grid">
      <div
        v-for="child in children"
        :key="child.id"
        class="child-card"
        @click="$emit('select-child', child)"
      >
        <div class="child-card-header">
          <div class="child-icon-wrapper" :class="child.type === 'package' ? 'package-type' : 'function-type'">
            <img
              v-if="child.type === 'package'"
              src="/service-tree/custom-folder.svg"
              alt="目录"
              class="child-icon-img"
            />
            <template v-else-if="child.type === 'function'">
              <img
                v-if="child.template_type === TEMPLATE_TYPE.FORM"
                src="/service-tree/编辑.svg"
                alt="表单"
                class="child-icon-img"
              />
              <el-icon v-else class="child-icon">
                <component :is="getChildFunctionIcon(child)" />
              </el-icon>
            </template>
            <img
              v-else-if="child.type === 'board'"
              src="/讨论区.svg"
              alt="讨论区"
              class="child-icon-img"
            />
            <el-icon v-else-if="child.type === 'docs'" class="child-icon">
              <Document />
            </el-icon>
            <el-icon v-else class="child-icon">
              <Document />
            </el-icon>
          </div>
          <el-tag
            v-if="child.type === 'function'"
            size="small"
            :type="getTemplateTypeTag(child.template_type)"
            class="child-type-tag"
          >
            {{ getTemplateTypeText(child.template_type) }}
          </el-tag>
          <el-tag v-else-if="child.type === 'board'" size="small" type="success" class="child-type-tag">讨论区</el-tag>
          <el-tag v-else-if="child.type === 'docs'" size="small" type="info" class="child-type-tag">文档</el-tag>
        </div>

        <div class="child-card-body">
          <div class="child-name">{{ child.name }}</div>
          <div class="child-description" v-if="child.description">
            {{ child.description }}
          </div>
        </div>

        <div v-if="child.type === 'function'" class="child-run-badge">
          <el-icon class="child-run-badge-icon"><DataLine /></el-icon>
          <span class="child-run-badge-num">{{ child.run_count ?? 0 }}</span>
        </div>
      </div>
    </div>
  </div>

  <el-empty
    v-else
    description="该目录下暂无子目录或函数"
    :image-size="120"
    class="empty-state"
  />
</template>

<script setup lang="ts">
import { DataLine, Document, Grid } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import ChartIcon from '@/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/shared/components/icons/TableIcon.vue'
import FormIcon from '@/shared/components/icons/FormIcon.vue'

defineProps<{
  children: ServiceTree[]
}>()

defineEmits<{
  'select-child': [child: ServiceTree]
}>()

function getTemplateTypeTag(templateType?: string): string {
  const typeMap: Record<string, string> = {
    table: 'success',
    form: 'primary',
    chart: 'warning'
  }
  return templateType ? (typeMap[templateType] || 'info') : 'info'
}

function getTemplateTypeText(templateType?: string): string {
  const typeMap: Record<string, string> = {
    table: '表格',
    form: '表单',
    chart: '图表'
  }
  return templateType ? (typeMap[templateType] || '函数') : '函数'
}

function getChildFunctionIcon(child: ServiceTree) {
  if (child.template_type === TEMPLATE_TYPE.TABLE) {
    return TableIcon
  }
  if (child.template_type === TEMPLATE_TYPE.FORM) {
    return FormIcon
  }
  if (child.template_type === TEMPLATE_TYPE.CHART) {
    return ChartIcon
  }
  return Document
}
</script>

<style scoped lang="scss">
.children-section {
  margin-top: 32px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.section-title {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.section-icon {
  font-size: 22px;
  color: var(--el-color-primary);
}

.section-badge {
  font-weight: 600;
  padding: 4px 12px;
}

.children-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
  width: 100%;
}

.child-card {
  position: relative;
  background: var(--app-shell-panel-bg-strong);
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 24px;
  padding: 22px 22px 50px;
  transition: all 0.3s ease;
  cursor: pointer;
  width: 100%;
  box-sizing: border-box;
  box-shadow: 0 18px 34px rgba(15, 23, 42, 0.08);
  overflow: hidden;
}

.child-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 22px;
  right: 22px;
  height: 1px;
  background: var(--app-shell-panel-highlight);
  opacity: 0.7;
  pointer-events: none;
}

.child-card:hover {
  border-color: rgba(var(--el-color-primary-rgb), 0.24);
  box-shadow: 0 24px 42px rgba(15, 23, 42, 0.12);
  transform: translateY(-3px);
}

.child-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18px;
  gap: 12px;
}

.child-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 18px;
  flex-shrink: 0;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.42);
}

.child-icon-wrapper.package-type {
  background: rgba(var(--el-color-primary-rgb), 0.12);
}

.child-icon-wrapper.package-type .child-icon-img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.child-icon-wrapper.function-type {
  background: rgba(16, 185, 129, 0.12);
}

.child-icon-wrapper.function-type .child-icon {
  font-size: 24px;
  color: var(--el-color-success);
}

.child-icon-wrapper.function-type .child-icon-img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.child-type-tag {
  font-weight: 500;
  border-radius: 999px;
  padding: 0 10px;
}

.child-name {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.5;
  word-break: break-word;
  margin-bottom: 8px;
}

.child-description {
  font-size: 13px;
  color: var(--el-text-color-regular);
  line-height: 1.6;
  word-break: break-word;
  padding-top: 12px;
  border-top: 1px solid var(--app-shell-panel-border);
}

.child-run-badge {
  position: absolute;
  bottom: 16px;
  right: 16px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-weight: 500;
  background: var(--app-shell-panel-muted-bg);
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 999px;
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
}

.child-run-badge-icon {
  font-size: 13px;
  color: var(--el-text-color-placeholder);
}

.child-run-badge-num {
  min-width: 1ch;
}

.empty-state {
  margin-top: 60px;
}

@media (max-width: 768px) {
  .children-grid {
    grid-template-columns: 1fr;
  }
}
</style>
