import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import LogArchiveDetails from './LogArchiveDetails.vue'

const batch = {
  id: 156, archive_key: 'batch-156', archive_type: 'scheduled_execution', record_count: 10,
  range_started_at: '2026-09-07T14:41:04+08:00', range_ended_at: '2026-09-08T03:20:00+08:00',
  created_at: '2026-09-15T14:54:41.135+08:00', file_name: 'logs.jsonl.gz', file_size: 1024,
  sha256: 'verified-hash', status: 'completed',
} as any

describe('archive batch details', () => {
  it('preserves unknown outcomes and displays their share without treating them as success', () => {
    const wrapper = mount(LogArchiveDetails, { props: { batch: { ...batch, summary_json: {
      status_counts: { success: 7, failed: 2, interrupted: 1 },
      top_resource_paths: [{ resource_path: '/system/demos/meeting', count: 10 }],
    } } } })
    expect(wrapper.text()).toContain('interrupted')
    expect(wrapper.text()).toContain('70.0%')
    expect(wrapper.text()).toContain('10.0%')
    expect(wrapper.text()).toContain('100.0%')
    expect(wrapper.text()).toContain('verified-hash')
    expect(wrapper.text()).not.toContain('T14:54:41.135')
    wrapper.unmount()
  })
  it('marks missing historical aggregates as unavailable', () => {
    const wrapper = mount(LogArchiveDetails, { props: { batch } })
    expect(wrapper.text()).toContain('此历史批次未保存这项统计')
    expect(wrapper.text()).not.toContain('100.0%')
    wrapper.unmount()
  })
})
