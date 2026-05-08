import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import PrdChartPreview from './PrdChartPreview.vue'
import { loadEChartsRuntime } from './utils/chartEChartsRuntime'

const mocks = vi.hoisted(() => {
  const chart = {
    setOption: vi.fn(),
    resize: vi.fn(),
    dispose: vi.fn(),
  }
  const init = vi.fn(() => chart)
  const loadRuntime = vi.fn(async () => ({ init }))
  return { chart, init, loadRuntime }
})

vi.mock('./utils/chartEChartsRuntime', () => ({
  loadEChartsRuntime: mocks.loadRuntime,
}))

function chartRect(): DOMRect {
  return {
    x: 0,
    y: 0,
    top: 0,
    left: 0,
    bottom: 320,
    right: 520,
    width: 520,
    height: 320,
    toJSON: () => ({}),
  }
}

async function flushChartRender() {
  await flushPromises()
  await new Promise(resolve => window.setTimeout(resolve, 0))
  await flushPromises()
  await new Promise(resolve => window.setTimeout(resolve, 0))
  await flushPromises()
}

describe('PrdChartPreview', () => {
  beforeEach(() => {
    mocks.chart.setOption.mockClear()
    mocks.chart.resize.mockClear()
    mocks.chart.dispose.mockClear()
    mocks.init.mockClear()
    mocks.loadRuntime.mockClear()
    mocks.loadRuntime.mockImplementation(async () => ({ init: mocks.init }))

    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function getBoundingClientRect(this: HTMLElement) {
      if (this.classList?.contains('prd-chart-canvas')) {
        return chartRect()
      }
      return {
        x: 0,
        y: 0,
        top: 0,
        left: 0,
        bottom: 0,
        right: 0,
        width: 0,
        height: 0,
        toJSON: () => ({}),
      }
    })
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      return window.setTimeout(() => callback(Date.now()), 0)
    })
    vi.spyOn(window, 'cancelAnimationFrame').mockImplementation((id) => {
      window.clearTimeout(id)
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('initializes ECharts after the chart container has a usable size', async () => {
    const wrapper = mount(PrdChartPreview, {
      attachTo: document.body,
      props: {
        active: true,
        title: '工单统计',
        chartType: 'PieChart',
        dimension: '工单状态',
        metrics: ['工单数量'],
        previewData: [
          { name: '待处理', value: 12 },
          { name: '处理中', value: 6 },
          { name: '已完成', value: 24 },
        ],
      },
    })

    await flushChartRender()

    expect(loadEChartsRuntime).toHaveBeenCalledWith('pie')
    expect(mocks.init).toHaveBeenCalled()
    expect(mocks.chart.setOption).toHaveBeenCalled()
    const option = mocks.chart.setOption.mock.calls.at(-1)?.[0] as any
    expect(option.series[0].data).toEqual([
      { name: '待处理', value: 12 },
      { name: '处理中', value: 6 },
      { name: '已完成', value: 24 },
    ])
    expect(mocks.chart.resize).toHaveBeenCalled()
    expect(wrapper.find('.prd-chart-canvas').classes()).toContain('is-rendered')
    expect(wrapper.find('.prd-chart-fallback').exists()).toBe(false)

    wrapper.unmount()
  })

  it.each([
    ['LineChart', 'line'],
    ['BarChart', 'bar'],
  ])('uses PRD preview data for %s', async (chartType, runtimeType) => {
    const wrapper = mount(PrdChartPreview, {
      attachTo: document.body,
      props: {
        active: true,
        title: '销售统计',
        chartType,
        dimension: '日期',
        metrics: ['销售额', '订单数'],
        previewData: [
          { 日期: '2025-01-18', 销售额: 860.5, 订单数: 28 },
          { 日期: '2025-01-19', 销售额: 1024, 订单数: 34 },
        ],
      },
    })

    await flushChartRender()

    expect(loadEChartsRuntime).toHaveBeenCalledWith(runtimeType)
    const option = mocks.chart.setOption.mock.calls.at(-1)?.[0] as any
    expect(option.xAxis.data).toEqual(['2025-01-18', '2025-01-19'])
    expect(option.series[0].data).toEqual([860.5, 1024])
    expect(option.series[1].data).toEqual([28, 34])

    wrapper.unmount()
  })
})
