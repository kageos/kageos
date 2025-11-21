/**
 * UserWidget - 用户组件
 * 支持用户选择器和用户信息展示
 */

import { h, ref, computed, watch, onMounted } from 'vue'
import { ElSelect, ElOption, ElAvatar, ElMessage, ElPopover, ElButton, ElDialog } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps } from '../types/widget'
import { searchUsersFuzzy, queryUser } from '@/api/user'
import type { UserInfo } from '@/types'
import { Logger } from '../utils/logger'
import { getElementPlusFormProps } from './utils/widgetHelpers'

/**
 * User 配置
 */
export interface UserConfig {
  placeholder?: string
  clearable?: boolean
  filterable?: boolean
  [key: string]: any
}

/**
 * User 组件数据（用于快照）
 */
interface UserComponentData {
  userOptions: UserInfo[]
  loading: boolean
  userInfo: UserInfo | null
}

export class UserWidget extends BaseWidget {
  // 用户选项列表（用于选择器）
  private userOptions: any
  
  // 当前用户信息（用于显示）
  private userInfo: any
  
  // 加载状态
  private loading: any
  
  // User 配置
  private userConfig: UserConfig
  
  // 防抖定时器
  private searchTimer: ReturnType<typeof setTimeout> | null = null

  constructor(props: WidgetRenderProps) {
    super(props)
    
    // 🔥 在构造函数中初始化 ref
    this.userOptions = ref<UserInfo[]>([])
    this.userInfo = ref<UserInfo | null>(null)
    this.loading = ref(false)
    
    // 解析 User 配置
    this.userConfig = this.getConfig<UserConfig>()
    
    // 初始化用户信息
    this.initUserInfo()
  }

  /**
   * 初始化用户信息
   */
  private initUserInfo(): void {
    // ✅ 临时 Widget 不需要初始化
    if (this.isTemporary) {
      return
    }
    
    // 如果有初始值，加载用户信息
    const currentValue = this.safeGetValue()
    if (currentValue?.raw) {
      this.loadUserInfo(String(currentValue.raw))
    }
  }

  /**
   * 加载用户信息（用于显示）
   */
  private async loadUserInfo(username: string | null): Promise<void> {
    if (!username) {
      this.userInfo.value = null
      return
    }
    
    // 如果 meta 中已有用户信息，直接使用
    const currentValue = this.safeGetValue()
    if (currentValue?.meta?.userInfo) {
      this.userInfo.value = currentValue.meta.userInfo
      return
    }
    
    try {
      // 使用批量查询接口（即使只有一个用户）
      const response = await getUsersByUsernames([username])
      if (response.users && response.users.length > 0) {
        this.userInfo.value = response.users[0]
        
        // 更新 meta 中的用户信息
        if (this.hasFormManager) {
          const value = this.safeGetValue()
          this.safeSetValue(this.fieldPath, {
            ...value,
            meta: {
              ...value.meta,
              userInfo: response.users[0]
            }
          })
        }
      } else {
        this.userInfo.value = null
      }
    } catch (error) {
      Logger.error('UserWidget', `查询用户信息失败: ${username}`, error)
      this.userInfo.value = null
    }
  }

  /**
   * 处理远程搜索（防抖）
   */
  private async handleRemoteSearch(query: string): Promise<void> {
    if (this.searchTimer) {
      clearTimeout(this.searchTimer)
    }
    
    this.searchTimer = setTimeout(async () => {
      if (!query || query.trim() === '') {
        this.userOptions.value = []
        return
      }
      
      try {
        this.loading.value = true
        const response = await searchUsersFuzzy(query.trim(), 20)
        this.userOptions.value = response.users || []
      } catch (error) {
        Logger.error('UserWidget', '搜索用户失败', error)
        this.userOptions.value = []
      } finally {
        this.loading.value = false
      }
    }, 300) // 300ms 防抖
  }

  /**
   * 处理选择变化
   */
  private handleChange(value: any): void {
    const selectedUser = this.userOptions.value.find((u: UserInfo) => u.username === value)
    const newValue: FieldValue = {
      raw: value,
      display: selectedUser?.nickname || selectedUser?.username || String(value),
      meta: {
        userInfo: selectedUser
      }
    }
    
    this.safeSetValue(this.fieldPath, newValue)
    this.onChange(newValue)
  }

