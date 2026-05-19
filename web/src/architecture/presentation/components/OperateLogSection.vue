<template>
  <div class="operate-log-section" :class="{ 'is-embedded': embedded }">
    <el-divider v-if="!embedded" />
    <div class="operate-log-header">
      <div class="operate-log-title-group">
        <el-icon class="operate-log-icon"><Clock /></el-icon>
        <span class="operate-log-title">{{ title }}</span>
      </div>
      <el-button
        v-if="showRefresh"
        size="small"
        :icon="Refresh"
        :loading="loading"
        @click="load"
      >
        刷新
      </el-button>
    </div>
    <div v-loading="loading" class="operate-log-content">
      <div v-if="logs.length > 0" class="operate-log-timeline">
        <div
          v-for="log in logs"
          :key="log.id"
          class="log-card"
        >
          <div class="log-card-main">
            <div class="log-card-marker">
              <span class="marker-dot" :class="`is-${getActionTagType(log.action)}`" />
            </div>
            <div class="log-card-body">
              <div class="log-card-head">
                <div class="log-title-wrap">
                  <el-tag :type="getActionTagType(log.action)" size="small" effect="light">
                    {{ getActionLabel(log.action) }}
                  </el-tag>
                  <span class="log-title">{{ getLogTitle(log) }}</span>
                </div>
                <div class="log-time">
                  <span>{{ formatRelativeTime(log.created_at) }}</span>
                  <span>{{ formatDateTime(log.created_at) }}</span>
                </div>
              </div>

              <div class="log-actor-row">
                <UserDisplay
                  :user-info="getUserInfo(log.request_user)"
                  :username="log.request_user"
                  mode="simple"
                  layout="horizontal"
                  size="small"
                />
                <span v-if="showRowIdColumn && log.row_id" class="log-chip">记录 #{{ log.row_id }}</span>
                <span v-if="log.version" class="log-chip">版本 {{ log.version }}</span>
              </div>

              <div v-if="showResourceColumn" class="log-resource">
                {{ log.full_code_path || '-' }}
              </div>

              <div
                v-if="log.action === 'OnTableUpdateRow' && getChangeEntries(log).length > 0"
                class="change-list"
              >
                <div
                  v-for="item in getChangeEntries(log)"
                  :key="item.fieldCode"
                  class="change-row"
                >
                  <span class="change-field">{{ item.fieldName }}</span>
                  <div class="change-values">
                    <span v-if="item.hasOldValue" class="change-old">{{ formatLogValue(item.oldValue) }}</span>
                    <span v-else class="change-old is-empty">未记录旧值</span>
                    <span class="change-arrow">→</span>
                    <span class="change-new">{{ formatLogValue(item.newValue) }}</span>
                  </div>
                </div>
              </div>

              <div
                v-else-if="getValueEntries(log).length > 0"
                class="value-list"
              >
                <div
                  v-for="item in getValueEntries(log)"
                  :key="item.fieldCode"
                  class="value-row"
                >
                  <span class="value-field">{{ item.fieldName }}</span>
                  <span class="value-text">{{ formatLogValue(item.value) }}</span>
                </div>
              </div>

              <div v-else class="text-muted">{{ getLogEmptyText(log) }}</div>

              <div v-if="log.trace_id" class="log-meta">
                Trace: {{ log.trace_id }}
              </div>
            </div>
          </div>
        </div>
      </div>
      <el-empty v-else description="暂无操作日志" :image-size="80" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { toRef } from 'vue'
import { Clock, Refresh } from '@element-plus/icons-vue'
import { ElButton, ElDivider, ElEmpty, ElIcon, ElTag } from 'element-plus'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'
import { useOperateLogSection } from '@/architecture/presentation/composables/useOperateLogSection'

interface Props {
  fullCodePath: string
  rowId: number
  functionDetail?: any
  autoLoad?: boolean
  scope?: 'row' | 'function' | 'directory'
  embedded?: boolean
  showRefresh?: boolean
  title?: string
}

