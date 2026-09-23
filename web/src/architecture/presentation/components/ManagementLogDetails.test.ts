import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { setLocale } from '@/architecture/shared/i18n'
import { managementChanges } from '../composables/managementLog'
import ManagementLogDetails from './ManagementLogDetails.vue'

describe('management audit detail', () => {
  it('shows only changed configuration fields with their before/after values', () => {
    const log = { action: 'timer.task.updated', status: 'success', old_values_json: { title: 'Job', interval_seconds: 3600 }, new_values_json: { title: 'Job', interval_seconds: 60 } }
    const wrapper = mount(ManagementLogDetails, { props: { log } })
    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    expect(wrapper.text()).toContain('间隔秒数')
    expect(wrapper.text()).toContain('3600')
    expect(wrapper.text()).toContain('60')
  })
  it('preserves false and zero values as changes', () => {
    expect(managementChanges({ enabled: true, max_uses: 10 }, { enabled: false, max_uses: 0 })).toHaveLength(2)
  })
  it('does not imply success for an unfinished or failed operation', () => {
    const wrapper = mount(ManagementLogDetails, { props: { log: { action: 'workspace.deleted', status: 'pending', details_json: { stage: 'deleting_runtime' } } } })
    expect(wrapper.text()).toContain('尚未记录最终结果')
    expect(wrapper.text()).toContain('删除运行时资源')
    const failed = mount(ManagementLogDetails, { props: { log: { action: 'directory.installed', status: 'failed', details_json: { stage: 'writing_files', error: '<script>bad()</script>' } } } })
    expect(failed.text()).toContain('不一定已回滚')
    expect(failed.find('script').exists()).toBe(false)
  })
  it('renders English field and stage labels', () => {
    setLocale('en-US')
    const wrapper = mount(ManagementLogDetails, { props: { log: { action: 'directory.installed', details_json: { stage: 'installing_docs', overwrite: false } } } })
    expect(wrapper.text()).toContain('Installing documents')
    expect(wrapper.text()).toContain('Allow overwrite')
    expect(wrapper.text()).toContain('No')
  })
})