  /**
   * 处理聚焦（如果有初始值，加载用户选项）
   */
  private handleFocus(): void {
    const currentValue = this.safeGetValue()
    if (currentValue?.raw && this.userOptions.value.length === 0) {
      // 如果有值但没有选项，尝试搜索
      this.handleRemoteSearch(String(currentValue.raw))
    }
  }

  /**
   * 处理下拉框展开
   */
  private handleVisibleChange(visible: boolean): void {
    if (visible && this.userOptions.value.length === 0) {
      // 展开时，如果有当前值，尝试搜索
      const currentValue = this.safeGetValue()
      if (currentValue?.raw) {
        this.handleRemoteSearch(String(currentValue.raw))
      }
    }
  }

  /**
   * 获取显示名称
   */
  private getDisplayName(): string {
    if (this.userInfo.value) {
      return this.userInfo.value.nickname || this.userInfo.value.username
    }
    const value = this.safeGetValue()
    if (value?.display) {
      return value.display
    }
    if (value?.raw) {
      return String(value.raw)
    }
    return '-'
  }

  /**
   * 渲染组件
   */
  render(): any {
    const value = this.safeGetValue()
    const rawValue = value?.raw
    
    // 如果是临时 Widget（用于表格渲染），使用 renderTableCell
    if (this.isTemporary) {
      return this.renderTableCell(value)
    }
    
    // 正常渲染：用户选择器
    return h(ElSelect, {
      modelValue: rawValue,
      'onUpdate:modelValue': (newValue: any) => this.handleChange(newValue),
      disabled: this.userConfig.disabled || false,
      placeholder: this.userConfig.placeholder || this.field.desc || `请选择${this.field.name}`,
      clearable: this.userConfig.clearable !== false,
      filterable: true,
      loading: this.loading.value,
      remote: true,
      'remote-method': (query: string) => this.handleRemoteSearch(query),
      popperClass: 'user-select-dropdown-popper',
      onFocus: () => this.handleFocus(),
      onVisibleChange: (visible: boolean) => this.handleVisibleChange(visible),
      ...getElementPlusFormProps(this.field)
    }, {
      default: () => {
        return this.userOptions.value.map((user: UserInfo) => {
          return h(ElOption, {
            key: user.username,
            value: user.username,
            label: user.username
          }, {
            default: () => {
              return h('div', {
                class: 'user-option',
                style: {
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px'
                }
              }, [
                h(ElAvatar, {
                  src: user.avatar,
                  size: 24
                }, {
                  default: () => user.username?.[0]?.toUpperCase() || 'U'
                }),
                h('span', {
                  style: {
                    flex: 1,
                    fontSize: '14px',
                    color: 'var(--el-text-color-primary)'
                  }
                }, user.username),
                user.nickname ? h('span', {
                  style: {
                    fontSize: '12px',
                    color: 'var(--el-text-color-secondary)'
                  }
                }, `(${user.nickname})`) : null
              ])
            }
          })
        })
      }
    })
  }

