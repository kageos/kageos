import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const createPublicShareMock = vi.hoisted(() => vi.fn())
const createQrMock = vi.hoisted(() => vi.fn())

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => params
      ? `${key}:${JSON.stringify(params)}`
      : key,
  }),
}))

vi.mock('qrcode', () => ({
  default: { toDataURL: createQrMock },
}))

vi.mock('@/architecture/presentation/context/api/publicShare', () => ({
  createPublicShare: createPublicShareMock,
}))

import PublicShareCreateDialog from './PublicShareCreateDialog.vue'

const stubs = {
  ElDialog: {
    props: ['modelValue', 'title'],
    template: '<section><h2>{{ title }}</h2><slot /><footer><slot name="footer" /></footer></section>',
  },
  ElForm: { template: '<form><slot /></form>' },
  ElFormItem: { template: '<div><slot /></div>' },
  ElInput: { template: '<input />' },
  ElRadioGroup: { template: '<div><slot /></div>' },
  ElRadioButton: { template: '<button type="button"><slot /></button>' },
  ElDatePicker: { template: '<input />' },
  ElInputNumber: { template: '<input />' },
  ElAlert: { template: '<div />' },
  ElButton: { template: '<button type="button"><slot /></button>' },
  ElIcon: { template: '<i><slot /></i>' },
  ElSkeleton: { template: '<div class="skeleton" />' },
  ElEmpty: { template: '<div class="empty" />' },
  CircleCheckFilled: { template: '<span />' },
}

describe('PublicShareCreateDialog', () => {
  beforeEach(() => {
    createPublicShareMock.mockReset()
    createQrMock.mockReset()
  })

  it('keeps the dialog open and shows the link and QR code after creation', async () => {
    createPublicShareMock.mockResolvedValue({
      share_id: 'share-123',
      title: '活动报名',
      description: '请填写真实信息',
      expires_at: '2026-09-01T08:00:00.000Z',
      max_uses: 100,
      use_count: 0,
      public_url: 'https://app.kageos.com/s/share-123',
    })
    createQrMock.mockResolvedValue('data:image/png;base64,qr')

    const wrapper = mount(PublicShareCreateDialog, {
      props: {
        modelValue: false,
        fullCodePath: '/demo/activity/submit.form',
        defaultTitle: '活动报名',
        defaultDescription: '请填写真实信息',
      },
      global: { stubs },
    })

    await wrapper.setProps({ modelValue: true })
    const createButton = wrapper.findAll('button').find(button => button.text().includes('createAndGenerateQr'))
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')
    await flushPromises()

    expect(createPublicShareMock).toHaveBeenCalledWith(expect.objectContaining({
      full_code_path: '/demo/activity/submit.form',
      title: '活动报名',
      description: '请填写真实信息',
    }))
    expect(createQrMock).toHaveBeenCalledWith(
      'https://app.kageos.com/s/share-123',
      expect.objectContaining({ width: 256 })
    )
    expect(wrapper.emitted('created')).toHaveLength(1)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    expect(wrapper.text()).toContain('publicSharePanel.createdResultHint')
    expect(wrapper.text()).toContain('https://app.kageos.com/s/share-123')
    expect(wrapper.find('img.qr-image').attributes('src')).toBe('data:image/png;base64,qr')
  })
})
