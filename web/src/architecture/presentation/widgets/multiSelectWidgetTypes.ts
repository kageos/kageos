import type { SelectOptionValue } from './selectWidgetTypes'

export type MultiSelectOptionItem = {
  label: string
  value: SelectOptionValue
  disabled?: boolean
  displayInfo?: unknown
  icon?: string
  richText?: string
  files?: string
  rich_text?: string
}
