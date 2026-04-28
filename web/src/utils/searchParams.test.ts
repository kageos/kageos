import { describe, expect, it } from 'vitest'
import {
  getSearchOperatorFieldValue,
  parseSearchOperatorParams
} from './searchParams'

describe('searchParams parser', () => {
  it('parses operator params by known field boundaries instead of plain comma split', () => {
    expect(
      parseSearchOperatorParams('title:春风,又绿江南岸,author:王安石', ['title', 'author'])
    ).toEqual({
      title: '春风,又绿江南岸',
      author: '王安石'
    })
  })

  it('handles field codes with shared prefixes', () => {
    expect(
      parseSearchOperatorParams('style:律诗,style_id:7', ['style', 'style_id'])
    ).toEqual({
      style: '律诗',
      style_id: '7'
    })
  })

  it('returns a single field value for restore callers', () => {
    expect(
      getSearchOperatorFieldValue('title:春风,又绿江南岸,author:王安石', 'title', ['title', 'author'])
    ).toBe('春风,又绿江南岸')
  })
})
