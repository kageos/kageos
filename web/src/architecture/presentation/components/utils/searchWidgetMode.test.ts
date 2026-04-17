import { describe, expect, it } from 'vitest'
import { WidgetType } from '@/core/constants/widget'
import {
  buildSearchWidgetField,
  adaptSearchModelValueForWidget,
  resolveWidgetTypeForSearchRenderer,
  shouldUseWidgetSearchRenderer
} from './searchWidgetMode'

describe('searchWidgetMode', () => {
  it('uses widget search mode for plain input search', () => {
    expect(shouldUseWidgetSearchRenderer({
      widgetType: WidgetType.INPUT,
      searchType: 'like',
      hasRegisteredWidget: true
    })).toBe(true)
  })

  it('maps select IN search to multiselect renderer', () => {
    expect(resolveWidgetTypeForSearchRenderer({
      widgetType: WidgetType.SELECT,
      searchType: 'in'
    })).toBe(WidgetType.MULTI_SELECT)
  })

  it('maps radio eq search to select renderer', () => {
    expect(resolveWidgetTypeForSearchRenderer({
      widgetType: WidgetType.RADIO,
      searchType: 'eq'
    })).toBe(WidgetType.SELECT)
  })

  it('maps radio in search to multiselect renderer', () => {
    expect(resolveWidgetTypeForSearchRenderer({
      widgetType: WidgetType.RADIO,
      searchType: 'in'
    })).toBe(WidgetType.MULTI_SELECT)
  })

  it('maps textarea search to compact input renderer', () => {
    expect(resolveWidgetTypeForSearchRenderer({
      widgetType: WidgetType.TEXT_AREA,
      searchType: 'like'
    })).toBe(WidgetType.INPUT)
  })

  it('maps rich text search to compact input renderer', () => {
    expect(resolveWidgetTypeForSearchRenderer({
      widgetType: WidgetType.RICH_TEXT,
      searchType: 'like'
    })).toBe(WidgetType.INPUT)
  })

  it('uses widget search mode for select fields with IN search', () => {
    expect(shouldUseWidgetSearchRenderer({
      widgetType: WidgetType.SELECT,
      searchType: 'in',
      hasRegisteredWidget: true
    })).toBe(true)
  })

  it('uses widget search mode for multiselect contains search', () => {
    expect(shouldUseWidgetSearchRenderer({
      widgetType: WidgetType.MULTI_SELECT,
      searchType: 'contains',
      hasRegisteredWidget: true
    })).toBe(true)
  })

  it('builds a cloned multiselect field for select IN search', () => {
    const field = {
      code: 'status',
      name: '状态',
      widget: {
        type: WidgetType.SELECT,
        config: {
          options: [{ label: '启用', value: 'enabled' }]
        }
      }
    }

    const searchField = buildSearchWidgetField(field as any, 'in')

    expect(searchField).not.toBe(field)
    expect(searchField.widget.type).toBe(WidgetType.MULTI_SELECT)
    expect(field.widget.type).toBe(WidgetType.SELECT)
  })

  it('keeps fallback mode for widget types without registered search widget', () => {
    expect(shouldUseWidgetSearchRenderer({
      widgetType: WidgetType.INPUT,
      searchType: 'like',
      hasRegisteredWidget: false
    })).toBe(false)
  })

  it('keeps users search on the widget renderer path', () => {
    expect(shouldUseWidgetSearchRenderer({
      widgetType: WidgetType.USERS,
      searchType: 'in',
      hasRegisteredWidget: true
    })).toBe(true)
  })

  it('keeps department search on the widget renderer path', () => {
    expect(shouldUseWidgetSearchRenderer({
      widgetType: WidgetType.DEPARTMENT,
      searchType: 'eq',
      hasRegisteredWidget: true
    })).toBe(true)
  })

  it('keeps multi-user contains search on the widget renderer path', () => {
    expect(shouldUseWidgetSearchRenderer({
      widgetType: WidgetType.USERS,
      searchType: 'contains',
      hasRegisteredWidget: true
    })).toBe(true)
  })

  it('keeps multi-department contains search on the widget renderer path', () => {
    expect(shouldUseWidgetSearchRenderer({
      widgetType: WidgetType.DEPARTMENTS,
      searchType: 'contains',
      hasRegisteredWidget: true
    })).toBe(true)
  })

  it('adapts checkbox string values into arrays for widget search mode', () => {
    expect(adaptSearchModelValueForWidget('a,b', WidgetType.CHECKBOX)).toEqual(['a', 'b'])
  })

  it('adapts departments arrays into comma separated strings for widget search mode', () => {
    expect(adaptSearchModelValueForWidget(['dept.a', 'dept.b'], WidgetType.DEPARTMENTS)).toBe('dept.a,dept.b')
  })
})
