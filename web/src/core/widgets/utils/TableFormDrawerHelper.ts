/**
 * TableFormDrawerHelper - 表格中 Form 字段详情抽屉的公共逻辑
 * 
 * 用于 TableWidget 和 ResponseTableWidget 共享 Form 字段详情抽屉功能
 * 避免代码重复，提高可维护性
 */

import { h, ref, computed, type Ref } from 'vue'
import { ElDrawer, ElButton, ElIcon } from 'element-plus'
import { View } from '@element-plus/icons-vue'
import { Logger } from '../../utils/logger'
import type { FieldConfig, FieldValue } from '../../types/field'

/**
 * Form 字段详情抽屉的状态
 */
export interface FormDrawerState {
  showFormDetailDrawer: Ref<boolean>
  formDetailField: Ref<FieldConfig | null>
  formDetailValue: Ref<FieldValue | null>
}

/**
 * 创建 Form 字段详情抽屉的状态
 */
export function createFormDrawerState(): FormDrawerState {
  return {
    showFormDetailDrawer: ref(false),
    formDetailField: ref<FieldConfig | null>(null),
    formDetailValue: ref<FieldValue | null>(null)
  }
}

/**
 * 处理 Form 字段点击（打开详情抽屉）
 */
export function handleFormFieldClick(
  state: FormDrawerState,
  field: FieldConfig,
  value: FieldValue,
  widgetName: string = 'TableWidget'
): void {
  Logger.info(`[${widgetName}]`, `点击 Form 字段: ${field.code}`)
  state.formDetailField.value = field
  state.formDetailValue.value = value
  state.showFormDetailDrawer.value = true
}

/**
 * 关闭 Form 字段详情抽屉
 */
export function handleCloseFormDetail(state: FormDrawerState): void {
  state.showFormDetailDrawer.value = false
  state.formDetailField.value = null
  state.formDetailValue.value = null
}

/**
 * 渲染 Form 字段的查看按钮（用于表格单元格）
 */
export function renderFormFieldButton(
  field: FieldConfig,
  value: FieldValue,
  onClick: (e: MouseEvent) => void
): any {
  const raw = value?.raw
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return null
  }
  
  const fieldCount = Object.keys(raw).length
  
  return h(ElButton, {
    link: true,
    type: 'primary',
    size: 'small',
    style: {
      padding: '0',
      height: 'auto',
      fontSize: '14px'
    },
    onClick: (e: MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      onClick(e)
    }
  }, {
    default: () => [
      h('span', `共 ${fieldCount} 个字段`),
      h('span', { style: { marginLeft: '4px' } }, ' '),
      h(ElIcon, {
        style: { 
          fontSize: '14px',
          verticalAlign: 'middle'
        }
      }, {
        default: () => h(View)
      })
    ]
  })
}

/**
 * 渲染 Form 字段详情抽屉的内容（抽象接口）
 * 遵循依赖倒置原则：工具类不依赖具体的 Widget 实现，而是依赖抽象
 */
export type DrawerContentRenderer = (
  field: FieldConfig,
  value: FieldValue,
  fieldPath: string
) => any

/**
 * 渲染 Form 字段详情抽屉
 * 
 * 遵循依赖倒置原则：
 * - 工具类不依赖具体的 Widget 实现（如 ResponseFormWidget）
 * - 通过 renderContent 回调函数注入具体的渲染逻辑
 * - 调用者负责提供具体的渲染实现
 * 
 * 安全措施：
 * - 添加 try-catch 防止渲染错误
 * - 确保在组件卸载时能正确清理
 */
export function renderFormDetailDrawer(
  state: FormDrawerState,
  fieldPath: string,
  renderContent: DrawerContentRenderer,
  widgetName: string = 'TableWidget'
): any {
  const show = state.showFormDetailDrawer.value
  const field = state.formDetailField.value
  const value = state.formDetailValue.value
  
  // 开发环境下输出调试日志
  if (import.meta.env.DEV) {
    Logger.info(`[${widgetName}]`, `renderFormDetailDrawer 调用: show=${show}, field=${field?.code}`)
  }
  
  if (!show || !field || !value) {
    return null
  }
  
  try {
    // 🔥 通过回调函数渲染内容，不依赖具体实现
    const content = renderContent(field, value, `${fieldPath}.${field.code}`)
    
    // 🔥 使用 key 确保 Vue 能正确追踪和清理组件
    // key 使用 fieldPath + field.code 确保唯一性
    const drawerKey = `drawer-${fieldPath}-${field.code}`
    
    return h(ElDrawer, {
      key: drawerKey,
      modelValue: show,
      title: field.name || '详细信息',
      size: '50%',
      destroyOnClose: true,
      'onUpdate:modelValue': (val: boolean) => {
        if (!val) {
          handleCloseFormDetail(state)
        }
      },
      onClose: () => {
        handleCloseFormDetail(state)
      }
    }, {
      default: () => content
    })
  } catch (error) {
    // 🔥 如果渲染出错，关闭抽屉并记录错误
    Logger.error(`[${widgetName}]`, '渲染抽屉内容失败', error)
    handleCloseFormDetail(state)
    return null
  }
}

/**
 * 创建 computed 包装的抽屉内容
 * 
 * 遵循依赖倒置原则：
 * - 通过 renderDrawer 回调函数注入具体的渲染逻辑
 * - 工具类不依赖具体的实现
 * 
 * 安全措施：
 * - 添加 try-catch 防止计算错误
 * - 确保在组件卸载时能正确清理
 */
export function createDrawerContentComputed(
  state: FormDrawerState,
  renderDrawer: () => any,
  widgetName: string = 'TableWidget'
): ReturnType<typeof computed> {
  // 🔥 使用 WeakMap 存储每个实例的日志状态，避免闭包变量共享问题
  const logStateMap = new WeakMap<FormDrawerState, { lastShow: boolean; lastFieldCode?: string }>()
  
  return computed(() => {
    const show = state.showFormDetailDrawer.value
    const field = state.formDetailField.value
    const value = state.formDetailValue.value
    
    // 🔥 只在状态真正变化时输出日志（避免频繁日志）
    const fieldCode = field?.code
    const logState = logStateMap.get(state) || { lastShow: false }
    if (import.meta.env.DEV && (show !== logState.lastShow || fieldCode !== logState.lastFieldCode)) {
      Logger.info(`[${widgetName}]`, `drawerContent computed: show=${show}, field=${fieldCode}`)
      logState.lastShow = show
      logState.lastFieldCode = fieldCode
      logStateMap.set(state, logState)
    }
    
    if (!show || !field || !value) {
      return null
    }
    
    try {
      return renderDrawer()
    } catch (error) {
      // 🔥 如果渲染出错，关闭抽屉并记录错误
      Logger.error(`[${widgetName}]`, '计算抽屉内容失败', error)
      handleCloseFormDetail(state)
      return null
    }
  })
}

