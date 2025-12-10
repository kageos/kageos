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
    title="开发调试工具"
    width="600px"
    :close-on-click-modal="false"
  >
    <div class="debug-content">
      <!-- 缓存清理区域 -->
      <div class="debug-section">
        <div class="section-title">缓存清理</div>
        <div class="cache-actions">
          <el-button
            type="danger"
            @click="handleClearFunctionCache"
            :loading="clearingFunctionCache"
          >
            清理函数详情缓存
          </el-button>
          <el-button
            type="danger"
            @click="handleClearUserCache"
            :loading="clearingUserCache"
          >
            清理用户信息缓存
          </el-button>
          <el-button
            type="danger"
            @click="handleClearAllCache"
            :loading="clearingAllCache"
          >
            清理所有缓存
          </el-button>
        </div>
      </div>

      <!-- 缓存统计区域 -->
      <div class="debug-section">
        <div class="section-title">缓存统计</div>
        <div class="cache-stats">
          <div class="stat-item">
            <span class="stat-label">函数详情缓存：</span>
            <span class="stat-value">{{ functionCacheCount }} 个</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">用户信息缓存：</span>
            <span class="stat-value">
              {{ userCacheStats.total }} 个
              <span v-if="userCacheStats.expired > 0" class="expired-count">
                （{{ userCacheStats.expired }} 个已过期）
              </span>
            </span>
          </div>
          <div class="stat-item">
            <span class="stat-label">正在加载的用户：</span>
            <span class="stat-value">{{ userCacheStats.loading }} 个</span>
          </div>
        </div>
      </div>

      <!-- 函数详情缓存列表 -->
      <div class="debug-section">
        <div class="section-title">
          函数详情缓存
          <el-button
            text
            type="primary"
            size="small"
            @click="showFunctionCacheDetails = !showFunctionCacheDetails"
            style="margin-left: 8px;"
          >
            {{ showFunctionCacheDetails ? '收起' : '展开' }}
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
            <el-table-column prop="key" label="缓存键" min-width="200" show-overflow-tooltip />
            <el-table-column prop="type" label="类型" width="100">
              <template #default="{ row }">
                <el-tag :type="row.type === 'id' ? 'primary' : 'success'" size="small">
                  {{ row.type === 'id' ? 'ID' : 'Path' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="functionName" label="函数名称" min-width="150" show-overflow-tooltip />
            <el-table-column prop="templateType" label="模板类型" width="100">
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
          用户信息缓存
          <el-button
            text
            type="primary"
            size="small"
            @click="showUserCacheDetails = !showUserCacheDetails"
            style="margin-left: 8px;"
          >
            {{ showUserCacheDetails ? '收起' : '展开' }}
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
            <el-table-column prop="username" label="用户名" width="120" />
            <el-table-column prop="nickname" label="昵称" width="120" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag
                  :type="row.isExpired ? 'warning' : 'success'"
                  size="small"
                >
                  {{ row.isExpired ? '已过期' : '有效' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="cachedTime" label="缓存时间" width="180" />
            <el-table-column prop="expiredTime" label="过期时间" width="180">
              <template #default="{ row }">
                <span :class="{ 'expired-text': row.isExpired }">
                  {{ row.expiredTime }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="age" label="缓存时长" width="120">
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
        <div class="section-title">其他工具</div>
        <div class="tool-actions">
          <el-button
            type="primary"
            @click="handleReloadPage"
          >
            刷新页面
          </el-button>
          <el-button
            type="info"
            @click="handleCopyCacheInfo"
          >
            复制缓存信息
          </el-button>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { functionLoader } from '../../infrastructure/functionLoader'
import { useUserInfoStore } from '@/stores/userInfo'
import { cacheManager } from '../../infrastructure/cacheManager'

interface Props {
  modelValue: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val: boolean) => emit('update:modelValue', val)
})

const userInfoStore = useUserInfoStore()

// 加载状态
const clearingFunctionCache = ref(false)
const clearingUserCache = ref(false)
const clearingAllCache = ref(false)

// 显示/隐藏详情
const showFunctionCacheDetails = ref(false)
const showUserCacheDetails = ref(false)

// 缓存统计
const functionCacheCount = ref(0)
const userCacheStats = ref({
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

const functionCacheList = ref<FunctionCacheItem[]>([])
const userCacheList = ref<UserCacheItem[]>([])

// 格式化时间
const formatTime = (timestamp: number): string => {
  return new Date(timestamp).toLocaleString('zh-CN', {
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
    return `${days}天${hours % 24}小时`
  } else if (hours > 0) {
    return `${hours}小时${minutes % 60}分钟`
  } else if (minutes > 0) {
    return `${minutes}分钟${seconds % 60}秒`
  } else {
    return `${seconds}秒`
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
      const isId = key.startsWith('function:id:')
      const identifier = isId ? key.replace('function:id:', '') : key.replace('function:path:', '')
      
      return {
        key,
        type: isId ? 'id' : 'path',
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
      console.warn('[DebugDialog] 无法获取用户信息缓存详情', error)
      userCacheList.value = []
    }
  } catch (error) {
    console.error('[DebugDialog] 获取缓存统计失败', error)
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
      '确定要清理函数详情缓存吗？这将清除所有函数详情缓存，需要重新加载。',
      '清理函数详情缓存',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    clearingFunctionCache.value = true
    functionLoader.clearCache()
    ElMessage.success('函数详情缓存已清理')
    updateCacheStats()
  } catch (error) {
    // 忽略取消操作
  } finally {
    clearingFunctionCache.value = false
  }
}

// 清理用户信息缓存
const handleClearUserCache = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要清理用户信息缓存吗？这将清除所有用户信息缓存（包括 localStorage）。',
      '清理用户信息缓存',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    clearingUserCache.value = true
    userInfoStore.clearCache()
    ElMessage.success('用户信息缓存已清理')
    updateCacheStats()
  } catch (error) {
    // 忽略取消操作
  } finally {
    clearingUserCache.value = false
  }
}

// 清理所有缓存
const handleClearAllCache = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要清理所有缓存吗？这将清除函数详情缓存和用户信息缓存，需要重新加载页面。',
      '清理所有缓存',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    clearingAllCache.value = true
    functionLoader.clearCache()
    userInfoStore.clearCache()
    ElMessage.success('所有缓存已清理')
    updateCacheStats()
    
    // 延迟刷新页面，让用户看到成功消息
    setTimeout(() => {
      window.location.reload()
    }, 500)
  } catch (error) {
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
      timestamp: new Date().toISOString()
    }
    
    const text = JSON.stringify(cacheInfo, null, 2)
    await navigator.clipboard.writeText(text)
    ElMessage.success('缓存信息已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败')
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

