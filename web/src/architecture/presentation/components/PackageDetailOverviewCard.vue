<template>
  <div v-if="packageNode" class="overview-section">
    <div class="overview-card">
      <div class="overview-item">
        <div class="overview-icon-wrapper name-icon">
          <el-icon class="overview-icon"><Document /></el-icon>
        </div>
        <div class="overview-content">
          <div class="overview-label">目录名称</div>
          <div class="overview-value">{{ packageNode.name }}</div>
        </div>
      </div>

      <div class="overview-divider"></div>

      <div class="overview-item overview-item-run">
        <div class="overview-icon-wrapper run-icon">
          <el-icon class="overview-icon"><DataLine /></el-icon>
        </div>
        <div class="overview-content">
          <div class="overview-label">本目录调用次数</div>
          <div class="overview-value overview-value-run">
            <span class="overview-run-num">{{ totalRunCount }}</span>
            <span class="overview-run-unit">次</span>
          </div>
        </div>
      </div>

      <div v-if="packageNode.owner && packageNode.owner.trim()" class="overview-divider"></div>

      <div v-if="packageNode.owner && packageNode.owner.trim()" class="overview-item">
        <div class="overview-icon-wrapper owner-icon">
          <el-icon class="overview-icon"><Star /></el-icon>
        </div>
        <div class="overview-content">
          <div class="overview-label">创建者</div>
          <div class="overview-value">
            <UserWidget
              :field="ownerField"
              :value="ownerFieldValue"
              mode="detail"
              field-path="owner"
            />
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { DataLine, Document, Star } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { WidgetType } from '@/core/constants/widget'
import UserWidget from '@/shared/components/UserWidget.vue'

const props = defineProps<{
  packageNode: ServiceTree | null
  totalRunCount: number
}>()

const ownerField = computed<FieldConfig>(() => ({
  code: 'owner',
  name: '创建者',
  widget: {
    type: WidgetType.USER,
    config: {}
  }
}))

const ownerFieldValue = computed<FieldValue>(() => {
  if (!props.packageNode?.owner || !props.packageNode.owner.trim()) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }

  return {
    raw: props.packageNode.owner.trim(),
    display: props.packageNode.owner.trim(),
    meta: {}
  }
})

</script>

<style scoped lang="scss">
.overview-section {
  margin-bottom: 28px;
}

.overview-card {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
}

.overview-item {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  min-height: 120px;
  padding: 20px 20px 18px;
  background: var(--app-shell-panel-bg-strong);
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 22px;
  box-shadow: 0 18px 34px rgba(15, 23, 42, 0.08);
  transition: transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    border-color: rgba(var(--el-color-primary-rgb), 0.22);
    box-shadow: 0 22px 40px rgba(15, 23, 42, 0.11);
  }
}

.overview-item.overview-item-run {
  background: rgba(var(--el-color-primary-rgb), 0.06);
}

.overview-icon-wrapper {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 54px;
  height: 54px;
  border-radius: 16px;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.4);
}

.overview-icon-wrapper.name-icon {
  background: rgba(var(--el-color-primary-rgb), 0.12);
}

.overview-icon-wrapper.name-icon .overview-icon {
  font-size: 24px;
  color: var(--el-color-primary);
}

.overview-icon-wrapper.admins-icon {
  background: rgba(147, 51, 234, 0.12);
}

.overview-icon-wrapper.admins-icon .overview-icon {
  font-size: 24px;
  color: #9333ea;
}

.overview-icon-wrapper.owner-icon {
  background: rgba(37, 99, 235, 0.12);
}

.overview-icon-wrapper.owner-icon .overview-icon {
  font-size: 24px;
  color: #2563eb;
}

.overview-icon-wrapper.run-icon {
  background: rgba(var(--el-color-primary-rgb), 0.12);
}

.overview-icon-wrapper.run-icon .overview-icon {
  font-size: 24px;
  color: var(--el-color-primary);
}

.overview-content {
  flex: 1;
  min-width: 0;
  padding-top: 2px;
}

.overview-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
  font-weight: 500;
  letter-spacing: 0.02em;
}

.overview-value {
  font-size: 22px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.25;
  word-break: break-word;
}

.overview-value.code-text {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  color: var(--el-color-success);
  font-size: 18px;
}

.overview-value.overview-value-run .overview-run-num,
.overview-value.overview-value-run .overview-run-unit {
  font-size: 22px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.overview-value.overview-value-run .overview-run-unit {
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-left: 4px;
}

.overview-divider {
  display: none;
}

@media (max-width: 768px) {
  .overview-card {
    grid-template-columns: 1fr;
  }
}
</style>
