import type { EChartsType } from 'echarts/core'

type EChartsRuntime = typeof import('echarts')
type EChartsInitOptions = Parameters<EChartsRuntime['init']>[2]

let echartsRuntimePromise: Promise<EChartsRuntime> | null = null

export async function loadEChartsRuntime(_chartType: string): Promise<{
  init: (el: HTMLElement, theme?: string | object | null, opts?: EChartsInitOptions) => EChartsType
}> {
  if (!echartsRuntimePromise) {
    echartsRuntimePromise = import('echarts')
  }

  const runtime = await echartsRuntimePromise
  return {
    init: (el, theme, opts) => runtime.init(el, theme, opts) as unknown as EChartsType,
  }
}
