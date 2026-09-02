import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SystemBackupPanel from './SystemBackupPanel.vue'

const backupApi = vi.hoisted(() => ({
  getSystemBackupOverview: vi.fn(),
  runSystemBackupNow: vi.fn(),
  testSystemBackupS3: vi.fn(),
  updateSystemBackupConfig: vi.fn(),
}))

vi.mock('@/architecture/presentation/context/api/system-settings', () => backupApi)
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('SystemBackupPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    backupApi.getSystemBackupOverview.mockResolvedValue({
      config: {
        enabled: true, schedule_time: '03:30', endpoint: '', region: 'us-east-1', bucket: 'backups', prefix: 'kageos',
        access_key_id: 'key', secret_access_key_set: true, use_ssl: true, force_path_style: false, keep_local: 2, retention_days: 30,
      },
      agent_available: true,
      running: false,
      records: [],
    })
  })

  it('groups execution policy, S3 destination, credentials, and actions', async () => {
    const wrapper = mount(SystemBackupPanel)
    await flushPromises()

    expect(wrapper.findAll('.backup-form-section')).toHaveLength(2)
    expect(wrapper.text()).toContain('systemSettings.dataBackup.scheduleTitle')
    expect(wrapper.text()).toContain('systemSettings.dataBackup.s3Destination')
    expect(wrapper.text()).toContain('systemSettings.dataBackup.credentialsTitle')
    expect(wrapper.find('.backup-actions').exists()).toBe(true)
  })
})
