<template>
  <div class="workspace-function-renderer">
    <FunctionConnectorBar
      v-if="showConnectorBar"
      :current-function="currentFunction"
      :function-detail="functionDetail"
    />

    <div class="function-runtime">
      <template v-if="matchedFunctionDetail">
        <FormView
          v-if="functionDetail?.template_type === TEMPLATE_TYPE.FORM"
          :ref="formViewRefTarget || undefined"
          :key="`form-${keyBase}`"
          :function-detail="functionDetail"
        />
        <TableView
          v-else-if="functionDetail?.template_type === TEMPLATE_TYPE.TABLE"
          :key="`table-${keyBase}`"
          :function-detail="functionDetail"
          :current-function="currentFunction"
        />
        <ChartView
          v-else-if="functionDetail?.template_type === TEMPLATE_TYPE.CHART"
          :key="`chart-${keyBase}`"
          :function-detail="functionDetail"
        />
        <div v-else :key="`empty-${keyBase}`" class="function-loading">
          <el-skeleton :rows="8" animated />
        </div>
      </template>
      <div v-else :key="`loading-${keyBase}`" class="function-loading">
        <el-skeleton :rows="8" animated />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import type { FunctionDetail } from '@/architecture/domain/types'
import type { ServiceTree as ServiceTreeType } from '@/architecture/domain/types'
import { TEMPLATE_TYPE } from '@/architecture/domain/constants/functionTypes'
import { featureFlags } from '@/architecture/shared/config/features'
import FunctionConnectorBar from './FunctionConnectorBar.vue'

const FormView = defineAsyncComponent(() => import('@/architecture/presentation/views/FormView.vue'))
const TableView = defineAsyncComponent(() => import('@/architecture/presentation/views/TableView.vue'))
const ChartView = defineAsyncComponent(() => import('@/architecture/presentation/views/ChartView.vue'))

const props = withDefaults(defineProps<{
  currentFunction: ServiceTreeType | null
  functionDetail: FunctionDetail | null
  formViewRefTarget?: any
  showConnectorBar?: boolean
}>(), {
  showConnectorBar: true
})

const keyBase = computed(() => props.currentFunction?.full_code_path || props.currentFunction?.id || 'unknown')
const showConnectorBar = computed(() => featureFlags.connectorSettings && props.showConnectorBar)

const matchedFunctionDetail = computed(() => {
  if (!props.functionDetail || !props.currentFunction) {
    return false
  }

  return (
    props.functionDetail.id === props.currentFunction.ref_id ||
    props.functionDetail.router === props.currentFunction.full_code_path
  )
})

</script>

<style scoped lang="scss">
.workspace-function-renderer {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.function-runtime {
  flex: 1;
  min-height: 0;
}

.function-loading {
  padding: 24px;
  width: 100%;
}
</style>
