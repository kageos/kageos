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

      <div class="overview-item">
        <div class="overview-icon-wrapper code-icon">
          <el-icon class="overview-icon"><Key /></el-icon>
        </div>
        <div class="overview-content">
          <div class="overview-label">目录代码</div>
          <div class="overview-value code-text">{{ packageNode.code }}</div>
        </div>
      </div>

      <div class="overview-divider"></div>

      <div class="overview-item">
        <div class="overview-icon-wrapper count-icon">
          <el-icon class="overview-icon"><Files /></el-icon>
        </div>
        <div class="overview-content">
          <div class="overview-label">子项数量</div>
          <div class="overview-value">{{ packageNode.children?.length || 0 }} 项</div>
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

      <div v-if="packageNode.admins && packageNode.admins.trim()" class="overview-divider"></div>

      <div v-if="packageNode.admins && packageNode.admins.trim()" class="overview-item">
        <div class="overview-icon-wrapper admins-icon">
          <el-icon class="overview-icon"><Avatar /></el-icon>
        </div>
        <div class="overview-content">
          <div class="overview-label">管理员</div>
          <div class="overview-value">
            <UsersWidget
              :field="adminsField"
              :value="adminsFieldValue"
              :field-path="adminsField.code"
              mode="detail"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Avatar, DataLine, Document, Files, Key, Star } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { WidgetType } from '@/core/constants/widget'
import UserWidget from '@/shared/components/UserWidget.vue'
import UsersWidget from '@/shared/components/UsersWidget.vue'

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

const adminsField = computed<FieldConfig>(() => ({
  code: 'admins',
  name: '管理员',
  widget: {
    type: WidgetType.USERS,
    config: {}
  }
}))

const adminsFieldValue = computed<FieldValue>(() => {
  if (!props.packageNode?.admins || !props.packageNode.admins.trim()) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }

  const admins = props.packageNode.admins
    .split(',')
    .map((s: string) => s.trim())
    .filter((s: string) => Boolean(s))

  return {
    raw: admins.join(','),
    display: admins.join(', '),
    meta: {}
  }
})
</script>

<style scoped lang="scss">
.overview-section {
  margin-bottom: 32px;
}

.overview-card {
  display: flex;
  align-items: center;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.overview-item {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 16px;
}

.overview-icon-wrapper {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 12px;
}

.overview-icon-wrapper.name-icon {
  background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));
}

.overview-icon-wrapper.name-icon .overview-icon {
  font-size: 24px;
  color: var(--el-color-primary);
}

.overview-icon-wrapper.code-icon {
  background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));
}

.overview-icon-wrapper.code-icon .overview-icon {
  font-size: 24px;
  color: var(--el-color-success);
}

.overview-icon-wrapper.count-icon {
  background: linear-gradient(135deg, var(--el-color-warning-light-8), var(--el-color-warning-light-9));
}

.overview-icon-wrapper.count-icon .overview-icon {
  font-size: 24px;
  color: var(--el-color-warning);
}

.overview-icon-wrapper.admins-icon {
  background: linear-gradient(135deg, #f3e8ff, #e9d5ff);
}

.overview-icon-wrapper.admins-icon .overview-icon {
  font-size: 24px;
  color: #9333ea;
}

.overview-icon-wrapper.owner-icon {
  background: linear-gradient(135deg, #eff6ff, #dbeafe);
}

.overview-icon-wrapper.owner-icon .overview-icon {
  font-size: 24px;
  color: #2563eb;
}

.overview-icon-wrapper.run-icon {
  background: var(--el-fill-color-light);
}

.overview-icon-wrapper.run-icon .overview-icon {
  font-size: 24px;
  color: var(--el-text-color-secondary);
}

.overview-content {
  flex: 1;
  min-width: 0;
}

.overview-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
  font-weight: 500;
}

.overview-value {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.overview-value.code-text {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  color: var(--el-color-success);
  font-size: 16px;
}

.overview-value.overview-value-run .overview-run-num,
.overview-value.overview-value-run .overview-run-unit {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.overview-value.overview-value-run .overview-run-unit {
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-left: 2px;
}

.overview-divider {
  width: 1px;
  height: 48px;
  background: var(--el-border-color-lighter);
  margin: 0 24px;
}

@media (max-width: 768px) {
  .overview-card {
    flex-direction: column;
    gap: 20px;
  }

  .overview-divider {
    width: 100%;
    height: 1px;
    margin: 0;
  }
}
</style>
