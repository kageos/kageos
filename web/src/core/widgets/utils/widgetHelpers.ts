/**
 * Widget 渲染辅助函数
 * 用于统一处理常见的组件属性配置
 */

import type { ReactiveFormDataManager } from '../../managers/ReactiveFormDataManager'
import type { FormRendererContext } from '../../types/widget'

/**
 * 获取 Element Plus 表单组件的通用属性
 * 
 * @param formManager 表单数据管理器
 * @param formRenderer FormRenderer 上下文
 * @param fieldPath 字段路径
 * @returns Element Plus 组件通用属性
 */
export function getElementPlusFormProps(
  formManager: ReactiveFormDataManager | null,
  formRenderer: FormRendererContext | null,
  fieldPath: string
): Record<string, any> {
  const props: Record<string, any> = {
    // 🔥 禁用 Element Plus 的原生验证（使用我们的自定义验证系统）
    validateEvent: false
  }
  
  // 🔥 失去焦点时触发验证（通过 formManager.setValue 触发字段变化事件）
  if (formManager && formRenderer) {
    props.onBlur = () => {
      // 获取当前值并触发字段变化事件，formRenderer 会监听并验证
      const currentValue = formManager.getValue(fieldPath)
      formManager.setValue(fieldPath, currentValue)
    }
  }
  
  return props
}

