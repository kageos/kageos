export function tableRecycleFeedback(action: '恢复' | '彻底删除', count: unknown, requested: number) {
  if (typeof count !== 'number' || !Number.isInteger(count) || count < 0 || count > requested) {
    return { type: 'warning' as const, message: '操作已返回，但未能确认实际处理数量，请刷新列表核对。' }
  }
  if (count === 0) {
    return { type: 'warning' as const, message: `未${action}任何记录（选中 ${requested} 条），请刷新列表核对。` }
  }
  if (count < requested) {
    return { type: 'warning' as const, message: `已${action} ${count} 条，另有 ${requested - count} 条未处理，请刷新列表核对。` }
  }
  return { type: 'success' as const, message: `已${action} ${count} 条记录` }
}
