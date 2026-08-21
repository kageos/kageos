import { describe, expect, it } from 'vitest'

import { serializeQueryParams } from './queryParamsSerializer'

describe('serializeQueryParams', () => {
  it('serializes flat arrays as repeated keys for the Go form decoder', () => {
    expect(serializeQueryParams({
      target_ids: [16, 18, 15, 17],
      time_range: '最近1天',
      bucket: '按分钟'
    })).toBe(
      'target_ids=16&target_ids=18&target_ids=15&target_ids=17&time_range=%E6%9C%80%E8%BF%911%E5%A4%A9&bucket=%E6%8C%89%E5%88%86%E9%92%9F'
    )
  })
})
