import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ElMessage, ElMessageBox } from 'element-plus'
import OutputFilesDisplay from './OutputFilesDisplay.vue'

const storageApiMocks = vi.hoisted(() => ({
  resolveFileRefs: vi.fn(),
  deleteFileRefs: vi.fn(),
}))

vi.mock('@/architecture/presentation/context/api/storage', () => ({
  resolveFileRefs: storageApiMocks.resolveFileRefs,
  deleteFileRefs: storageApiMocks.deleteFileRefs,
}))

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
    download: { type: String, default: '' },
  },
  template: '<a class="el-link" :href="href" :download="download"><slot /></a>',
})

function mountDisplay(props: Record<string, unknown>, attachTo?: HTMLElement) {
  return mount(OutputFilesDisplay, {
    props,
    attachTo,
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
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    storageApiMocks.resolveFileRefs.mockReset()
    storageApiMocks.deleteFileRefs.mockReset()
  })

  it('downloads the file blob using the Python output display name', async () => {
    const blob = new Blob(['workbook'])
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, blob: vi.fn().mockResolvedValue(blob) })
    vi.stubGlobal('fetch', fetchMock)
    const createObjectURL = vi.fn().mockReturnValue('blob:download')
    const revokeObjectURL = vi.fn()
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: createObjectURL })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: revokeObjectURL })
    let downloadedName = ''
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      downloadedName = this.download
    })

    const wrapper = mountDisplay({
      fileGroups: [{
        label: 'Output Files',
        files: [{
          name: 'generated-12345.xlsx',
          source_name: '商品导入模板_已填充.xlsx',
          download_url: '/files/generated-12345.xlsx',
        }],
      }],
    })

    await wrapper.get('.output-files-download').trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/files/generated-12345.xlsx'))
    expect(createObjectURL).toHaveBeenCalledWith(blob)
    expect(downloadedName).toBe('商品导入模板_已填充.xlsx')
  })

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

  it('renders images as full-width media cards that open the original file', () => {
    const wrapper = mountDisplay({
      fileGroups: [{
        label: 'Output Files',
        files: [{
          name: 'dashboard.png',
          source_name: 'dashboard.png',
          download_url: '/files/dashboard.png',
          thumbnail_url: '/files/dashboard-preview.webp',
          content_type: 'image/png',
        }],
      }],
    })

    expect(wrapper.get('.output-files-item').classes()).toContain('output-files-item--media')
    const previewLink = wrapper.get('a.output-files-preview-link')
    expect(previewLink.attributes('href')).toContain('/files/dashboard.png')
    expect(previewLink.attributes('target')).toBe('_blank')
    expect(wrapper.get('img.output-files-img').attributes('src')).toContain('/files/dashboard-preview.webp')
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

  it('deletes an owned workbench output and keeps a visible tombstone', async () => {
    storageApiMocks.resolveFileRefs.mockResolvedValue([{
      ref: 'kageos/alice/report.xlsx',
      bucket: 'kageos',
      key: 'alice/report.xlsx',
      name: 'report.xlsx',
      source_name: '经营分析.xlsx',
      size: 2048,
      status: 'completed',
      can_delete: true,
      download_url: '/kageos/alice/report.xlsx',
    }])
    storageApiMocks.deleteFileRefs.mockResolvedValue({
      results: [{
        ref: 'kageos/alice/report.xlsx',
        status: 'deleted',
        released_bytes: 2048,
        deleted_at: Date.now(),
      }],
      deleted_count: 1,
      released_bytes: 2048,
    })
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue(undefined as never)
    vi.spyOn(ElMessage, 'success').mockImplementation(() => undefined as never)

    const workstation = document.createElement('div')
    workstation.dataset.testid = 'mini-workstation'
    const wrapper = mountDisplay({
      deletable: true,
      fileGroups: [{
        label: 'Output Files',
        files: [{ ref: 'kageos/alice/report.xlsx', name: 'report.xlsx' }],
      }],
    }, workstation)
    await flushPromises()

    expect(wrapper.text()).toContain('共 1 个 · 2.0 KB')
    await wrapper.get('.output-files-delete').trigger('click')
    await flushPromises()

    expect(storageApiMocks.deleteFileRefs).toHaveBeenCalledWith(['kageos/alice/report.xlsx'])
    expect(confirmSpy).toHaveBeenCalledWith(
      expect.any(String),
      expect.any(String),
      expect.objectContaining({ appendTo: workstation })
    )
    expect(wrapper.find('.output-files-delete').exists()).toBe(false)
    expect(wrapper.text()).toContain('经营分析.xlsx')
    expect(wrapper.text()).toContain('已删除')
    expect(wrapper.text()).toContain('已释放空间')
    wrapper.unmount()
  })

  it('does not expose delete controls when the backend denies ownership', async () => {
    storageApiMocks.resolveFileRefs.mockResolvedValue([{
      ref: 'kageos/alice/private.pdf',
      bucket: 'kageos',
      key: 'alice/private.pdf',
      name: 'private.pdf',
      size: 1024,
      status: 'completed',
      can_delete: false,
      download_url: '/kageos/alice/private.pdf',
    }])

    const wrapper = mountDisplay({
      deletable: true,
      fileGroups: [{
        label: 'Output Files',
        files: [{ ref: 'kageos/alice/private.pdf', name: 'private.pdf' }],
      }],
    })
    await flushPromises()

    expect(wrapper.find('.output-files-delete').exists()).toBe(false)
  })
})
