import { computed, nextTick, onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { WidgetType } from '@/core/constants/widget'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { createAutoFieldValue, createEmptyRawFieldValue } from '@/core/utils/createFieldValue'

interface UseTableRowDetailLayoutOptions {
  fields: Ref<FieldConfig[]>
  rowData: Ref<Record<string, any> | null>
}

const RICH_TEXT_PREVIEW_HEIGHT = 320

function getInitialLayout(): boolean {
  try {
    const stored = localStorage.getItem('useGroupedDetailLayout')
    const layoutVersion = localStorage.getItem('useGroupedDetailLayoutVersion')

    if (stored === 'true' || stored === 'false') {
      if (layoutVersion) {
        return stored === 'true'
      }
      localStorage.removeItem('useGroupedDetailLayout')
    }

    return true
  } catch {
    return true
  }
}

export function useTableRowDetailLayout({
  fields,
  rowData
}: UseTableRowDetailLayoutOptions) {
  const useGroupedDetailLayout = ref<boolean>(getInitialLayout())
  const richTextExpanded = ref<Record<string, boolean>>({})
  const richTextOverflow = ref<Record<string, boolean>>({})
  const richTextContentRefs = new Map<string, HTMLElement>()
  const richTextResizeObservers = new Map<string, ResizeObserver>()

  const linkFields = computed(() => {
    return fields.value.filter((field: FieldConfig) => field.widget?.type === WidgetType.LINK)
  })

  const groupedFields = computed(() => {
    const fieldsToGroup = fields.value.filter((field: FieldConfig) => field.widget?.type !== WidgetType.LINK)

    const idField = fieldsToGroup.find((field: FieldConfig) => field.widget?.type === WidgetType.ID)

    const statusFields = fieldsToGroup.filter((field: FieldConfig) => {
      const widgetType = field.widget?.type
      return (
        widgetType === WidgetType.SELECT ||
        widgetType === WidgetType.MULTI_SELECT ||
        widgetType === WidgetType.RADIO ||
        widgetType === WidgetType.CHECKBOX ||
        widgetType === WidgetType.SWITCH
      )
    })

    const userFields = fieldsToGroup.filter((field: FieldConfig) => field.widget?.type === WidgetType.USER)
    const timestampFields = fieldsToGroup.filter(
      (field: FieldConfig) => field.widget?.type === WidgetType.TIMESTAMP
    )
    const richTextFields = fieldsToGroup.filter(
      (field: FieldConfig) => field.widget?.type === WidgetType.RICH_TEXT
    )
    const complexFields = fieldsToGroup.filter((field: FieldConfig) => {
      const widgetType = field.widget?.type
      return widgetType === WidgetType.FORM || widgetType === WidgetType.TABLE
    })

    const mainContentFields = fieldsToGroup.filter((field: FieldConfig) => {
      const widgetType = field.widget?.type
      return (
        widgetType !== WidgetType.ID &&
        widgetType !== WidgetType.SELECT &&
        widgetType !== WidgetType.MULTI_SELECT &&
        widgetType !== WidgetType.RADIO &&
        widgetType !== WidgetType.CHECKBOX &&
        widgetType !== WidgetType.SWITCH &&
        widgetType !== WidgetType.USER &&
        widgetType !== WidgetType.TIMESTAMP &&
        widgetType !== WidgetType.FORM &&
        widgetType !== WidgetType.TABLE
      )
    })

    return {
      idField,
      statusFields,
      userFields,
      timestampFields,
      richTextFields,
      complexFields,
      mainContentFields
    }
  })

  const toggleDetailLayout = (): void => {
    useGroupedDetailLayout.value = !useGroupedDetailLayout.value
    localStorage.setItem('useGroupedDetailLayout', String(useGroupedDetailLayout.value))
    localStorage.setItem('useGroupedDetailLayoutVersion', '1.0')
  }

  const syncRichTextState = () => {
    const activeCodes = new Set(groupedFields.value.richTextFields.map((field: FieldConfig) => field.code))
    const nextExpanded: Record<string, boolean> = {}
    const nextOverflow: Record<string, boolean> = {}

    activeCodes.forEach((code) => {
      nextExpanded[code] = richTextExpanded.value[code] ?? false
      nextOverflow[code] = richTextOverflow.value[code] ?? false
    })

    const hasExpandedChanged =
      Object.keys(nextExpanded).length !== Object.keys(richTextExpanded.value).length ||
      Object.entries(nextExpanded).some(([code, value]) => richTextExpanded.value[code] !== value)
    const hasOverflowChanged =
      Object.keys(nextOverflow).length !== Object.keys(richTextOverflow.value).length ||
      Object.entries(nextOverflow).some(([code, value]) => richTextOverflow.value[code] !== value)

    if (hasExpandedChanged) {
      richTextExpanded.value = nextExpanded
    }

    if (hasOverflowChanged) {
      richTextOverflow.value = nextOverflow
    }

    Array.from(richTextResizeObservers.keys()).forEach((code) => {
      if (!activeCodes.has(code)) {
        richTextResizeObservers.get(code)?.disconnect()
        richTextResizeObservers.delete(code)
        richTextContentRefs.delete(code)
      }
    })
  }

  const measureRichTextOverflow = (fieldCode: string) => {
    const content = richTextContentRefs.get(fieldCode)
    if (!content) {
      return
    }

    const contentHeight = Math.ceil(content.getBoundingClientRect().height || content.scrollHeight || 0)
    const hasOverflow = contentHeight > RICH_TEXT_PREVIEW_HEIGHT + 8

    if (richTextOverflow.value[fieldCode] !== hasOverflow) {
      richTextOverflow.value = {
        ...richTextOverflow.value,
        [fieldCode]: hasOverflow
      }
    }

    if (!hasOverflow && richTextExpanded.value[fieldCode]) {
      richTextExpanded.value = {
        ...richTextExpanded.value,
        [fieldCode]: false
      }
    }
  }

  const measureAllRichTextOverflow = () => {
    if (!useGroupedDetailLayout.value) {
      return
    }

    groupedFields.value.richTextFields.forEach((field: FieldConfig) => {
      measureRichTextOverflow(field.code)
    })
  }

  const scheduleRichTextMeasurement = () => {
    nextTick(() => {
      measureAllRichTextOverflow()
    })
  }

  const setRichTextContentRef = (fieldCode: string, el: Element | null) => {
    if (!(el instanceof HTMLElement)) {
      richTextResizeObservers.get(fieldCode)?.disconnect()
      richTextResizeObservers.delete(fieldCode)
      richTextContentRefs.delete(fieldCode)
      return
    }

    richTextContentRefs.set(fieldCode, el)

    if (typeof ResizeObserver !== 'undefined') {
      let observer = richTextResizeObservers.get(fieldCode)
      if (!observer) {
        observer = new ResizeObserver(() => {
          measureRichTextOverflow(fieldCode)
        })
        richTextResizeObservers.set(fieldCode, observer)
      }
      observer.disconnect()
      observer.observe(el)
    }

    measureRichTextOverflow(fieldCode)
  }

  const isRichTextExpanded = (fieldCode: string): boolean => !!richTextExpanded.value[fieldCode]
  const isRichTextOverflow = (fieldCode: string): boolean => !!richTextOverflow.value[fieldCode]

  const toggleRichTextExpanded = (fieldCode: string) => {
    richTextExpanded.value = {
      ...richTextExpanded.value,
      [fieldCode]: !richTextExpanded.value[fieldCode]
    }
  }

  const getFieldValue = (fieldCode: string): FieldValue => {
    if (!rowData.value) return createEmptyRawFieldValue()
    return createAutoFieldValue(rowData.value[fieldCode])
  }

  watch(
    () => [
      useGroupedDetailLayout.value,
      groupedFields.value.richTextFields.map((field: FieldConfig) => field.code).join('|'),
      rowData.value
    ],
    () => {
      syncRichTextState()
      scheduleRichTextMeasurement()
    },
    { immediate: true }
  )

  watch(
    () => rowData.value,
    () => {
      richTextExpanded.value = {}
    }
  )

  onBeforeUnmount(() => {
    richTextResizeObservers.forEach((observer) => observer.disconnect())
    richTextResizeObservers.clear()
    richTextContentRefs.clear()
  })

  return {
    RICH_TEXT_PREVIEW_HEIGHT,
    useGroupedDetailLayout,
    toggleDetailLayout,
    linkFields,
    groupedFields,
    setRichTextContentRef,
    isRichTextExpanded,
    isRichTextOverflow,
    toggleRichTextExpanded,
    getFieldValue
  }
}
