import { onMounted, ref, type ComputedRef, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { executeFunction } from '@/api/function'
import type { FieldConfig, FunctionDetail } from '@/architecture/domain/types'
import type { Chart } from '@/architecture/runtime/types/chart'

interface UseChartDataLoaderOptions<TChart extends Chart> {
  functionDetail: ComputedRef<FunctionDetail>
  requestFields: ComputedRef<FieldConfig[]>
  chartData: Ref<TChart | null>
  initializeFieldValues: () => void
  watchChartData: () => void
  buildRequestParams: () => Record<string, any>
  resetFilterValues: () => void
  normalizeChartData: (chart: Chart, hasFilters: boolean) => TChart
  createPendingChart: (title: string) => TChart
}

export function useChartDataLoader<TChart extends Chart>(options: UseChartDataLoaderOptions<TChart>) {
  const loading = ref(false)

  const loadChartData = async (): Promise<void> => {
    if (!options.functionDetail.value.router || !options.functionDetail.value.method) {
      return
    }

    loading.value = true
    try {
      const params = options.buildRequestParams()
      const response = await executeFunction(
        options.functionDetail.value.method,
        options.functionDetail.value.router,
        params,
        'chart'
      )

      if (response && response.chart) {
        options.chartData.value = options.normalizeChartData(response.chart, options.requestFields.value.length > 0)
      } else {
        options.chartData.value = options.requestFields.value.length > 0
          ? options.createPendingChart(options.functionDetail.value.name || '图表')
          : null
      }
    } catch (error: any) {
      ElMessage.error(error?.message || '加载图表数据失败')
      options.chartData.value = null
    } finally {
      loading.value = false
    }
  }

  const handleSearch = (): void => {
    void loadChartData()
  }

  const handleReset = (): void => {
    options.resetFilterValues()
    options.chartData.value = options.requestFields.value.length > 0
      ? options.createPendingChart(options.functionDetail.value.name || '图表')
      : null
  }

  const handleRefresh = (): void => {
    void loadChartData()
  }

  onMounted(() => {
    options.initializeFieldValues()
    options.watchChartData()
    void loadChartData()
  })

  return {
    loading,
    loadChartData,
    handleSearch,
    handleReset,
    handleRefresh,
  }
}
