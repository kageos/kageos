/**
 * useTableCellMode - table-cell 模式的公共逻辑
 * 
 * 功能：
 * - 判断 table-cell 模式是在编辑上下文还是响应上下文中使用
 * - 决定抽屉中使用的渲染模式
 * - 提供抽屉状态管理
 * 
 * 设计思路：
 * - table-cell 模式本身不区分编辑/响应，它只是一个显示模式（简化显示 + 抽屉）
 * - 但是抽屉中的内容需要根据上下文决定是编辑还是只读
 * - 通过 parentMode 显式传递父级模式，避免间接判断导致的错误
 */

import { computed, ref } from 'vue'
import type { WidgetComponentProps } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'

/**
 * 判断 table-cell 模式是在编辑上下文还是响应上下文中使用
 * 
 * 判断逻辑（优先级从高到低）：
 * 1. parentMode === 'edit' → 编辑上下文
 * 2. parentMode === 'response' → 响应上下文
 * 3. 如果没有 parentMode，使用备用判断（formManager 或 formDataStore）
 * 
 * @param props Widget 组件的 props
 * @returns 是否在编辑上下文中
 */
function useEditContext(props: WidgetComponentProps) {
  return computed(() => {
    // 优先判断：如果 parentMode 是 'edit'，说明是在编辑模式中
    if (props.parentMode === 'edit') {
      return true
    }
    // 优先判断：如果 parentMode 是 'response'，说明是在响应模式中
    if (props.parentMode === 'response') {
      return false
    }
    // 备用判断：如果没有 parentMode，使用 formManager 或 formDataStore 判断
    // （这种情况应该很少出现，因为我们在调用时都会传递 parentMode）
    if (props.formManager) {
      return true
    }
    const formDataStore = useFormDataStore()
    const value = formDataStore.getValue(props.fieldPath)
    return value !== null && value !== undefined && value.raw !== null && value.raw !== undefined
  })
}

/**
 * table-cell 模式抽屉中使用的模式（根据上下文决定）
 * 
 * 预期行为：
 * - 编辑上下文：使用 edit 模式，支持编辑，显示确认按钮
 * - 响应上下文：使用 response 模式，只读展示，不显示确认按钮
 */
function useDrawerMode(isInEditContext: ReturnType<typeof useEditContext>) {
  return computed(() => {
    return isInEditContext.value ? 'edit' : 'response'
  })
}

/**
 * table-cell 模式的组合式函数
 * 
 * @param props Widget 组件的 props
 * @returns table-cell 模式相关的状态和方法
 */
export function useTableCellMode(props: WidgetComponentProps) {
  // 抽屉显示状态
  const showDrawer = ref(false)
  
  // 判断是否在编辑上下文
  const isInEditContext = useEditContext(props)
  
  // 抽屉中使用的渲染模式
  const drawerMode = useDrawerMode(isInEditContext)
  
  // 关闭抽屉
  const closeDrawer = () => {
    showDrawer.value = false
  }
  
  // 打开抽屉
  const openDrawer = () => {
    showDrawer.value = true
  }
  
  return {
    showDrawer,
    isInEditContext,
    drawerMode,
    closeDrawer,
    openDrawer
  }
}

