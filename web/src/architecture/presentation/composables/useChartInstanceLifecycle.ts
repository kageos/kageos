import { nextTick, onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import type { EChartsType, EChartsCoreOption } from 'echarts/core'

interface UseChartInstanceLifecycleOptions<TChart> {
  chartData: Ref<TChart | null>
  chartContainerRef: Ref<HTMLElement | null>
  loadChartRuntime: (chartType: string) => Promise<{ init: (el: HTMLElement, theme?: string | object | null, opts?: any) => EChartsType }>
  buildOption: (chart: TChart) => EChartsCoreOption
  isDebugEnabled?: () => boolean
}

export function useChartInstanceLifecycle<TChart extends { chart_type: string }>(
  options: UseChartInstanceLifecycleOptions<TChart>
) {
  const chartInstance = ref<EChartsType | null>(null)
  let resizeObserver: ResizeObserver | null = null
  let lastChartDataHash: string | null = null

  const handleResize = () => {
    chartInstance.value?.resize()
  }

  const renderChart = async () => {
    if (!options.chartContainerRef.value || !options.chartData.value) return

    const { init } = await options.loadChartRuntime(options.chartData.value.chart_type)
    if (!options.chartContainerRef.value || !options.chartData.value) return

    const needRecreate = !chartInstance.value || chartInstance.value.getDom() !== options.chartContainerRef.value

    if (needRecreate) {
      chartInstance.value?.dispose()
      chartInstance.value = init(options.chartContainerRef.value, null, {
        renderer: 'canvas',
        useDirtyRect: false
      })
    }

    const option = options.buildOption(options.chartData.value)
    if (!option || Object.keys(option).length === 0) {
      chartInstance.value?.dispose()
      chartInstance.value = null
      return
    }

    chartInstance.value?.setOption(option)
    window.addEventListener('resize', handleResize)
  }

  onMounted(() => {
    nextTick(() => {
      if (options.chartContainerRef.value) {
        resizeObserver = new ResizeObserver(() => {
          chartInstance.value?.resize()
        })
        resizeObserver.observe(options.chartContainerRef.value)
      }
    })
  })

  onUnmounted(() => {
    chartInstance.value?.dispose()
    chartInstance.value = null
    window.removeEventListener('resize', handleResize)
    if (resizeObserver && options.chartContainerRef.value) {
      resizeObserver.unobserve(options.chartContainerRef.value)
      resizeObserver.disconnect()
      resizeObserver = null
    }
  })

  watch(
    () => options.chartData.value,
    (newData) => {
      if (!newData) {
        chartInstance.value?.dispose()
        chartInstance.value = null
        lastChartDataHash = null
        return
      }

      const currentHash = JSON.stringify(newData)
      if (currentHash === lastChartDataHash) {
        return
      }
      lastChartDataHash = currentHash

      nextTick(() => {
        void renderChart()
      })
    },
    { flush: 'post' }
  )

  return {
    chartInstance,
    renderChart,
    handleResize,
  }
}