  /**
   * 🔥 渲染表格单元格（用于 TableRenderer）
   * @param value 字段值
   * @param userInfoMap 用户信息映射（可选，用于批量查询优化）
   */
  renderTableCell(value?: FieldValue, userInfoMap?: Map<string, UserInfo>): any {
    const fieldValue = value || this.safeGetValue()
    const username = fieldValue?.raw ? String(fieldValue.raw) : null
    
    if (!username) {
      return h('span', '-')
    }
    
    // 🔥 优先从 userInfoMap 中获取（批量查询优化）
    let user: UserInfo | null = null
    if (userInfoMap && userInfoMap.has(username)) {
      user = userInfoMap.get(username)!
    } else if (fieldValue?.meta?.userInfo) {
      // 如果没有 userInfoMap，尝试从 meta 中获取
      user = fieldValue.meta.userInfo as UserInfo
    }
    
    // 如果有用户信息，显示头像和名称（点击头像显示弹窗，点击名称复制）
    if (user) {
      // 显示格式：username(昵称) 或 username
      const displayName = user.nickname ? `${user.username}(${user.nickname})` : user.username
      const copyText = displayName
      
      // 复制名称
      const handleCopyName = (e: Event) => {
        e.stopPropagation() // 阻止事件冒泡
        navigator.clipboard.writeText(copyText).then(() => {
          ElMessage.success('已复制名称')
        }).catch(() => {
          ElMessage.error('复制失败')
        })
      }
      
      // 复制用户信息
      const handleCopyUserInfo = (e: Event) => {
        e.stopPropagation()
        navigator.clipboard.writeText(copyText).then(() => {
          ElMessage.success('已复制用户信息')
        }).catch(() => {
          ElMessage.error('复制失败')
        })
      }
      
      return h('div', {
        class: 'user-cell',
        style: {
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          position: 'relative'
        }
      }, [
        h(ElPopover, {
          placement: 'top',
          width: 280,
          trigger: 'click',
          popperClass: 'user-info-popover',
          teleported: true
        }, {
          reference: () => h(ElAvatar, {
            src: user.avatar,
            size: 24,
            style: {
              cursor: 'pointer',
              transition: 'all 0.2s'
            },
            onMouseenter: (e: MouseEvent) => {
              const target = e.currentTarget as HTMLElement
              target.style.opacity = '0.8'
              target.style.transform = 'scale(1.05)'
            },
            onMouseleave: (e: MouseEvent) => {
              const target = e.currentTarget as HTMLElement
              target.style.opacity = '1'
              target.style.transform = 'scale(1)'
            }
          }, {
            default: () => user.username?.[0]?.toUpperCase() || 'U'
          }),
          default: () => h('div', {
            class: 'user-info-card',
            style: {
              padding: '0'
            }
          }, [
        h('div', {
          class: 'user-card-header',
          style: {
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
            padding: '16px',
            borderBottom: '1px solid var(--el-border-color-lighter)'
          }
        }, [
          h(ElAvatar, {
            src: user.avatar,
            size: 48
          }, {
            default: () => user.username?.[0]?.toUpperCase() || 'U'
          }),
          h('div', {
            style: {
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              gap: '4px'
            }
          }, [
            h('div', {
              style: {
                fontSize: '16px',
                fontWeight: 500,
                color: 'var(--el-text-color-primary)'
              }
            }, displayName),
            h('div', {
              style: {
                fontSize: '12px',
                color: 'var(--el-text-color-secondary)'
              }
            }, `@${user.username}`)
          ])
        ]),
        h('div', {
          class: 'user-card-content',
          style: {
            padding: '12px 16px'
          }
        }, [
          user.email ? h('div', {
            style: {
              display: 'flex',
              alignItems: 'center',
              marginBottom: '8px',
              fontSize: '14px'
            }
          }, [
            h('span', {
              style: {
                color: 'var(--el-text-color-secondary)',
                marginRight: '8px',
                minWidth: '60px'
              }
            }, '邮箱：'),
            h('span', {
              style: {
                color: 'var(--el-text-color-primary)',
                flex: 1,
                wordBreak: 'break-all'
              }
            }, user.email)
          ]) : null,
          user.nickname ? h('div', {
            style: {
              display: 'flex',
              alignItems: 'center',
              marginBottom: '8px',
              fontSize: '14px'
            }
          }, [
            h('span', {
              style: {
                color: 'var(--el-text-color-secondary)',
                marginRight: '8px',
                minWidth: '60px'
              }
            }, '昵称：'),
            h('span', {
              style: {
                color: 'var(--el-text-color-primary)',
                flex: 1
              }
            }, user.nickname)
          ]) : null,
          user.signature ? h('div', {
            style: {
              display: 'flex',
              alignItems: 'flex-start',
              marginBottom: '8px',
              fontSize: '14px'
            }
          }, [
            h('span', {
              style: {
                color: 'var(--el-text-color-secondary)',
                marginRight: '8px',
                minWidth: '60px',
                flexShrink: 0
              }
            }, '签名：'),
            h('span', {
              style: {
                color: 'var(--el-text-color-primary)',
                flex: 1,
                wordBreak: 'break-word',
                whiteSpace: 'pre-wrap',
                lineHeight: '1.5'
              }
            }, user.signature)
          ]) : null,
          h('div', {
            style: {
              display: 'flex',
              alignItems: 'center',
              marginBottom: '8px',
              fontSize: '14px'
            }
          }, [
            h('span', {
              style: {
                color: 'var(--el-text-color-secondary)',
                marginRight: '8px',
                minWidth: '60px'
              }
            }, '用户名：'),
            h('span', {
              style: {
                color: 'var(--el-text-color-primary)',
                flex: 1
              }
            }, user.username)
          ])
        ]),
        h('div', {
          class: 'user-card-footer',
          style: {
            padding: '12px 16px',
            borderTop: '1px solid var(--el-border-color-lighter)',
            textAlign: 'center'
          }
        }, [
          h(ElButton, {
            size: 'small',
            type: 'primary',
            onClick: handleCopyUserInfo
          }, {
            default: () => '点击复制'
          })
        ])
          ])
        }),
        h('span', {
          style: {
            fontSize: '14px',
            color: 'var(--el-color-primary)',
            cursor: 'pointer',
            userSelect: 'none',
            transition: 'all 0.2s'
          },
          onClick: handleCopyName,
          onMouseenter: (e: MouseEvent) => {
            const target = e.currentTarget as HTMLElement
            target.style.textDecoration = 'underline'
            target.style.opacity = '0.8'
          },
          onMouseleave: (e: MouseEvent) => {
            const target = e.currentTarget as HTMLElement
            target.style.textDecoration = 'none'
            target.style.opacity = '1'
          }
        }, displayName)
      ])
    }
    
    // 如果没有用户信息，显示用户名（fallback）
    return h('span', username)
  }

