import type { EChartsType } from 'echarts/core'

type EChartsUseArgument = Parameters<(typeof import('echarts/core'))['use']>[0]
type EChartsInstaller = Exclude<EChartsUseArgument, readonly unknown[]>

type EChartsRuntime = {
  init: (typeof import('echarts/core'))['init']
  use: (typeof import('echarts/core'))['use']
}
type EChartsInitOptions = Parameters<EChartsRuntime['init']>[2]

let echartsRuntimePromise: Promise<EChartsRuntime> | null = null
let echartsBaseRegistered = false
const registeredChartTypes = new Set<string>()

async function loadChartInstaller(chartType: string): Promise<EChartsInstaller | null> {
  switch (chartType) {
    case 'bar':
      return (await import('echarts/lib/chart/bar/install.js')).install
    case 'line':
      return (await import('echarts/lib/chart/line/install.js')).install
    case 'pie':
      return (await import('echarts/lib/chart/pie/install.js')).install
    case 'gauge':
      return (await import('echarts/lib/chart/gauge/install.js')).install
    default:
      return null
  }
}

export async function loadEChartsRuntime(chartType: string): Promise<{
  init: (el: HTMLElement, theme?: string | object | null, opts?: EChartsInitOptions) => EChartsType
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
        { install: DataZoomComponent },
        { installLegacyGridContainLabel: LegacyGridContainLabel },
        { install: CanvasRenderer },
      ] = await Promise.all([
        import('echarts/core'),
        import('echarts/lib/component/title/install.js'),
        import('echarts/lib/component/tooltip/install.js'),
        import('echarts/lib/component/axisPointer/install.js'),
        import('echarts/lib/component/legend/install.js'),
        import('echarts/lib/component/grid/install.js'),
        import('echarts/lib/component/dataZoom/install.js'),
        import('echarts/lib/coord/cartesian/legacyContainLabel.js'),
        import('echarts/lib/renderer/installCanvasRenderer.js'),
      ])

      if (!echartsBaseRegistered) {
        use([
          TitleComponent,
          TooltipComponent,
          AxisPointerComponent,
          LegendComponent,
          GridComponent,
          DataZoomComponent,
          LegacyGridContainLabel,
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
      runtime.use([chartInstaller])
      registeredChartTypes.add(chartType)
    }
  }

  return {
    init: runtime.init
  }
}
