import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it } from 'vitest'
import OutputFilesDisplay from './OutputFilesDisplay.vue'

const IconStub = defineComponent({
  template: '<span><slot /></span>',
})

const ButtonStub = defineComponent({
  emits: ['click'],
  template: '<button type="button" class="el-button" @click="$emit(\'click\')"><slot /></button>',
})

const LinkStub = defineComponent({
  props: {
    href: { type: String, default: '' },
  },
  template: '<a class="el-link" :href="href"><slot /></a>',
})

function mountDisplay(props: Record<string, unknown>) {
  return mount(OutputFilesDisplay, {
    props,
    global: {
      stubs: {
        ElButton: ButtonStub,
        ElIcon: IconStub,
        ElLink: LinkStub,
      },
    },
  })
}

describe('OutputFilesDisplay', () => {
  it('renders videos as inline browser previews', () => {
    const wrapper = mountDisplay({
      fileGroups: [{
        label: 'Output Files',
        files: [{
          name: 'clip.mp4',
          source_name: 'clip.mp4',
          download_url: '/files/clip.mp4',
          thumbnail_url: '/files/clip-poster.webp',
          content_type: 'video/mp4',
          preview_kind: 'video',
        }],
      }],
    })

    const video = wrapper.get('video.output-files-video')
    expect(video.attributes('poster')).toContain('/files/clip-poster.webp')
    expect(wrapper.get('video.output-files-video source').attributes('src')).toContain('/files/clip.mp4')
    expect(wrapper.get('video.output-files-video source').attributes('type')).toBe('video/mp4')
  })

  it('collapses long file groups and expands them on demand', async () => {
    const files = Array.from({ length: 6 }, (_, index) => ({
      name: `image-${index + 1}.png`,
      source_name: `image-${index + 1}.png`,
      download_url: `/files/image-${index + 1}.png`,
      content_type: 'image/png',
    }))
    const wrapper = mountDisplay({
      fileGroups: [{ label: 'Images', files }],
    })

    expect(wrapper.findAll('.output-files-item')).toHaveLength(4)
    expect(wrapper.get('.output-files-collapse-btn').text()).toContain('展开全部 6 个文件')

    await wrapper.get('.output-files-collapse-btn').trigger('click')

    expect(wrapper.findAll('.output-files-item')).toHaveLength(6)
    expect(wrapper.get('.output-files-collapse-btn').text()).toContain('收起')
  })
})
