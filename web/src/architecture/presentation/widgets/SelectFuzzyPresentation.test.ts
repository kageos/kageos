import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SelectFuzzyPresentation from './SelectFuzzyPresentation.vue'

describe('SelectFuzzyPresentation', () => {
  it('passes rich text and file refs to the existing readonly widgets', () => {
    const wrapper = mount(SelectFuzzyPresentation, {
      props: {
        richText: '<p>投票说明</p>',
        files: 'vote/cover.jpg,vote/rules.pdf',
      },
      global: {
        stubs: {
          RichTextResponseWidget: {
            name: 'RichTextResponseWidget',
            props: ['value'],
            template: '<div data-testid="rich-text">{{ value.raw }}</div>',
          },
          FilesWidget: {
            name: 'FilesWidget',
            props: ['value', 'mode'],
            template: '<div data-testid="files" :data-mode="mode">{{ value.raw }}</div>',
          },
        },
      },
    })

    expect(wrapper.get('[data-testid="rich-text"]').text()).toBe('<p>投票说明</p>')
    expect(wrapper.get('[data-testid="files"]').text()).toBe('vote/cover.jpg,vote/rules.pdf')
    expect(wrapper.get('[data-testid="files"]').attributes('data-mode')).toBe('table-cell')
  })

  it('uses compact file preview mode for dropdown candidates', () => {
    const wrapper = mount(SelectFuzzyPresentation, {
      props: {
        files: 'vote/option-a.jpg',
        compact: true,
      },
      global: {
        stubs: {
          RichTextResponseWidget: true,
          FilesWidget: {
            name: 'FilesWidget',
            props: ['mode'],
            template: '<div data-testid="files" :data-mode="mode" />',
          },
        },
      },
    })

    expect(wrapper.get('[data-testid="files"]').attributes('data-mode')).toBe('table-cell')
  })
})
