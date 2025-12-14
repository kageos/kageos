import { get, post } from '@/utils/request'

// License 状态接口
export interface LicenseStatus {
  is_valid: boolean
  is_community: boolean
  edition: string
  customer?: string
  description?: string
  expires_at?: string
  features?: {
    operate_log?: boolean
  }
}

// 激活 License 响应
export interface ActivateLicenseResponse {
  message: string
  status: LicenseStatus
}

/**
 * 获取 License 状态
 */
export function getLicenseStatus(): Promise<LicenseStatus> {
  return get<LicenseStatus>('/api/v1/control/license/status')
}

/**
 * 激活 License
 * @param licenseFile License 文件
 */
export async function activateLicense(licenseFile: File): Promise<LicenseStatus> {
  // 读取文件内容
  const licenseText = await licenseFile.text()
  
  // 验证是否为有效的 JSON
  try {
    JSON.parse(licenseText)
  } catch (error) {
    throw new Error('License 文件格式错误，必须是有效的 JSON 文件')
  }
  
  // 后端使用 c.GetRawData() 读取原始数据，需要发送原始 JSON 字符串
  // 使用 fetch 直接发送，避免被 axios 拦截器处理
  const token = localStorage.getItem('token') || ''
  const baseURL = import.meta.env.VITE_API_BASE_URL || ''
  const url = `${baseURL}/api/v1/control/license/activate`
  
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Token': token,
    },
    body: licenseText, // 直接发送 JSON 字符串
  })
  
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: '激活失败' }))
    throw new Error(error.error || `激活失败: ${response.statusText}`)
  }
  
  const result: ActivateLicenseResponse = await response.json()
  return result.status
}

/**
 * 注销 License
 */
export function deactivateLicense(): Promise<LicenseStatus> {
  return post<LicenseStatus>('/api/v1/control/license/deactivate')
}

