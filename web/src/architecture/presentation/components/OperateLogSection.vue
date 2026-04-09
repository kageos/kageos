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
import { ref, computed, watch, h } from 'vue'
import { Clock } from '@element-plus/icons-vue'
import { ElIcon, ElEmpty, ElTag, ElCard, ElDivider, ElButton, ElMessage } from 'element-plus'
import type { TagProps } from 'element-plus'
import { formatTimestamp } from '@/utils/date'
import { useLicenseStore } from '@/stores/license'
import { useUserInfoStore } from '@/stores/userInfo'
import { getTableOperateLogs, type TableOperateLog } from '@/api/operateLog'
import { widgetComponentFactory } from '@/architecture/infrastructure/widgetRegistry'
import { convertToFieldValue } from '@/utils/field'
import type { FieldConfig } from '@/types'
import { getFunctionByPath } from '@/api/function'
import type { FunctionDetail } from '@/architecture/domain/types'
import UserDisplay from '@/shared/components/UserDisplay.vue'

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
const userInfoStore = useUserInfoStore()

/**
 * 格式化日期时间（支持字符串和时间戳）
 */
const formatDateTime = (dateTime: string | number | null | undefined): string => {
  if (!dateTime) return '-'
  
  // 如果是字符串格式（如 "2025-12-13 23:39:16"），直接返回
  if (typeof dateTime === 'string') {
    // 检查是否是时间戳字符串
    if (/^\d+$/.test(dateTime)) {
      // 是时间戳字符串，转换为数字
      return formatTimestamp(Number(dateTime))
    }
    // 是日期时间字符串，直接返回
    return dateTime
  }
  
  // 如果是数字（时间戳），使用 formatTimestamp
  return formatTimestamp(dateTime)
}

/**
 * 格式化相对时间（如：5分钟前、2天前）
 */
const formatRelativeTime = (dateTime: string | number | null | undefined): string => {
  if (!dateTime) return '-'
  
  // 转换为时间戳（毫秒）
  let timestamp: number
  if (typeof dateTime === 'string') {
    // 检查是否是时间戳字符串
    if (/^\d+$/.test(dateTime)) {
      timestamp = Number(dateTime)
    } else {
      // 是日期时间字符串，转换为时间戳
      timestamp = new Date(dateTime).getTime()
    }
  } else {
    timestamp = dateTime
  }
  
  // 检查时间戳是否有效
  if (isNaN(timestamp)) {
    return '-'
  }
  
  const now = Date.now()
  const diff = now - timestamp
  
  // 如果时间在未来，返回绝对时间
  if (diff < 0) {
    return formatDateTime(dateTime)
  }
  
  // 计算时间差
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  const months = Math.floor(days / 30)
  const years = Math.floor(days / 365)
  
  // 根据时间差返回相对时间
  if (seconds < 60) {
    return '刚刚'
  } else if (minutes < 60) {
    return `${minutes}分钟前`
  } else if (hours < 24) {
    return `${hours}小时前`
  } else if (days < 30) {
    return `${days}天前`
  } else if (months < 12) {
    return `${months}个月前`
  } else {
    return `${years}年前`
  }
}

const licenseStore = useLicenseStore()
const logs = ref<TableOperateLog[]>([])
const loading = ref(false)
const functionDetailCache = ref<FunctionDetail | null>(null)
const userInfoMap = ref<Map<string, any>>(new Map()) // Cache for user info

/** 是否支持操作日志功能 */
const hasOperateLog = computed(() => licenseStore.hasOperateLog)

/**
 * 加载函数详情（如果 functionDetail 没有 response，则根据 fullCodePath 加载）
 */
const loadFunctionDetail = async () => {
  // 如果已经有 functionDetail 且包含 response，直接返回
  if (props.functionDetail) {
    const hasResponse = Array.isArray(props.functionDetail.response) && props.functionDetail.response.length > 0
    if (hasResponse) {
      functionDetailCache.value = props.functionDetail as FunctionDetail
      return
    }
  }
  
  // 如果没有 functionDetail 或没有 response，根据 fullCodePath 加载
  if (props.fullCodePath && !functionDetailCache.value) {
    try {
      const detail = await getFunctionByPath(props.fullCodePath)
      if (detail && Array.isArray(detail.response) && detail.response.length > 0) {
        functionDetailCache.value = detail as unknown as FunctionDetail
      }
    } catch (error) {
      console.warn('[OperateLogSection] 加载函数详情失败:', error)
    }
  }
}

