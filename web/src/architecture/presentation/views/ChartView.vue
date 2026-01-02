<!--
  ChartView - 图表视图
  新架构的展示层组件
  
  职责：
  - 纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
  - 支持筛选表单和图表渲染
-->

<template>
  <div class="chart-view">
    <ChartRenderer
      :function-detail="props.functionDetail"
    />
  </div>
</template>

<script setup lang="ts">
import ChartRenderer from '@/components/ChartRenderer.vue'
import type { FunctionDetail } from '@/architecture/domain/types'

const props = defineProps<{
  functionDetail: FunctionDetail
}>()
</script>

<style scoped lang="scss">
.chart-view {
  width: 100%;
  height: 100%;
}
</style>

