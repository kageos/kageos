/**
 * TableFormDrawerHelper - 表格中 Form 字段详情抽屉的公共逻辑
 * 
 * 用于 TableWidget 和 ResponseTableWidget 共享 Form 字段详情抽屉功能
 * 避免代码重复，提高可维护性
 */

import { h, ref, computed, type Ref } from 'vue'
import { ElDrawer, ElButton, ElIcon } from 'element-plus'
import { View } from '@element-plus/icons-vue'
import { ResponseFormWidget } from '../ResponseFormWidget'
import { Logger } from '../../utils/logger'
import type { FieldConfig, FieldValue } from '../../types/field'
import type { ReactiveFormDataManager } from '../../managers/ReactiveFormDataManager'
import type { FormRendererContext } from '../../types/widget'

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
 * 渲染 Form 字段详情抽屉
 */
export function renderFormDetailDrawer(
  state: FormDrawerState,
  fieldPath: string,
  formManager: ReactiveFormDataManager | null,
  formRenderer: FormRendererContext | null,
  depth: number,
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
  
  // 🔥 使用 ResponseFormWidget 渲染表单内容（只读模式）
  const responseWidget = new ResponseFormWidget({
    field: field,
    currentFieldPath: `${fieldPath}.${field.code}`,
    value: value,
    onChange: () => {},
    formManager: formManager,
    formRenderer: formRenderer,
    depth: depth + 1
  })
  
  return h(ElDrawer, {
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
    default: () => responseWidget.render()
  })
}

/**
 * 创建 computed 包装的抽屉内容
 */
export function createDrawerContentComputed(
  state: FormDrawerState,
  renderDrawer: () => any,
  widgetName: string = 'TableWidget'
): ReturnType<typeof computed> {
  return computed(() => {
    const show = state.showFormDetailDrawer.value
    const field = state.formDetailField.value
    const value = state.formDetailValue.value
    
    // 开发环境下输出调试日志
    if (import.meta.env.DEV) {
      Logger.info(`[${widgetName}]`, `drawerContent computed: show=${show}, field=${field?.code}`)
    }
    
    if (!show || !field || !value) {
      return null
    }
    
    return renderDrawer()
  })
}

