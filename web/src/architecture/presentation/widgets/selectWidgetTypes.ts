export type SelectOptionValue = string | number | boolean | Record<string, unknown>

export type SelectOptionItem = {
  label: string
  value: SelectOptionValue
  disabled?: boolean
  displayInfo?: unknown
  icon?: string
  richText?: string
  files?: string
  rich_text?: string
}
