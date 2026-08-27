import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MiniWorkstationResourceIdentity from './MiniWorkstationResourceIdentity.vue'

describe('MiniWorkstationResourceIdentity', () => {
  it('renders the table identity from a table resource path', () => {
    const wrapper = mount(MiniWorkstationResourceIdentity, {
      props: {
        name: '客户表',
        fullCodePath: '/demo/crm/customers.table',
      },
    })

    expect(wrapper.classes()).toContain('is-table')
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.findAll('svg path')).toHaveLength(7)
    expect(wrapper.find('svg path').attributes('fill')).toBe('#553CCE')
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.text()).toContain('客户表')
  })

  it('keeps a directory identity distinct from table functions', () => {
    const wrapper = mount(MiniWorkstationResourceIdentity, {
      props: {
        name: '客户管理',
        fullCodePath: '/demo/crm',
        resourceType: 'package',
      },
    })

    expect(wrapper.classes()).toContain('is-package')
    expect(wrapper.find('img').attributes('src')).toBe('/service-tree/custom-folder.svg')
  })
})
