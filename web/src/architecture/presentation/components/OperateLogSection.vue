<template>
  <div class="operate-log-section">
    <!-- 企业版：显示操作日志 -->
    <template v-if="hasOperateLog">
      <el-divider />
      <div class="operate-log-header">
        <el-icon class="operate-log-icon"><Clock /></el-icon>
        <span class="operate-log-title">操作日志</span>
      </div>
      <div v-loading="loading" class="operate-log-content">
        <!-- 🔥 卡片列表形式，不再使用表格 -->
        <div v-if="logs.length > 0" class="operate-log-cards">
          <div v-for="(log, index) in logs" :key="index" class="operate-log-card">
            <!-- 卡片头部：操作类型、操作人、操作时间 -->
            <div class="card-header">
              <div class="card-header-left">
                <el-tag :type="getActionTagType(log.action)" size="small" class="action-tag">
                  {{ getActionLabel(log.action) }}
                </el-tag>
                <UserDisplay
                  :user-info="getUserInfo(log.request_user)"
                  :username="log.request_user"
                  mode="card"
                  layout="horizontal"
                  size="small"
                  class="user-display"
                />
              </div>
              <div class="card-header-right">
                <div class="card-time-wrapper">
                  <span class="card-time-relative">{{ formatRelativeTime(log.created_at) }}</span>
                  <span class="card-time-absolute">{{ formatDateTime(log.created_at) }}</span>
                </div>
              </div>
            </div>
            
            <!-- 卡片内容：变更内容 -->
            <div class="card-body">
              <div v-if="log.action === 'OnTableUpdateRow' && log.updates" class="update-content">
                <div v-for="(value, key) in parseJSON(log.updates)" :key="key" class="update-item">
                  <!-- 🔥 上中下布局：上面字段名称，中间组件，下面时间和用户 -->
                  <div class="update-item-vertical">
                    <!-- 上面：字段名称 -->
                    <div class="update-field-label-top">{{ getFieldName(key) }}</div>
                    
                    <!-- 中间：组件（更新后和更新前） -->
                    <div class="update-values-middle">
                      <!-- 更新后的值 -->
                      <div class="update-value-new">
                        <div class="value-label">更新后</div>
                        <div class="value-content">
                          <component :is="renderFieldValue(key, value)" />
                        </div>
                      </div>
                      <!-- 更新前的值 -->
                      <div v-if="log.old_values && parseJSON(log.old_values)[key] !== undefined" class="update-value-old">
                        <div class="value-label">更新前</div>
                        <div class="value-content">
                          <component :is="renderFieldValue(key, parseJSON(log.old_values)[key])" />
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div v-else class="no-updates">
                <span class="text-muted">-</span>
              </div>
            </div>
          </div>
        </div>
        <el-empty v-else description="暂无操作日志" :image-size="80" />
      </div>
    </template>

    <!-- 非企业版：显示升级提示 -->
    <template v-else>
      <el-divider />
      <el-card shadow="never" class="upgrade-card">
        <div class="upgrade-content">
          <el-icon class="upgrade-icon"><Clock /></el-icon>
          <div class="upgrade-text">
            <div class="upgrade-title">操作日志功能</div>
            <div class="upgrade-desc">升级到企业版即可查看完整的操作日志记录</div>
          </div>
          <el-button type="primary" size="small" @click="handleUpgrade">
            升级企业版
          </el-button>
        </div>
      </el-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { toRef } from 'vue'
import { Clock } from '@element-plus/icons-vue'
import { ElIcon, ElEmpty, ElTag, ElCard, ElDivider, ElButton } from 'element-plus'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import { useOperateLogSection } from '@/architecture/presentation/composables/useOperateLogSection'

interface Props {
  /** 完整代码路径 */
  fullCodePath: string
  /** 记录ID */
  rowId: number
  /** 函数详情（用于获取字段名称和渲染组件） */
  functionDetail?: any
  /** 是否自动加载（默认 false，需要手动调用 load 方法） */
  autoLoad?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  fullCodePath: '',
  rowId: 0,
  functionDetail: undefined,
  autoLoad: false
})
const {
  hasOperateLog,
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
  handleUpgrade,
  load,
} = useOperateLogSection({
  fullCodePath: toRef(props, 'fullCodePath'),
  rowId: toRef(props, 'rowId'),
  functionDetail: toRef(props, 'functionDetail'),
  autoLoad: toRef(props, 'autoLoad'),
})

// 暴露 load 方法供外部调用
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

/* 🔥 卡片列表样式 */
.operate-log-cards {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.operate-log-card {
  background-color: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.2s ease;
}

.operate-log-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  border-color: var(--el-border-color);
}

/* 卡片头部 */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.card-header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.action-tag {
  flex-shrink: 0;
}

.user-display {
  flex: 1;
  min-width: 0;
}

.card-header-right {
  flex-shrink: 0;
  margin-left: 12px;
}

.card-time-wrapper {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.card-time-relative {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-color-primary);
  white-space: nowrap;
}

.card-time-absolute {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  white-space: nowrap;
}

/* 卡片内容 */
.card-body {
  width: 100%;
}

.no-updates {
  padding: 8px 0;
  text-align: center;
}

.update-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.update-item {
  padding: 12px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 6px;
  margin-bottom: 8px;
  border: 1px solid var(--el-border-color-lighter);
}

/* 🔥 上中下布局 */
.update-item-vertical {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}

/* 上面：字段名称 */
.update-field-label-top {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  padding-bottom: 6px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

/* 中间：组件（新值和原值） */
.update-values-middle {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  width: 100%;
}

.update-value-new {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 4px;
  /* 🔥 新值：微微的绿色背景 */
  background-color: rgba(103, 194, 58, 0.08);
  border: 1px solid rgba(103, 194, 58, 0.2);
  width: 100%;
  box-sizing: border-box;
}

.update-value-old {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 4px;
  /* 🔥 旧值：微微的红色背景 */
  background-color: rgba(245, 108, 108, 0.08);
  border: 1px solid rgba(245, 108, 108, 0.2);
  width: 100%;
  box-sizing: border-box;
}

.value-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.value-content {
  flex: 1;
  min-width: 0;
  width: 100%;
  font-size: 13px;
  word-break: break-word;
}

/* 下面：时间和操作用户 */
.update-meta-bottom {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 6px;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.update-time {
  white-space: nowrap;
}

.update-separator {
  color: var(--el-text-color-placeholder);
}

.update-user {
  flex: 1;
  min-width: 0;
}

.text-fallback {
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.text-muted {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

/* 升级提示卡片 */
.upgrade-card {
  border: 1px solid var(--el-border-color-light);
  background-color: var(--el-fill-color-lighter);
}

.upgrade-content {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 8px 0;
}

.upgrade-icon {
  font-size: 24px;
  color: var(--el-color-primary);
}

.upgrade-text {
  flex: 1;
}

.upgrade-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}

.upgrade-desc {
  font-size: 12px;
  color: var(--el-text-color-regular);
}
</style>
