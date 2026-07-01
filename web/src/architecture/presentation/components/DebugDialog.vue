<!--
  DebugDialog - 开发调试弹窗
  功能：
  - 清理各种缓存
  - 查看缓存统计信息
  - 其他调试工具
-->
<template>
  <el-dialog
    v-model="visible"
    :title="t('debug.title')"
    width="600px"
    :close-on-click-modal="false"
  >
    <div class="debug-content">
      <!-- 缓存清理区域 -->
      <div class="debug-section">
        <div class="section-title">{{ t('debug.cacheClear') }}</div>
        <div class="cache-actions">
          <el-button
            type="danger"
            @click="handleClearFunctionCache"
            :loading="clearingFunctionCache"
          >
            {{ t('debug.clearFunctionCache') }}
          </el-button>
          <el-button
            type="danger"
            @click="handleClearUserCache"
            :loading="clearingUserCache"
          >
            {{ t('debug.clearUserCache') }}
          </el-button>
          <el-button
            type="danger"
            @click="handleClearDepartmentCache"
            :loading="clearingDepartmentCache"
          >
            {{ t('debug.clearDepartmentCache') }}
          </el-button>
          <el-button
            type="danger"
            @click="handleClearAllCache"
            :loading="clearingAllCache"
          >
            {{ t('debug.clearAllCache') }}
          </el-button>
        </div>
      </div>

      <!-- 缓存统计区域 -->
      <div class="debug-section">
        <div class="section-title">{{ t('debug.cacheStats') }}</div>
        <div class="cache-stats">
          <div class="stat-item">
            <span class="stat-label">{{ t('debug.functionDetailCache') }}:</span>
            <span class="stat-value">{{ t('debug.itemCount', { count: functionCacheCount }) }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">{{ t('debug.userInfoCache') }}:</span>
            <span class="stat-value">
              {{ t('debug.itemCount', { count: userCacheStats.total }) }}
              <span v-if="userCacheStats.expired > 0" class="expired-count">
                {{ t('debug.expiredCount', { count: userCacheStats.expired }) }}
              </span>
            </span>
          </div>
          <div class="stat-item">
            <span class="stat-label">{{ t('debug.loadingUsers') }}:</span>
            <span class="stat-value">{{ t('debug.itemCount', { count: userCacheStats.loading }) }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">{{ t('debug.departmentInfoCache') }}:</span>
            <span class="stat-value">
              {{ t('debug.itemCount', { count: departmentCacheStats.total }) }}
              <span v-if="departmentCacheStats.expired > 0" class="expired-count">
                {{ t('debug.expiredCount', { count: departmentCacheStats.expired }) }}
              </span>
            </span>
          </div>
          <div class="stat-item">
            <span class="stat-label">{{ t('debug.loadingDepartments') }}:</span>
            <span class="stat-value">{{ t('debug.itemCount', { count: departmentCacheStats.loading }) }}</span>
          </div>
        </div>
      </div>

      <!-- 函数详情缓存列表 -->
      <div class="debug-section">
        <div class="section-title">
          {{ t('debug.functionDetailCache') }}
          <el-button
            text
            type="primary"
            size="small"
            @click="showFunctionCacheDetails = !showFunctionCacheDetails"
            style="margin-left: 8px;"
          >
            {{ showFunctionCacheDetails ? t('debug.collapse') : t('debug.expand') }}
          </el-button>
        </div>
        <div v-if="showFunctionCacheDetails" class="cache-details">
          <el-table
            :data="functionCacheList"
            stripe
            size="small"
            max-height="300"
            style="width: 100%"
          >
            <el-table-column prop="key" :label="t('debug.cacheKey')" min-width="200" show-overflow-tooltip />
            <el-table-column prop="type" :label="t('debug.type')" width="100">
              <template #default="{ row }">
                <el-tag :type="row.type === 'id' ? 'primary' : 'success'" size="small">
                  {{ row.type === 'id' ? 'ID' : 'Path' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="functionName" :label="t('debug.functionName')" min-width="150" show-overflow-tooltip />
            <el-table-column prop="templateType" :label="t('debug.templateType')" width="100">
              <template #default="{ row }">
                <el-tag v-if="row.templateType" size="small">{{ row.templateType }}</el-tag>
                <span v-else>-</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <!-- 用户信息缓存列表 -->
      <div class="debug-section">
        <div class="section-title">
          {{ t('debug.userInfoCache') }}
          <el-button
            text
            type="primary"
            size="small"
            @click="showUserCacheDetails = !showUserCacheDetails"
            style="margin-left: 8px;"
          >
            {{ showUserCacheDetails ? t('debug.collapse') : t('debug.expand') }}
          </el-button>
        </div>
        <div v-if="showUserCacheDetails" class="cache-details">
          <el-table
            :data="userCacheList"
            stripe
            size="small"
            max-height="400"
            style="width: 100%"
          >
            <el-table-column prop="username" :label="t('debug.username')" width="120" />
            <el-table-column prop="nickname" :label="t('debug.nickname')" width="120" show-overflow-tooltip />
            <el-table-column prop="status" :label="t('debug.status')" width="100">
              <template #default="{ row }">
                <el-tag
                  :type="row.isExpired ? 'warning' : 'success'"
                  size="small"
                >
                  {{ row.isExpired ? t('debug.expired') : t('debug.valid') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="cachedTime" :label="t('debug.cachedTime')" width="180" />
            <el-table-column prop="expiredTime" :label="t('debug.expiredTime')" width="180">
              <template #default="{ row }">
                <span :class="{ 'expired-text': row.isExpired }">
                  {{ row.expiredTime }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="age" :label="t('debug.cacheAge')" width="120">
              <template #default="{ row }">
                <span :class="{ 'expired-text': row.isExpired }">
                  {{ row.age }}
                </span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <!-- 部门信息缓存列表 -->
      <div class="debug-section">
        <div class="section-title">
          {{ t('debug.departmentInfoCache') }}
          <el-button
            text
            type="primary"
            size="small"
            @click="showDepartmentCacheDetails = !showDepartmentCacheDetails"
            style="margin-left: 8px;"
          >
            {{ showDepartmentCacheDetails ? t('debug.collapse') : t('debug.expand') }}
          </el-button>
        </div>
        <div v-if="showDepartmentCacheDetails" class="cache-details">
          <el-table
            :data="departmentCacheList"
            stripe
            size="small"
            max-height="400"
            style="width: 100%"
          >
            <el-table-column prop="path" :label="t('debug.departmentPath')" min-width="200" show-overflow-tooltip />
            <el-table-column prop="name" :label="t('debug.departmentName')" width="150" show-overflow-tooltip />
            <el-table-column prop="status" :label="t('debug.status')" width="100">
              <template #default="{ row }">
                <el-tag
                  :type="row.isExpired ? 'warning' : 'success'"
                  size="small"
                >
                  {{ row.isExpired ? t('debug.expired') : t('debug.valid') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="cachedTime" :label="t('debug.cachedTime')" width="180" />
            <el-table-column prop="expiredTime" :label="t('debug.expiredTime')" width="180">
              <template #default="{ row }">
                <span :class="{ 'expired-text': row.isExpired }">
                  {{ row.expiredTime }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="age" :label="t('debug.cacheAge')" width="120">
              <template #default="{ row }">
                <span :class="{ 'expired-text': row.isExpired }">
                  {{ row.age }}
                </span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <!-- 其他工具区域 -->
      <div class="debug-section">
        <div class="section-title">{{ t('debug.tools') }}</div>
        <div class="tool-actions">
          <el-button
            type="primary"
            @click="handleReloadPage"
          >
            {{ t('debug.reloadPage') }}
          </el-button>
          <el-button
            type="info"
            @click="handleCopyCacheInfo"
          >
            {{ t('debug.copyCacheInfo') }}
          </el-button>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">{{ t('common.close') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { functionLoader } from '../../infrastructure/functionLoader'
import { useDepartmentInfoStore, useUserInfoStore } from '@/architecture/presentation/context/appStoresContext'
import { cacheManager } from '../../infrastructure/cacheManager'
import { Logger } from '@/architecture/shared/logger'

interface Props {
  modelValue: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const { t, locale } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (val: boolean) => emit('update:modelValue', val)
})

const userInfoStore = useUserInfoStore()
const departmentInfoStore = useDepartmentInfoStore()

// 加载状态
const clearingFunctionCache = ref(false)
const clearingUserCache = ref(false)
const clearingDepartmentCache = ref(false)
const clearingAllCache = ref(false)

// 显示/隐藏详情
const showFunctionCacheDetails = ref(false)
const showUserCacheDetails = ref(false)
const showDepartmentCacheDetails = ref(false)

// 缓存统计
const functionCacheCount = ref(0)
const userCacheStats = ref({
  total: 0,
  valid: 0,
  expired: 0,
  loading: 0
})
const departmentCacheStats = ref({
  total: 0,
  valid: 0,
  expired: 0,
  loading: 0
})

// 缓存详情列表
interface FunctionCacheItem {
  key: string
  type: 'id' | 'path'
  functionName: string
  templateType: string
}

interface UserCacheItem {
  username: string
  nickname: string
  isExpired: boolean
  cachedTime: string
  expiredTime: string
  age: string
}

interface DepartmentCacheItem {
  path: string
  name: string
  isExpired: boolean
  cachedTime: string
  expiredTime: string
  age: string
}

const functionCacheList = ref<FunctionCacheItem[]>([])
const userCacheList = ref<UserCacheItem[]>([])
const departmentCacheList = ref<DepartmentCacheItem[]>([])

// 格式化时间
const formatTime = (timestamp: number): string => {
  return new Date(timestamp).toLocaleString(String(locale.value), {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 格式化时长
const formatAge = (ms: number): string => {
  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  
  if (days > 0) {
    return t('debug.daysHours', { days, hours: hours % 24 })
  } else if (hours > 0) {
    return t('debug.hoursMinutes', { hours, minutes: minutes % 60 })
  } else if (minutes > 0) {
    return t('debug.minutesSeconds', { minutes, seconds: seconds % 60 })
  } else {
    return t('debug.seconds', { seconds })
  }
}

// 更新缓存统计和详情
const updateCacheStats = () => {
  try {
    // 获取函数详情缓存
    const allKeys = cacheManager.getKeys?.() || []
    const functionKeys = allKeys.filter((key: string) => key.startsWith('function:'))
    functionCacheCount.value = functionKeys.length
    
    // 构建函数详情缓存列表
    functionCacheList.value = functionKeys.map((key: string) => {
      const cached = cacheManager.get<any>(key)
      const identifier = key.replace('function:path:', '')
      
      return {
        key,
        type: 'path',
        functionName: cached?.name || cached?.router || identifier,
        templateType: cached?.template_type || '-'
      }
    })
    
    // 获取用户信息缓存统计
    const stats = userInfoStore.getCacheStats()
    userCacheStats.value = stats
    
    // 构建用户信息缓存列表
    // 使用 userInfoStore 的 getCacheDetails 方法获取详情
    try {
      const details = userInfoStore.getCacheDetails()
      userCacheList.value = details.map((item: {
        username: string
        nickname: string
        isExpired: boolean
        cachedTime: number
        expiredTime: number
        age: number
      }) => ({
        username: item.username,
        nickname: item.nickname || '-',
        isExpired: item.isExpired,
        cachedTime: formatTime(item.cachedTime),
        expiredTime: formatTime(item.expiredTime),
        age: formatAge(item.age)
      }))
    } catch (error) {
      Logger.warn('[DebugDialog]', 'failed to get user info cache details', { error })
      userCacheList.value = []
    }
    
    // 获取部门信息缓存统计
    const deptStats = departmentInfoStore.getCacheStats()
    departmentCacheStats.value = deptStats
    
    // 构建部门信息缓存列表
    // 使用 departmentInfoStore 的 getCacheDetails 方法获取详情
    try {
      const details = departmentInfoStore.getCacheDetails()
      departmentCacheList.value = details.map((item: {
        path: string
        name: string
        isExpired: boolean
        cachedTime: number
        expiredTime: number
        age: number
      }) => ({
        path: item.path,
        name: item.name || '-',
        isExpired: item.isExpired,
        cachedTime: formatTime(item.cachedTime),
        expiredTime: formatTime(item.expiredTime),
        age: formatAge(item.age)
      }))
    } catch (error) {
      Logger.warn('[DebugDialog]', 'failed to get department info cache details', { error })
      departmentCacheList.value = []
    }
  } catch (_error) {
    Logger.error('[DebugDialog]', 'failed to get cache stats', { error: _error })
  }
}

// 监听弹窗打开，更新统计信息
watch(visible, (newVal: boolean) => {
  if (newVal) {
    updateCacheStats()
  }
})

// 清理函数详情缓存
const handleClearFunctionCache = async () => {
  try {
    await ElMessageBox.confirm(
      t('debug.clearFunctionConfirm'),
      t('debug.clearFunctionTitle'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )
    
    clearingFunctionCache.value = true
    functionLoader.clearCache()
    ElMessage.success(t('debug.clearFunctionSuccess'))
    updateCacheStats()
  } catch (_error) {
    // 忽略取消操作
  } finally {
    clearingFunctionCache.value = false
  }
}

// 清理用户信息缓存
const handleClearUserCache = async () => {
  try {
    await ElMessageBox.confirm(
      t('debug.clearUserConfirm'),
      t('debug.clearUserTitle'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )
    
    clearingUserCache.value = true
    userInfoStore.clearCache()
    ElMessage.success(t('debug.clearUserSuccess'))
    updateCacheStats()
  } catch (_error) {
    // 忽略取消操作
  } finally {
    clearingUserCache.value = false
  }
}

// 清理部门信息缓存
const handleClearDepartmentCache = async () => {
  try {
    await ElMessageBox.confirm(
      t('debug.clearDepartmentConfirm'),
      t('debug.clearDepartmentTitle'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )
    
    clearingDepartmentCache.value = true
    departmentInfoStore.clearCache()
    ElMessage.success(t('debug.clearDepartmentSuccess'))
    updateCacheStats()
  } catch (_error) {
    // 忽略取消操作
  } finally {
    clearingDepartmentCache.value = false
  }
}

// 清理所有缓存
const handleClearAllCache = async () => {
  try {
    await ElMessageBox.confirm(
      t('debug.clearAllConfirm'),
      t('debug.clearAllTitle'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )
    
    clearingAllCache.value = true
    functionLoader.clearCache()
    userInfoStore.clearCache()
    departmentInfoStore.clearCache()
    ElMessage.success(t('debug.clearAllSuccess'))
    updateCacheStats()
    
    // 延迟刷新页面，让用户看到成功消息
    setTimeout(() => {
      window.location.reload()
    }, 500)
  } catch (_error) {
    // 忽略取消操作
  } finally {
    clearingAllCache.value = false
  }
}

// 刷新页面
const handleReloadPage = () => {
  window.location.reload()
}

// 复制缓存信息
const handleCopyCacheInfo = async () => {
  try {
    const cacheInfo = {
      functionCache: {
        count: functionCacheCount.value
      },
      userCache: userCacheStats.value,
      departmentCache: departmentCacheStats.value,
      timestamp: new Date().toISOString()
    }
    
    const text = JSON.stringify(cacheInfo, null, 2)
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('debug.copied'))
  } catch (_error) {
    ElMessage.error(t('debug.copyFailed'))
  }
}
</script>

<style scoped lang="scss">
.debug-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.debug-section {
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}

.cache-actions,
.tool-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.cache-stats {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stat-item {
  display: flex;
  align-items: center;
  font-size: 14px;
}

.stat-label {
  color: var(--el-text-color-regular);
  min-width: 140px;
}

.stat-value {
  color: var(--el-text-color-primary);
  font-weight: 500;
}

.expired-count {
  color: var(--el-color-warning);
  font-size: 12px;
}

.cache-details {
  margin-top: 12px;
}

.expired-text {
  color: var(--el-color-warning);
  font-weight: 500;
}
</style>