  /**
   * 🔥 渲染详情展示（用于 TableRenderer 详情抽屉）
   * @param value 字段值
   * @param context 上下文信息（包含 userInfoMap）
   */
  renderForDetail(value?: FieldValue, context?: { functionName?: string; recordId?: string | number; userInfoMap?: Map<string, UserInfo> }): any {
    const fieldValue = value || this.safeGetValue()
    const username = fieldValue?.raw ? String(fieldValue.raw) : null
    
    if (!username) {
      return h('span', '-')
    }
    
    // 🔥 优先从 userInfoMap 中获取（批量查询优化）
    let user: UserInfo | null = null
    if (context?.userInfoMap && context.userInfoMap.has(username)) {
      user = context.userInfoMap.get(username)!
    } else if (fieldValue?.meta?.userInfo) {
      // 如果没有 userInfoMap，尝试从 meta 中获取
      user = fieldValue.meta.userInfo as UserInfo
    }
    
    // 如果有用户信息，显示完整信息（hover 显示提示，点击复制）
    if (user) {
      const displayName = user.nickname ? `${user.username}(${user.nickname})` : user.username
      const copyText = displayName
      
      // 复制用户信息
      const handleCopy = (e: Event) => {
        e.stopPropagation() // 阻止事件冒泡
        navigator.clipboard.writeText(copyText).then(() => {
          ElMessage.success('已复制用户信息')
        }).catch(() => {
          ElMessage.error('复制失败')
        })
      }
      
      return h('div', {
        class: 'user-detail',
        style: {
          display: 'flex',
          alignItems: 'flex-start',
          gap: '16px',
          cursor: 'pointer',
          userSelect: 'none',
          transition: 'all 0.2s'
        },
        title: `点击复制：${copyText}\n邮箱：${user.email || '无'}\n昵称：${user.nickname || '无'}`,
        onClick: handleCopy,
        onMouseenter: (e: MouseEvent) => {
          const target = e.currentTarget as HTMLElement
          target.style.opacity = '0.8'
          target.style.transform = 'translateY(-1px)'
        },
        onMouseleave: (e: MouseEvent) => {
          const target = e.currentTarget as HTMLElement
          target.style.opacity = '1'
          target.style.transform = 'translateY(0)'
        }
      }, [
        h(ElAvatar, {
          src: user.avatar,
          size: 48
        }, {
          default: () => user.username?.[0]?.toUpperCase() || 'U'
        }),
        h('div', {
          style: {
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            gap: '4px'
          }
        }, [
          h('div', {
            style: {
              fontSize: '16px',
              fontWeight: 500,
              color: 'var(--el-text-color-primary)'
            }
          }, displayName),
          user.email ? h('div', {
            style: {
              fontSize: '12px',
              color: 'var(--el-text-color-secondary)'
            }
          }, user.email) : null
        ])
      ])
    }
    
    // 如果没有用户信息，显示用户名（fallback）
    return h('span', username)
  }

  /**
   * 获取提交时的原始值
   */
  getRawValueForSubmit(): any {
    const value = this.safeGetValue()
    return value?.raw || null
  }

  /**
   * 从原始数据加载值
   */
  static loadFromRawData(rawValue: any, field: FieldConfig): FieldValue {
    if (rawValue === null || rawValue === undefined || rawValue === '') {
      return {
        raw: null,
        display: '',
        meta: {}
      }
    }
    
    return {
      raw: String(rawValue),
      display: String(rawValue),
      meta: {}
    }
  }
}