/**
 * 加载操作日志
 */
const loadOperateLogs = async () => {
  // 只有企业版且支持操作日志功能时才加载
  if (!hasOperateLog.value) {
    return
  }

  if (!props.fullCodePath || !props.rowId) {
    return
  }

  loading.value = true
  try {
    // 先加载函数详情（如果还没有）
    await loadFunctionDetail()
    
    // 然后加载操作日志
    const response = await getTableOperateLogs({
      full_code_path: props.fullCodePath,
      row_id: props.rowId,
      page: 1,
      page_size: 50,
      order_by: 'created_at DESC'
    })
    logs.value = response.logs || []
    
    // 批量加载用户信息
    await loadUserInfos()
    
    // 标记为已加载，记录参数
    hasLoaded.value = true
    lastLoadParams.value = {
      fullCodePath: props.fullCodePath,
      rowId: props.rowId
    }
  } catch (error: any) {
    console.error('[OperateLogSection] 加载操作日志失败:', error)
    ElMessage.warning('加载操作日志失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

/**
 * 批量加载用户信息
 */
const loadUserInfos = async () => {
  if (logs.value.length === 0) {
    return
  }
  
  // 收集所有唯一的用户名
  const usernames = new Set<string>()
  logs.value.forEach((log: TableOperateLog) => {
    if (log.request_user) {
      usernames.add(log.request_user)
    }
  })
  
  if (usernames.size === 0) {
    return
  }
  
  try {
    const users = await userInfoStore.batchGetUserInfo(Array.from(usernames))
    
    // 更新用户信息映射
    userInfoMap.value = new Map()
    users.forEach((user: any) => {
      userInfoMap.value.set(user.username, user)
    })
  } catch (error) {
    console.warn('[OperateLogSection] 加载用户信息失败:', error)
  }
}

/**
 * 获取用户信息
 */
const getUserInfo = (username: string | null | undefined): any => {
  if (!username) {
    return null
  }
  return userInfoMap.value.get(username) || null
}

/**
 * 获取操作类型标签类型
 */
const getActionTagType = (action: string): TagProps['type'] => {
  switch (action) {
    case 'OnTableAddRow':
      return 'success'
    case 'OnTableUpdateRow':
      return 'warning'
    case 'OnTableDeleteRows':
      return 'danger'
    default:
      return 'info'
  }
}

/**
 * 获取操作类型标签文本
 */
const getActionLabel = (action: string): string => {
  switch (action) {
    case 'OnTableAddRow':
      return '新增'
    case 'OnTableUpdateRow':
      return '更新'
    case 'OnTableDeleteRows':
      return '删除'
    default:
      return action
  }
}

/**
 * 解析 JSON 字符串
 */
const parseJSON = (jsonStr: string | any): any => {
  if (typeof jsonStr === 'string') {
    try {
      return JSON.parse(jsonStr)
    } catch {
      return {}
    }
  }
  return jsonStr || {}
}

/**
 * 格式化值
 */
const formatValue = (value: any): string => {
  if (value === null || value === undefined) {
    return '-'
  }
  if (typeof value === 'object') {
    return JSON.stringify(value)
  }
  return String(value)
}

/**
 * 根据字段 code 获取字段配置
 */
const getFieldConfig = (fieldCode: string): FieldConfig | null => {
  // 优先使用缓存的 functionDetail
  const detail = functionDetailCache.value || props.functionDetail
  
  if (!detail) {
    return null
  }
  
  // 只使用 functionDetail.response（响应字段）
  // 注意：不要使用 request，因为 request 是编辑模式下的字段，response 才是详情展示的字段
  let fields: FieldConfig[] | null = null
  
  if (Array.isArray(detail.response) && detail.response.length > 0) {
    fields = detail.response
  } else if (Array.isArray(detail)) {
    // 如果 detail 本身就是字段数组（兼容旧格式）
    fields = detail
  }
  
  if (!Array.isArray(fields) || fields.length === 0) {
    return null
  }
  
  const field = fields.find((f: any) => f.code === fieldCode)
  return field || null
}

/**
 * 根据字段 code 获取字段名称
 */
const getFieldName = (fieldCode: string | number): string => {
  const normalizedFieldCode = String(fieldCode)
  const field = getFieldConfig(normalizedFieldCode)
  return field?.name || normalizedFieldCode
}

/**
 * 渲染字段值（使用组件渲染，与详情页一致）
 */
const renderFieldValue = (fieldCode: string | number, rawValue: any) => {
  const normalizedFieldCode = String(fieldCode)
  const field = getFieldConfig(normalizedFieldCode)
  
  if (!field) {
    // 如果没有字段配置，返回纯文本
    console.warn('[OperateLogSection] 未找到字段配置:', fieldCode, 'functionDetail:', props.functionDetail)
    return h('span', { class: 'text-fallback' }, rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
  }
  
  try {
    // 对于 files 类型，rawValue 可能已经是一个包含 files 数组的对象
    // 需要确保值格式正确
    let processedValue = rawValue
    
    // 如果字段类型是 files，且 rawValue 是一个对象，确保它包含 files 数组
    if (field.widget?.type === 'files' && rawValue && typeof rawValue === 'object') {
      // 如果 rawValue 已经有 files 属性，直接使用
      if (rawValue.files && Array.isArray(rawValue.files)) {
        processedValue = rawValue
      } else {
        // 否则，将 rawValue 包装为 files 格式
        processedValue = {
          files: Array.isArray(rawValue) ? rawValue : [rawValue],
          remark: rawValue.remark || '',
          metadata: rawValue.metadata || null
        }
      }
    }
    
    // 将原始值转换为 FieldValue 格式
    const value = convertToFieldValue(processedValue, field)
    
    // 获取组件
    const WidgetComponent = widgetComponentFactory.getRequestComponent(
      field.widget?.type || 'input'
    )
    
    if (!WidgetComponent) {
      // 如果组件未找到，返回 fallback
      console.warn('[OperateLogSection] 未找到组件:', field.widget?.type || 'input')
      return h('span', { class: 'text-fallback' }, rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
    }
    
    // 使用 h() 渲染组件为 VNode（detail 模式）
    return h(WidgetComponent, {
      field: field,
      value: value,
      'model-value': value,
      'field-path': normalizedFieldCode,
      mode: 'detail', // 使用 detail 模式，只读展示
    })
  } catch (error) {
    console.error('[OperateLogSection] 渲染字段值失败:', error, 'fieldCode:', fieldCode, 'rawValue:', rawValue)
    // 错误处理：返回 fallback
    return h('span', { class: 'text-fallback' }, rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
  }
}

/**
 * 处理升级按钮点击
 */
const handleUpgrade = () => {
  ElMessage.info('请联系管理员升级到企业版')
}

// 是否已加载过（用于懒加载场景）
const hasLoaded = ref(false)
// 记录上次加载的参数，用于判断是否需要重新加载
const lastLoadParams = ref<{ fullCodePath: string; rowId: number } | null>(null)

// 监听 props 变化，自动加载操作日志和函数详情（仅在 autoLoad 为 true 时）
watch(
  [() => props.fullCodePath, () => props.rowId, () => props.functionDetail],
  ([newFullCodePath, newRowId, newFunctionDetail], [oldFullCodePath = '', oldRowId = 0, oldFunctionDetail]) => {
    
    // 如果参数变化，重置加载状态
    if (newFullCodePath && newRowId && (newFullCodePath !== oldFullCodePath || newRowId !== oldRowId)) {
      hasLoaded.value = false
      lastLoadParams.value = null
      logs.value = []
    }
    
    // 如果 functionDetail 变化，清除缓存
    if (newFunctionDetail !== oldFunctionDetail) {
      functionDetailCache.value = null
    }
    
    // 只有 autoLoad 为 true 时才自动加载
    if (!props.autoLoad) {
      return
    }
    
    // 只有当值真正变化时才加载（避免初始化时重复加载）
    if (newFullCodePath && newRowId && (newFullCodePath !== oldFullCodePath || newRowId !== oldRowId)) {
      loadOperateLogs()
    } else if (newFullCodePath && newRowId && !oldFullCodePath && !oldRowId) {
      // 初始化时也加载
      loadOperateLogs()
    }
  },
  { immediate: true }
)

// 暴露 load 方法供外部调用
defineExpose({
  load: () => {
    // 检查参数是否变化，如果变化了需要重新加载
    const paramsChanged = !lastLoadParams.value || 
      lastLoadParams.value.fullCodePath !== props.fullCodePath ||
      lastLoadParams.value.rowId !== props.rowId
    
    if (paramsChanged) {
      // 参数变化，清除状态重新加载
      hasLoaded.value = false
      logs.value = []
      functionDetailCache.value = null
    }
    
    // 如果还没加载过，或者参数变化了，则加载
    if (!hasLoaded.value) {
      loadOperateLogs()
    }
  }
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
