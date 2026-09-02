import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SystemResourceTrendChart from './SystemResourceTrendChart.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => params?.value ? `${key} ${params.value}` : key,
  }),
}))

describe('SystemResourceTrendChart', () => {
  it('shows exact values for the nearest point on hover', async () => {
    const wrapper = mount(SystemResourceTrendChart, {
      props: {
        history: [
          resourcePoint('2026-08-30T02:30:00Z', 10, 20, 30),
          resourcePoint('2026-08-31T02:30:00Z', 40, 50, 60),
        ],
      },
    })
    const chart = wrapper.findAll('svg')[0]!
    vi.spyOn(chart.element, 'getBoundingClientRect').mockReturnValue({
      left: 0, top: 0, right: 880, bottom: 240, width: 880, height: 240,
      x: 0, y: 0, toJSON: () => ({}),
    } as DOMRect)

    await chart.trigger('mousemove', { clientX: 66 })
    expect(wrapper.find('.trend-tooltip').text()).toContain('10.0%')
    expect(wrapper.find('.trend-tooltip').text()).toContain('20.0%')
    expect(wrapper.find('.trend-tooltip').text()).toContain('30.0%')

    await chart.trigger('mousemove', { clientX: 864 })
    expect(wrapper.find('.trend-tooltip').text()).toContain('40.0%')
    expect(wrapper.find('.trend-tooltip').text()).toContain('50.0%')
    expect(wrapper.find('.trend-tooltip').text()).toContain('60.0%')
  })
})

function resourcePoint(collectedAt: string, cpu: number, memory: number, disk: number) {
  return {
    collected_at: collectedAt,
    disk_used_bytes: 0,
    disk_used_percent: disk,
    memory_used_percent: memory,
    cpu_used_percent: cpu,
    cpu_max_percent: cpu,
    network_rx_bytes_per_second: 1024,
    network_tx_bytes_per_second: 2048,
    disk_read_bytes_per_second: 4096,
    disk_write_bytes_per_second: 8192,
    load_1: 0,
  }
}
