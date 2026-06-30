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
  let observedElement: HTMLElement | null = null
  let lastChartDataHash: string | null = null
  let windowResizeRegistered = false
  let renderedChartType: string | null = null
  let hasRenderedOption = false
  let resizeFrame: number | null = null
  let renderVersion = 0
  let lastRenderedWidth = 0
  let lastRenderedHeight = 0
  let isRendering = false

  const getContainerSize = () => {
    const element = options.chartContainerRef.value
    if (!element) {
      return { width: 0, height: 0 }
    }

    return {
      width: Math.round(element.clientWidth),
      height: Math.round(element.clientHeight),
    }
  }

  const rememberContainerSize = () => {
    const { width, height } = getContainerSize()
    lastRenderedWidth = width
    lastRenderedHeight = height
  }

  const hasContainerSizeChanged = () => {
    const { width, height } = getContainerSize()
    return width > 0
      && height > 0
      && (width !== lastRenderedWidth || height !== lastRenderedHeight)
  }

  const handleResize = () => {
    const instance = chartInstance.value
    const chartType = options.chartData.value?.chart_type || null
    if (
      isRendering
      || !instance
      || !hasRenderedOption
      || renderedChartType !== chartType
      || instance.getDom() !== options.chartContainerRef.value
      || !hasContainerSizeChanged()
    ) {
      return
    }
    instance.resize()
    if (chartInstance.value === instance) {
      rememberContainerSize()
    }
  }

  const cancelScheduledResize = () => {
    if (resizeFrame !== null && typeof window !== 'undefined' && typeof window.cancelAnimationFrame === 'function') {
      window.cancelAnimationFrame(resizeFrame)
    }
    resizeFrame = null
  }

  const scheduleResize = () => {
    if (typeof window === 'undefined') {
      handleResize()
      return
    }

    cancelScheduledResize()
    const scheduledRenderVersion = renderVersion
    const scheduledInstance = chartInstance.value
    resizeFrame = window.requestAnimationFrame(() => {
      resizeFrame = null
      if (scheduledRenderVersion !== renderVersion || scheduledInstance !== chartInstance.value) {
        return
      }
      handleResize()
    })
  }

  const ensureWindowResizeListener = () => {
    if (windowResizeRegistered || typeof window === 'undefined') {
      return
    }
    window.addEventListener('resize', scheduleResize)
    windowResizeRegistered = true
  }

  const removeWindowResizeListener = () => {
    if (!windowResizeRegistered || typeof window === 'undefined') {
      return
    }
    window.removeEventListener('resize', scheduleResize)
    windowResizeRegistered = false
  }

  const observeChartContainer = () => {
    const nextElement = options.chartContainerRef.value
    if (observedElement === nextElement) {
      return
    }

    if (resizeObserver && observedElement) {
      resizeObserver.unobserve(observedElement)
    }
    observedElement = nextElement

    if (!nextElement || typeof ResizeObserver === 'undefined') {
      return
    }
    if (!resizeObserver) {
      resizeObserver = new ResizeObserver(() => {
        scheduleResize()
      })
    }
    resizeObserver.observe(nextElement)
  }

  const disconnectResizeObserver = () => {
    if (resizeObserver) {
      if (observedElement) {
        resizeObserver.unobserve(observedElement)
      }
      resizeObserver.disconnect()
      resizeObserver = null
    }
    observedElement = null
  }

  const renderChart = async () => {
    if (!options.chartContainerRef.value || !options.chartData.value) return

    const currentRenderVersion = ++renderVersion
    const chartType = options.chartData.value.chart_type
    isRendering = true
    hasRenderedOption = false
    cancelScheduledResize()
    try {
      const { init } = await options.loadChartRuntime(chartType)
      if (currentRenderVersion !== renderVersion) {
        return
      }
      if (!options.chartContainerRef.value || !options.chartData.value) {
        return
      }
      if (options.chartData.value.chart_type !== chartType) {
        await renderChart()
        return
      }

      observeChartContainer()

      const needRecreate = !chartInstance.value
        || chartInstance.value.getDom() !== options.chartContainerRef.value
        || renderedChartType !== chartType

      if (needRecreate) {
        chartInstance.value?.dispose()
        hasRenderedOption = false
        renderedChartType = null
        lastRenderedWidth = 0
        lastRenderedHeight = 0
        chartInstance.value = init(options.chartContainerRef.value, null, {
          renderer: 'canvas',
          useDirtyRect: false
        })
      }

      const option = options.buildOption(options.chartData.value)
      if (!option || Object.keys(option).length === 0) {
        chartInstance.value?.dispose()
        chartInstance.value = null
        hasRenderedOption = false
        renderedChartType = null
        lastRenderedWidth = 0
        lastRenderedHeight = 0
        return
      }

      chartInstance.value?.clear()
      chartInstance.value?.setOption(option, {
        notMerge: true,
        lazyUpdate: false,
      })
      hasRenderedOption = true
      renderedChartType = chartType
      rememberContainerSize()
      ensureWindowResizeListener()
    } finally {
      if (currentRenderVersion === renderVersion) {
        isRendering = false
      }
    }
  }

  onMounted(() => {
    nextTick(() => {
      observeChartContainer()
    })
  })

  onUnmounted(() => {
    cancelScheduledResize()
    chartInstance.value?.dispose()
    chartInstance.value = null
    hasRenderedOption = false
    isRendering = false
    renderedChartType = null
    lastRenderedWidth = 0
    lastRenderedHeight = 0
    removeWindowResizeListener()
    disconnectResizeObserver()
  })

  watch(
    () => options.chartContainerRef.value,
    (newElement) => {
      observeChartContainer()
      if (!newElement || !options.chartData.value) {
        return
      }
      nextTick(() => {
        void renderChart()
      })
    },
    { flush: 'post' }
  )

  watch(
    () => options.chartData.value,
    (newData) => {
      if (!newData) {
        cancelScheduledResize()
        chartInstance.value?.dispose()
        chartInstance.value = null
        hasRenderedOption = false
        isRendering = false
        renderedChartType = null
        lastRenderedWidth = 0
        lastRenderedHeight = 0
        lastChartDataHash = null
        removeWindowResizeListener()
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
