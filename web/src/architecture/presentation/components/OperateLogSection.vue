<template>
  <div class="operate-log-section">
    <el-divider />
    <div class="operate-log-header">
      <el-icon class="operate-log-icon"><Clock /></el-icon>
      <span class="operate-log-title">操作日志</span>
    </div>
    <div v-loading="loading" class="operate-log-content">
      <el-table
        v-if="logs.length > 0"
        :data="logs"
        size="small"
        class="operate-log-table"
      >
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="log-expand">
              <div v-if="row.action === 'OnTableUpdateRow' && row.updates" class="update-list">
                <div v-for="(value, key) in parseJSON(row.updates)" :key="key" class="update-row">
                  <div class="update-field">{{ getFieldName(key) }}</div>
                  <div class="update-values">
                    <div class="update-value update-value-new">
                      <span class="value-label">更新后</span>
                      <div class="value-content">
                        <component :is="renderFieldValue(key, value)" />
                      </div>
                    </div>
                    <div
                      v-if="row.old_values && parseJSON(row.old_values)[key] !== undefined"
                      class="update-value update-value-old"
                    >
                      <span class="value-label">更新前</span>
                      <div class="value-content">
                        <component :is="renderFieldValue(key, parseJSON(row.old_values)[key])" />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <span v-else class="text-muted">无字段变更详情</span>
              <div class="log-meta">
                <span v-if="row.trace_id">Trace: {{ row.trace_id }}</span>
                <span v-if="row.version">版本: {{ row.version }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="92">
          <template #default="{ row }">
            <el-tag :type="getActionTagType(row.action)" size="small">
              {{ getActionLabel(row.action) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作人" min-width="160">
          <template #default="{ row }">
            <UserDisplay
              :user-info="getUserInfo(row.request_user)"
              :username="row.request_user"
              mode="simple"
              layout="horizontal"
              size="small"
            />
          </template>
        </el-table-column>
        <el-table-column label="时间" min-width="190">
          <template #default="{ row }">
            <div class="time-cell">
              <span>{{ formatRelativeTime(row.created_at) }}</span>
              <span>{{ formatDateTime(row.created_at) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="行 ID" prop="row_id" width="100" />
      </el-table>
      <el-empty v-else description="暂无操作日志" :image-size="80" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { toRef } from 'vue'
import { Clock } from '@element-plus/icons-vue'
import { ElDivider, ElEmpty, ElIcon, ElTable, ElTableColumn, ElTag } from 'element-plus'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'
import { useOperateLogSection } from '@/architecture/presentation/composables/useOperateLogSection'

interface Props {
  fullCodePath: string
  rowId: number
  functionDetail?: any
  autoLoad?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  fullCodePath: '',
  rowId: 0,
  functionDetail: undefined,
  autoLoad: false
})

const {
  logs,
  loading,
  formatDateTime,
  formatRelativeTime,
  getUserInfo,
  getActionTagType,
  getActionLabel,
  parseJSON,
  getFieldName,
  renderFieldValue,
  load,
} = useOperateLogSection({
  fullCodePath: toRef(props, 'fullCodePath'),
  rowId: toRef(props, 'rowId'),
  functionDetail: toRef(props, 'functionDetail'),
  autoLoad: toRef(props, 'autoLoad'),
})

defineExpose({
  load
})
</script>

<style scoped>
.operate-log-section {
  margin-top: 24px;
}

.operate-log-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.operate-log-icon {
  font-size: 18px;
  color: var(--el-color-primary);
}

.operate-log-title {
  flex: 1;
}

.operate-log-content {
  margin-top: 12px;
}

.operate-log-table {
  width: 100%;
}

.log-expand {
  padding: 10px 16px 14px;
}

.update-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.update-row {
  display: grid;
  grid-template-columns: minmax(120px, 180px) minmax(0, 1fr);
  gap: 12px;
  align-items: start;
}

.update-field {
  color: var(--el-text-color-regular);
  font-size: 13px;
  font-weight: 600;
  line-height: 28px;
}

.update-values {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 8px;
  min-width: 0;
}

.update-value {
  min-width: 0;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
}

.update-value-new {
  border-color: rgba(103, 194, 58, 0.22);
  background-color: rgba(103, 194, 58, 0.06);
}

.update-value-old {
  border-color: rgba(245, 108, 108, 0.18);
  background-color: rgba(245, 108, 108, 0.05);
}

.value-label {
  display: block;
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 500;
}

.value-content {
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 13px;
  word-break: break-word;
}

.time-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  color: var(--el-text-color-regular);
  font-size: 13px;
}

.time-cell span + span {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.log-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 12px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.text-fallback {
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.text-muted {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

@media (max-width: 720px) {
  .update-row {
    grid-template-columns: 1fr;
  }
}
</style>
