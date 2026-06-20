import { defineAsyncComponent, type Component } from 'vue'
import { widgetComponentFactory } from './factory'
import { WidgetType } from '@/architecture/domain/constants/widget'
import ProgressWidget from '@/architecture/presentation/widgets/ProgressWidget.vue'
import SliderWidget from '@/architecture/presentation/widgets/SliderWidget.vue'

type WidgetModuleLoader = () => Promise<{ default: Component }>

const lazyWidget = (loader: WidgetModuleLoader): Component => defineAsyncComponent({
  loader: async () => (await loader()).default,
  suspensible: false,
})

const InputWidget = lazyWidget(() => import('@/architecture/presentation/widgets/InputWidget.vue'))
const IntegerWidget = lazyWidget(() => import('@/architecture/presentation/widgets/IntegerWidget.vue'))
const FloatWidget = lazyWidget(() => import('@/architecture/presentation/widgets/FloatWidget.vue'))
const TextAreaWidget = lazyWidget(() => import('@/architecture/presentation/widgets/TextAreaWidget.vue'))
const SwitchWidget = lazyWidget(() => import('@/architecture/presentation/widgets/SwitchWidget.vue'))
const SelectWidget = lazyWidget(() => import('@/architecture/presentation/widgets/SelectWidget.vue'))
const MultiSelectWidget = lazyWidget(() => import('@/architecture/presentation/widgets/MultiSelectWidget.vue'))
const CheckboxWidget = lazyWidget(() => import('@/architecture/presentation/widgets/CheckboxWidget.vue'))
const RadioWidget = lazyWidget(() => import('@/architecture/presentation/widgets/RadioWidget.vue'))
const TextWidget = lazyWidget(() => import('@/architecture/presentation/widgets/TextWidget.vue'))
const DateTimeWidget = lazyWidget(() => import('@/architecture/presentation/widgets/DateTimeWidget.vue'))
const RateWidget = lazyWidget(() => import('@/architecture/presentation/widgets/RateWidget.vue'))
const ColorWidget = lazyWidget(() => import('@/architecture/presentation/widgets/ColorWidget.vue'))
const RichTextResponseWidget = lazyWidget(() => import('@/architecture/presentation/widgets/RichTextResponseWidget.vue'))
const FilesWidget = lazyWidget(() => import('@/architecture/presentation/widgets/FilesWidget.vue'))
const RichTextWidget = lazyWidget(() => import('@/architecture/presentation/widgets/RichTextWidget.vue'))
const UserWidget = lazyWidget(() => import('@/architecture/presentation/shared/components/UserWidget.vue'))
const UsersWidget = lazyWidget(() => import('@/architecture/presentation/shared/components/UsersWidget.vue'))
const DepartmentWidget = lazyWidget(() => import('@/architecture/presentation/widgets/DepartmentWidget.vue'))
const DepartmentsWidget = lazyWidget(() => import('@/architecture/presentation/shared/components/DepartmentsWidget.vue'))
const LinkWidget = lazyWidget(() => import('@/architecture/presentation/widgets/LinkWidget.vue'))
const ListWidget = lazyWidget(() => import('@/architecture/presentation/widgets/ListWidget.vue'))
const FormWidget = lazyWidget(() => import('@/architecture/presentation/widgets/FormWidget.vue'))
const TableWidget = lazyWidget(() => import('@/architecture/presentation/widgets/TableWidget.vue'))

let widgetComponentsRegistered = false

function registerWidgetComponents(): void {
  if (widgetComponentsRegistered) {
    return
  }
  widgetComponentsRegistered = true

  widgetComponentFactory.registerRequestComponent(WidgetType.INPUT, InputWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.INPUT, InputWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.TEXT, InputWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.ID, InputWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.ID, InputWidget)

  widgetComponentFactory.registerRequestComponent(WidgetType.INTEGER, IntegerWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.INTEGER, IntegerWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.FLOAT, FloatWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.FLOAT, FloatWidget)

  widgetComponentFactory.registerRequestComponent(WidgetType.TEXT_AREA, TextAreaWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.TEXT_AREA, TextAreaWidget)

  widgetComponentFactory.registerRequestComponent(WidgetType.SWITCH, SwitchWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.SWITCH, SwitchWidget)

  widgetComponentFactory.registerRequestComponent(WidgetType.SELECT, SelectWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.SELECT, SelectWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.MULTI_SELECT, MultiSelectWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.MULTI_SELECT, MultiSelectWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.CHECKBOX, CheckboxWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.CHECKBOX, CheckboxWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.RADIO, RadioWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.RADIO, RadioWidget)

  widgetComponentFactory.registerRequestComponent(WidgetType.DATETIME, DateTimeWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.DATETIME, DateTimeWidget)

  widgetComponentFactory.registerRequestComponent(WidgetType.SLIDER, SliderWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.SLIDER, SliderWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.RATE, RateWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.RATE, RateWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.COLOR, ColorWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.COLOR, ColorWidget)

  widgetComponentFactory.registerRequestComponent(WidgetType.RICH_TEXT, RichTextWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.RICH_TEXT, RichTextResponseWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.FILES, FilesWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.FILES, FilesWidget)

  widgetComponentFactory.registerRequestComponent(WidgetType.USER, UserWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.USER, UserWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.USERS, UsersWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.USERS, UsersWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.DEPARTMENT, DepartmentWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.DEPARTMENT, DepartmentWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.DEPARTMENTS, DepartmentsWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.DEPARTMENTS, DepartmentsWidget)

  widgetComponentFactory.registerRequestComponent(WidgetType.LINK, LinkWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.LINK, LinkWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.PROGRESS, ProgressWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.PROGRESS, ProgressWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.LIST, ListWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.LIST, ListWidget)

  widgetComponentFactory.registerRequestComponent(WidgetType.FORM, FormWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.FORM, FormWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.TABLE, TableWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.TABLE, TableWidget)

  widgetComponentFactory.registerResponseComponent(WidgetType.TEXT, TextWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.TEXT, TextWidget)
}

export function initializeWidgetComponentFactory(): Promise<void> {
  registerWidgetComponents()
  return Promise.resolve()
}

export function ensureInitialized(): Promise<void> {
  return Promise.resolve()
}

registerWidgetComponents()

export { widgetComponentFactory } from './factory'
