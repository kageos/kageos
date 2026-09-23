import { mount } from '@vue/test-utils'
import { expect, it, vi } from 'vitest'
vi.mock('vue-i18n', () => ({useI18n: () => ({t: (key: string) => key})}))
import ExecutionError from './ExecutionError.vue'
it('groups identical lines while preserving complete original errors', () => {
 const message = 'upload archive: clock mismatch\nupload archive: clock mismatch\nnetwork timeout'
 const wrapper = mount(ExecutionError, {props: {message}})
 expect(wrapper.findAll('.error-line')).toHaveLength(2)
 expect(wrapper.get('.error-line b').text()).toBe('× 2')
 expect(wrapper.get('pre').text()).toBe(message)
})
