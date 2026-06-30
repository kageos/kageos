import { nextTick, onMounted, ref, watch, type ComputedRef, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { executeFunction } from '@/architecture/presentation/context/api/function'
import type { FieldConfig, FunctionDetail } from '@/architecture/domain/types'
import type { Chart } from '@/architecture/domain/types/chart'

interface UseChartDataLoaderOptions<TChart extends Chart> {
  functionDetail: ComputedRef<FunctionDetail>
  requestFields: ComputedRef<FieldConfig[]>
  chartData: Ref<TChart | null>
  initializeFieldValues: () => void
  watchChartData: () => void
  buildRequestParams: () => Record<string, unknown>
  resetFilterValues: () => void
  normalizeChartData: (chart: Chart, hasFilters: boolean) => TChart
  createPendingChart: (title: string) => TChart
}

export function useChartDataLoader<TChart extends Chart>(options: UseChartDataLoaderOptions<TChart>) {
  const loading = ref(false)
  let loadSeq = 0
  let mounted = false
  let urlSyncWatching = false

  const loadChartData = async (): Promise<void> => {
    const detail = options.functionDetail.value
    if (!detail.router || !detail.method) {
      return
    }

    const currentSeq = ++loadSeq
    loading.value = true
    try {
      const params = options.buildRequestParams()
      const response = await executeFunction(
        detail.method,
        detail.router,
        params,
        'chart'
      )

      if (currentSeq !== loadSeq) {
        return
      }

      const responseRecord = response && typeof response === 'object'
        ? response as { chart?: Chart }
        : null
      if (responseRecord?.chart) {
        options.chartData.value = options.normalizeChartData(responseRecord.chart, options.requestFields.value.length > 0)
      } else {
        options.chartData.value = options.requestFields.value.length > 0
          ? options.createPendingChart(detail.name || '图表')
          : null
      }
    } catch (error: unknown) {
      if (currentSeq !== loadSeq) {
        return
      }
      ElMessage.error(error instanceof Error ? error.message : '加载图表数据失败')
      options.chartData.value = null
    } finally {
      if (currentSeq === loadSeq) {
        loading.value = false
      }
    }
  }

  const handleSearch = (): void => {
    void loadChartData()
  }

  const handleReset = (): void => {
    loadSeq += 1
    loading.value = false
    options.resetFilterValues()
    options.chartData.value = options.requestFields.value.length > 0
      ? options.createPendingChart(options.functionDetail.value.name || '图表')
      : null
  }

  const handleRefresh = (): void => {
    void loadChartData()
  }

  const initializeAndLoadChartData = async (): Promise<void> => {
    options.initializeFieldValues()
    if (!urlSyncWatching) {
      options.watchChartData()
      urlSyncWatching = true
    }
    await nextTick()
    void loadChartData()
  }

  onMounted(() => {
    mounted = true
    void initializeAndLoadChartData()
  })

  watch(
    () => {
      const detail = options.functionDetail.value
      return [
        detail.method || '',
        detail.router || '',
        options.requestFields.value.map((field) => field.code).join('|'),
      ].join('::')
    },
    () => {
      if (!mounted) return
      void initializeAndLoadChartData()
    }
  )

  watch(
    () => options.functionDetail.value.router,
    () => {
      loadSeq += 1
      loading.value = false
      options.chartData.value = null
    }
  )

  return {
    loading,
    loadChartData,
    handleSearch,
    handleReset,
    handleRefresh,
  }
}
