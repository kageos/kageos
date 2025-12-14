import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getLicenseStatus, activateLicense, deactivateLicense, type LicenseStatus } from '@/api/license'

const LICENSE_STORAGE_KEY = 'license'
const ACTIVATED_AT_KEY = 'license_activated_at'
const CHECK_INTERVAL = 60 * 60 * 1000 // 1小时（毫秒）

export const useLicenseStore = defineStore('license', () => {
  // 状态
  const license = ref<LicenseStatus | null>(null)
  const loading = ref(false)
  let checkTimer: number | null = null

  // 从本地存储加载 License
  function loadFromLocal() {
    const stored = localStorage.getItem(LICENSE_STORAGE_KEY)
    if (stored) {
      try {
        license.value = JSON.parse(stored)
      } catch (error) {
        console.error('解析本地 License 失败:', error)
        localStorage.removeItem(LICENSE_STORAGE_KEY)
        license.value = null
      }
    }
  }

  // 保存到本地存储
  function saveToLocal(status: LicenseStatus | null) {
    if (status && status.is_valid) {
      localStorage.setItem(LICENSE_STORAGE_KEY, JSON.stringify(status))
      // 保存激活时间
      localStorage.setItem(ACTIVATED_AT_KEY, new Date().toISOString())
    } else {
      localStorage.removeItem(LICENSE_STORAGE_KEY)
      localStorage.removeItem(ACTIVATED_AT_KEY)
    }
  }

  // 获取激活时间
  function getActivatedAt(): Date | null {
    const stored = localStorage.getItem(ACTIVATED_AT_KEY)
    if (stored) {
      try {
        return new Date(stored)
      } catch (error) {
        console.error('解析激活时间失败:', error)
        return null
      }
    }
    return null
  }

  // 计算属性
  const isEnterprise = computed(() => {
    return license.value?.is_valid && !license.value?.is_community
  })

  const hasOperateLog = computed(() => {
    return license.value?.features?.operate_log === true
  })

  const edition = computed(() => {
    return license.value?.edition || 'community'
  })

  const customer = computed(() => {
    return license.value?.customer
  })

  const description = computed(() => {
    return license.value?.description
  })

  const expiresAt = computed(() => {
    return license.value?.expires_at
  })

  // 判断是否过期（基于 expires_at）
  const isExpired = computed(() => {
    if (!license.value?.expires_at) {
      return false // 没有过期时间，认为不过期
    }
    const expires = new Date(license.value.expires_at)
    return expires < new Date()
  })

  // 从后端获取 License 状态
  async function fetchStatus() {
    loading.value = true
    try {
      const status = await getLicenseStatus()
      license.value = status
      saveToLocal(status)
      return status
    } catch (error: any) {
      console.error('获取 License 状态失败:', error)
      // 如果后端获取失败，尝试从本地恢复
      loadFromLocal()
      throw error
    } finally {
      loading.value = false
    }
  }

  // 启动定时检查（每小时一次）
  function startPeriodicCheck() {
    // 清除之前的定时器
    if (checkTimer !== null) {
      clearInterval(checkTimer)
    }

    // 设置新的定时器（每小时检查一次）
    checkTimer = window.setInterval(() => {
      console.log('[License Store] 定时检查 License 状态...')
      fetchStatus().catch((error) => {
        console.warn('[License Store] 定时检查失败，使用本地缓存:', error)
        // 检查失败时，使用本地缓存，但基于 expires_at 判断是否过期
        if (isExpired.value) {
          console.warn('[License Store] License 已过期（基于 expires_at）')
          // 可以在这里触发过期提示
        }
      })
    }, CHECK_INTERVAL)

    console.log('[License Store] 已启动定时检查，间隔: 1小时')
  }

  // 停止定时检查
  function stopPeriodicCheck() {
    if (checkTimer !== null) {
      clearInterval(checkTimer)
      checkTimer = null
      console.log('[License Store] 已停止定时检查')
    }
  }

  // 激活 License
  async function activate(licenseFile: File) {
    loading.value = true
    try {
      const status = await activateLicense(licenseFile)
      license.value = status
      saveToLocal(status) // 保存时会记录激活时间
      ElMessage.success('License 激活成功！')
      // 激活后立即检查一次，然后启动定时检查
      startPeriodicCheck()
      return status
    } catch (error: any) {
      console.error('激活 License 失败:', error)
      ElMessage.error(error.message || '激活 License 失败')
      throw error
    } finally {
      loading.value = false
    }
  }

  // 注销 License
  async function deactivate() {
    // 确认对话框
    try {
      await ElMessageBox.confirm(
        '确定要注销 License 吗？注销后系统将回到社区版，所有企业功能将不可用。',
        '确认注销',
        {
          confirmButtonText: '确定注销',
          cancelButtonText: '取消',
          type: 'warning',
        }
      )
    } catch {
      // 用户取消
      return
    }

    loading.value = true
    try {
      const status = await deactivateLicense()
      license.value = status
      saveToLocal(status) // 保存时会清除激活时间
      ElMessage.success('License 注销成功，系统已回到社区版')
      // 注销后停止定时检查
      stopPeriodicCheck()
      return status
    } catch (error: any) {
      console.error('注销 License 失败:', error)
      ElMessage.error(error.message || '注销 License 失败')
      throw error
    } finally {
      loading.value = false
    }
  }

  // 初始化：从本地加载
  loadFromLocal()

  // 如果已有激活的 License，启动定时检查
  if (license.value?.is_valid && !license.value?.is_community) {
    startPeriodicCheck()
  }

  return {
    // 状态
    license,
    loading,
    
    // 计算属性
    isEnterprise,
    hasOperateLog,
    edition,
    customer,
    description,
    expiresAt,
    isExpired,
    
    // 方法
    fetchStatus,
    activate,
    deactivate,
    loadFromLocal,
    startPeriodicCheck,
    stopPeriodicCheck,
    getActivatedAt,
  }
})

