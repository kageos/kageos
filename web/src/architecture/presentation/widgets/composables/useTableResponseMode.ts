/**
 * useTableResponseMode - TableWidget 响应模式组合式函数
 * 🔥 统一架构组件
 */

import { ref } from 'vue'

export function useTableResponseMode() {
  // 详情抽屉状态
  const showDetailDrawer = ref(false)
  const currentDetailRow = ref<any>(null)
  const currentDetailIndex = ref<number>(-1)
  
  // 显示详情
  function showDetail(row: any, index: number): void {
    currentDetailRow.value = row
    currentDetailIndex.value = index
    showDetailDrawer.value = true
  }
  
  // 关闭详情
  function closeDetail(): void {
    showDetailDrawer.value = false
    currentDetailRow.value = null
    currentDetailIndex.value = -1
  }
  
  // 导航（上一个/下一个）
  function navigate(direction: 'prev' | 'next', allRows: any[]): void {
    if (allRows.length === 0) return
    
    if (direction === 'prev' && currentDetailIndex.value > 0) {
      currentDetailIndex.value--
      currentDetailRow.value = allRows[currentDetailIndex.value]
    } else if (direction === 'next' && currentDetailIndex.value < allRows.length - 1) {
      currentDetailIndex.value++
      currentDetailRow.value = allRows[currentDetailIndex.value]
    }
  }
  
  return {
    showDetailDrawer,
    currentDetailRow,
    currentDetailIndex,
    showDetail,
    closeDetail,
    navigate
  }
}