const props = withDefaults(defineProps<Props>(), {
  fullCodePath: '',
  rowId: 0,
  functionDetail: undefined,
  autoLoad: false,
  scope: 'row',
  embedded: false,
  showRefresh: false,
  title: '操作日志'
})

const {
  logs,
  loading,
  formatDateTime,
  formatRelativeTime,
  getUserInfo,
  getActionTagType,
  getActionLabel,
  formatLogValue,
  getChangeEntries,
  getValueEntries,
  getLogTitle,
  getLogEmptyText,
  load,
  showRowIdColumn,
  showResourceColumn,
} = useOperateLogSection({
  fullCodePath: toRef(props, 'fullCodePath'),
  rowId: toRef(props, 'rowId'),
  functionDetail: toRef(props, 'functionDetail'),
  autoLoad: toRef(props, 'autoLoad'),
  scope: toRef(props, 'scope'),
})

defineExpose({
  load
})
</script>

<style scoped>
.operate-log-section {
  margin-top: 24px;
}

.operate-log-section.is-embedded {
  height: 100%;
  margin-top: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.operate-log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.operate-log-title-group {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
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
  min-height: 0;
}

.operate-log-section.is-embedded .operate-log-content {
  flex: 1;
  overflow: auto;
  margin-top: 0;
}

.operate-log-timeline {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.log-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.04);
}

.log-card-main {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 10px;
  padding: 14px 16px 16px 12px;
}

.log-card-marker {
  position: relative;
  display: flex;
  justify-content: center;
  padding-top: 5px;
}

.marker-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--el-color-info);
  box-shadow: 0 0 0 4px var(--el-fill-color-light);
}

.marker-dot.is-success {
  background: var(--el-color-success);
  box-shadow: 0 0 0 4px rgba(103, 194, 58, 0.12);
}

.marker-dot.is-warning {
  background: var(--el-color-warning);
  box-shadow: 0 0 0 4px rgba(230, 162, 60, 0.14);
}

.marker-dot.is-danger {
  background: var(--el-color-danger);
  box-shadow: 0 0 0 4px rgba(245, 108, 108, 0.13);
}

.log-card-body {
  min-width: 0;
}

.log-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
}

.log-title-wrap {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.log-title {
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.45;
}

.log-time {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  line-height: 1.35;
  white-space: nowrap;
}

.log-time span:first-child {
  color: var(--el-text-color-secondary);
  font-weight: 600;
}

.log-actor-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.log-chip {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.log-resource {
  margin-top: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-secondary);
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.45;
  word-break: break-all;
}

.change-list,
.value-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
}

.change-row,
.value-row {
  display: grid;
  grid-template-columns: minmax(96px, 160px) minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  min-width: 0;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  background: var(--el-fill-color-blank);
}

.change-field,
.value-field {
  color: var(--el-text-color-regular);
  font-size: 13px;
  font-weight: 700;
  line-height: 24px;
  word-break: break-word;
}

.change-values {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 13px;
  line-height: 24px;
}

.change-old,
.change-new,
.value-text {
  min-width: 0;
  max-width: 100%;
  word-break: break-word;
}

.change-old {
  color: var(--el-text-color-secondary);
  text-decoration: line-through;
  text-decoration-thickness: 1px;
  text-decoration-color: rgba(245, 108, 108, 0.5);
}

.change-old.is-empty {
  text-decoration: none;
  color: var(--el-text-color-placeholder);
}

.change-arrow {
  color: var(--el-text-color-placeholder);
  font-weight: 700;
}

.change-new {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.value-text {
  color: var(--el-text-color-primary);
  font-size: 13px;
  line-height: 24px;
}

.log-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 12px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.text-muted {
  display: block;
  margin-top: 12px;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

@media (max-width: 720px) {
  .log-card-head {
    flex-direction: column;
  }

  .log-time {
    align-items: flex-start;
  }

  .change-row,
  .value-row {
    grid-template-columns: 1fr;
  }
}
</style>
