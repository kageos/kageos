import { mount } from '@vue/test-utils'
import { defineComponent, markRaw } from 'vue'
import { describe, expect, it } from 'vitest'
import type { OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'
import type { FilePanelItem } from '../composables/useMiniWorkstationPanel'
import type { MiniArtifactItem } from '../composables/useMiniWorkstationArtifacts'
import MiniWorkstationArtifactPanel from './MiniWorkstationArtifactPanel.vue'

const IconStub = defineComponent({
  template: '<span><slot /></span>',
})

const KeyInfoSectionStub = defineComponent({
  props: {
    uploadedFiles: { type: Array, default: () => [] },
    outputFiles: { type: Array, default: () => [] },
    displayFields: { type: Array, default: () => [] },
  },
  emits: ['preview-file', 'download-file', 'preview-field', 'copy-field'],
  template: `
    <div class="key-info-stub">
      <button class="preview-file" @click="$emit('preview-file', uploadedFiles[0])">preview file</button>
      <button class="download-file" @click="$emit('download-file', outputFiles[0])">download file</button>
      <button class="preview-field" @click="$emit('preview-field', displayFields[0])">preview field</button>
      <button class="copy-field" @click="$emit('copy-field', displayFields[0])">copy field</button>
    </div>
  `,
})

const file: FilePanelItem = {
  name: 'prd.md',
  href: '/files/prd.md',
  source: 'output',
}

const field: OutputDisplayField = {
  label: '用户数',
  fieldKey: 'user_count',
  value: '42',
  type: 'number',
}

const artifact: MiniArtifactItem = {
  key: 'file:/files/prd.md:0',
  name: 'prd.md',
  meta: 'MD · 输出文件',
  tag: '文档',
  ext: 'MD',
  tone: 'document',
  iconComponent: markRaw(IconStub),
  file,
}

function mountPanel(props: Record<string, unknown> = {}) {
  return mount(MiniWorkstationArtifactPanel, {
    props: {
      artifactItems: [artifact],
      maximized: false,
      panelHasContent: true,
      panelItemCount: 2,
      uploadedFiles: [file],
      outputFiles: [file],
      displayFields: [field],
      displayFieldPreviewVisible: false,
      ...props,
    },
    global: {
      stubs: {
        ElDropdown: {
          template: '<div class="dropdown-stub"><slot /><slot name="dropdown" /></div>',
        },
        ElIcon: IconStub,
        DocumentIcon: IconStub,
        MiniWorkstationKeyInfoSection: KeyInfoSectionStub,
      },
    },
  })
}

describe('MiniWorkstationArtifactPanel', () => {
  it('renders artifact summary and emits artifact selection', async () => {
    const wrapper = mountPanel()

    expect(wrapper.text()).toContain('1 项')
    expect(wrapper.text()).toContain('prd.md')
    expect(wrapper.text()).toContain('文档')

    await wrapper.find('.mini-artifact-item').trigger('click')

    expect(wrapper.emitted('artifactClick')?.[0]?.[0]).toMatchObject({ key: artifact.key })
  })

  it('forwards key information actions', async () => {
    const wrapper = mountPanel()

    await wrapper.find('.preview-file').trigger('click')
    await wrapper.find('.download-file').trigger('click')
    await wrapper.find('.preview-field').trigger('click')
    await wrapper.find('.copy-field').trigger('click')

    expect(wrapper.emitted('previewFile')?.[0]?.[0]).toMatchObject({ name: 'prd.md' })
    expect(wrapper.emitted('downloadFile')?.[0]?.[0]).toMatchObject({ name: 'prd.md' })
    expect(wrapper.emitted('previewField')?.[0]?.[0]).toMatchObject({ label: '用户数' })
    expect(wrapper.emitted('copyField')?.[0]?.[0]).toMatchObject({ label: '用户数' })
  })

  it('shows empty state without artifacts', () => {
    const wrapper = mountPanel({
      artifactItems: [],
      panelHasContent: false,
      panelItemCount: 0,
    })

    expect(wrapper.find('.mini-artifact-empty').text()).toContain('暂无产物')
    expect(wrapper.find('.mini-files-dropdown-panel').exists()).toBe(false)
  })
})
