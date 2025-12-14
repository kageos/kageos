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
        <el-table
          v-if="logs.length > 0"
          :data="logs"
          stripe
          size="small"
          style="width: 100%"
        >
          <el-table-column prop="action" label="操作" width="120">
            <template #default="{ row }">
              <el-tag :type="getActionTagType(row.action)" size="small">
                {{ getActionLabel(row.action) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="request_user" label="操作人" width="150">
            <template #default="{ row }">
              <UserDisplay
                :user-info="getUserInfo(row.request_user)"
                :username="row.request_user"
                mode="card"
                layout="horizontal"
                size="small"
              />
            </template>
          </el-table-column>
              <el-table-column prop="created_at" label="操作时间" width="180">
                <template #default="{ row }">
                  {{ formatDateTime(row.created_at) }}
                </template>
              </el-table-column>
              <el-table-column prop="updates" label="变更内容" min-width="300">
                <template #default="{ row }">
                  <div v-if="row.action === 'OnTableUpdateRow' && row.updates" class="update-content">
                    <div v-for="(value, key) in parseJSON(row.updates)" :key="key" class="update-item">
                      <div class="update-item-row">
                        <!-- 左侧：字段名称 -->
                        <div class="update-field-label">{{ getFieldName(key) }}</div>
                        <!-- 中间：上下排列的值 -->
                        <div class="update-values-col">
                          <!-- 新值（修改后的值）- 在上面 -->
                          <div class="update-value-new">
                            <div class="value-label">新值</div>
                            <div class="value-content">
                              <component :is="renderFieldValue(key, value)" />
                            </div>
                          </div>
                          <!-- 原值（修改前的值）- 在下面 -->
                          <div v-if="row.old_values && parseJSON(row.old_values)[key] !== undefined" class="update-value-old">
                            <div class="value-label">原值</div>
                            <div class="value-content">
                              <component :is="renderFieldValue(key, parseJSON(row.old_values)[key])" />
                            </div>
                          </div>
                        </div>
                        <!-- 右侧：上箭头 -->
                        <div class="update-arrow">
                          <el-icon><ArrowUp /></el-icon>
                        </div>
                      </div>
                    </div>
                  </div>
                  <span v-else class="text-muted">-</span>
                </template>
              </el-table-column>
        </el-table>
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
import { Clock, ArrowUp } from '@element-plus/icons-vue'
import { ElIcon, ElTable, ElTableColumn, ElEmpty, ElTag, ElCard, ElDivider, ElButton, ElMessage } from 'element-plus'
import { formatTimestamp } from '@/utils/date'
import { useLicenseStore } from '@/stores/license'
import { getTableOperateLogs, type TableOperateLog } from '@/api/operateLog'
import { widgetComponentFactory } from '@/core/factories-v2'
import { convertToFieldValue } from '@/utils/field'
import type { FieldConfig } from '@/types'
import { getFunctionByPath } from '@/api/function'
import type { FunctionDetail } from '@/architecture/domain/interfaces/IFunctionLoader'
import UserDisplay from '@/core/widgets-v2/components/UserDisplay.vue'

interface Props {
  /** 完整代码路径 */
  fullCodePath: string
  /** 记录ID */
  rowId: number
  /** 函数详情（用于获取字段名称和渲染组件） */
  functionDetail?: any
}

const props = withDefaults(defineProps<Props>(), {
  fullCodePath: '',
  rowId: 0,
  functionDetail: undefined
})

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
        functionDetailCache.value = detail as FunctionDetail
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
  logs.value.forEach(log => {
    if (log.request_user) {
      usernames.add(log.request_user)
    }
  })
  
  if (usernames.size === 0) {
    return
  }
  
  try {
    const { useUserInfoStore } = await import('@/stores/userInfo')
    const userInfoStore = useUserInfoStore()
    const users = await userInfoStore.batchGetUserInfo(Array.from(usernames))
    
    // 更新用户信息映射
    userInfoMap.value = new Map()
    users.forEach(user => {
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
const getActionTagType = (action: string): string => {
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
const getFieldName = (fieldCode: string): string => {
  const field = getFieldConfig(fieldCode)
  return field?.name || fieldCode
}

/**
 * 渲染字段值（使用组件渲染，与详情页一致）
 */
const renderFieldValue = (fieldCode: string, rawValue: any) => {
  const field = getFieldConfig(fieldCode)
  
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
      'field-path': fieldCode,
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

// 监听 props 变化，自动加载操作日志和函数详情
watch(
  () => [props.fullCodePath, props.rowId, props.functionDetail],
  (newVal: [string, number, any], oldVal?: [string, number, any]) => {
    const [newFullCodePath, newRowId, newFunctionDetail] = newVal
    const [oldFullCodePath = '', oldRowId = 0, oldFunctionDetail] = oldVal || []
    
    // 如果 functionDetail 变化，清除缓存
    if (newFunctionDetail !== oldFunctionDetail) {
      functionDetailCache.value = null
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

.update-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.update-item {
  padding: 8px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 4px;
}

.update-item-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.update-field-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  min-width: 80px;
  flex-shrink: 0;
  padding-top: 4px;
}

.update-values-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.update-value-new,
.update-value-old {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px 8px;
  border-radius: 4px;
  background-color: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
}

.value-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-bottom: 2px;
}

.value-content {
  flex: 1;
  min-width: 0;
  font-size: 12px;
}

.update-arrow {
  display: flex;
  justify-content: center;
  align-items: center;
  color: var(--el-color-primary);
  font-size: 18px;
  font-weight: 600;
  flex-shrink: 0;
  padding-top: 4px;
  min-width: 24px;
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

