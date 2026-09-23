import { mount } from '@vue/test-utils'
import { expect, it, vi } from 'vitest'
vi.mock('vue-i18n', () => ({useI18n: () => ({locale:{value:'zh-CN'},t: (key: string) => key})}))
import ArchiveProgress from './ArchiveProgress.vue'
it('distinguishes missing progress, stale phases and finished runs', async () => {
 const now=Date.parse('2026-09-14T15:00:00+08:00')
 const wrapper=mount(ArchiveProgress,{props:{now}})
 expect(wrapper.text()).toContain('maintenance.progressUnavailable')
 const progress={execution_id:1,phase:'verifying',batch_id:8,batch_records:10000,records:20000,batches:2,failed_batches:0,stop_reason:'',started_at:'2026-09-14T14:00:00+08:00',updated_at:'2026-09-14T14:55:00+08:00'}
 await wrapper.setProps({progress})
 expect(wrapper.text()).toContain('20,000')
 expect(wrapper.text()).toContain('maintenance.progressStale')
 await wrapper.setProps({progress:{...progress,phase:'finished',finished_at:'2026-09-14T14:56:00+08:00',stop_reason:'batch_limit'}})
 expect(wrapper.text()).not.toContain('maintenance.progressStale')
 expect(wrapper.text()).toContain('maintenance.archiveStops.batch_limit')
})
