import type { EChartsType } from 'echarts/core'

type EChartsRuntime = {
  init: (typeof import('echarts/core'))['init']
  use: (typeof import('echarts/core'))['use']
}

let echartsRuntimePromise: Promise<EChartsRuntime> | null = null
let echartsBaseRegistered = false
const registeredChartTypes = new Set<string>()

async function loadChartInstaller(chartType: string): Promise<{ install: (registers: unknown) => void } | null> {
  switch (chartType) {
    case 'bar':
      return import('echarts/lib/chart/bar/install.js')
    case 'line':
      return import('echarts/lib/chart/line/install.js')
    case 'pie':
      return import('echarts/lib/chart/pie/install.js')
    case 'gauge':
      return import('echarts/lib/chart/gauge/install.js')
    default:
      return null
  }
}

export async function loadEChartsRuntime(chartType: string): Promise<{
  init: (el: HTMLElement, theme?: string | object | null, opts?: any) => EChartsType
}> {
  if (!echartsRuntimePromise) {
    echartsRuntimePromise = (async () => {
      const [
        { use, init },
        { install: TitleComponent },
        { install: TooltipComponent },
        { install: AxisPointerComponent },
        { install: LegendComponent },
        { install: GridComponent },
        { install: CanvasRenderer },
      ] = await Promise.all([
        import('echarts/core'),
        import('echarts/lib/component/title/install.js'),
        import('echarts/lib/component/tooltip/install.js'),
        import('echarts/lib/component/axisPointer/install.js'),
        import('echarts/lib/component/legend/install.js'),
        import('echarts/lib/component/grid/install.js'),
        import('echarts/lib/renderer/installCanvasRenderer.js'),
      ])

      if (!echartsBaseRegistered) {
        use([
          TitleComponent,
          TooltipComponent,
          AxisPointerComponent,
          LegendComponent,
          GridComponent,
          CanvasRenderer,
        ])
        echartsBaseRegistered = true
      }

      return { init, use }
    })()
  }

  const runtime = await echartsRuntimePromise
  if (!registeredChartTypes.has(chartType)) {
    const chartInstaller = await loadChartInstaller(chartType)
    if (chartInstaller) {
      runtime.use([chartInstaller.install])
      registeredChartTypes.add(chartType)
    }
  }

  return {
    init: runtime.init
  }
}
