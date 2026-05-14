import { SearchType, hasSearchType } from '@/architecture/domain/constants/search'
import { isStringDataType } from '@/architecture/domain/constants/widget'
import { convertArrayType } from './typeConverter'

export function buildMultiSelectRawValue(options: {
  values: string[]
  mode?: string
  dataType: string
  searchType?: string
}): any {
  const { values, mode, dataType, searchType } = options

  if (mode === 'search') {
    if (hasSearchType(searchType, SearchType.CONTAINS)) {
      return values.length > 0 ? values.join(',') : ''
    }

    return values
  }

  if (isStringDataType(dataType)) {
    return values.length > 0 ? values.join(',') : ''
  }

  return convertArrayType(values, dataType)
}
