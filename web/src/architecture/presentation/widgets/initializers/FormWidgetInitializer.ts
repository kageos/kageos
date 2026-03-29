import type { IWidgetInitializer, WidgetInitContext } from '@/architecture/presentation/widgets/interfaces/IWidgetInitializer'
import type { FieldValue } from '@/architecture/domain/types'
import { FieldValueMeta } from '@/core/constants/field'
import { hydrateFormField } from './nestedFieldHydrator'

export class FormWidgetInitializer implements IWidgetInitializer {
  async initialize(context: WidgetInitContext): Promise<FieldValue | null> {
    const { field, currentValue } = context
    
    console.log(`🔍 [FormWidgetInitializer] 开始初始化字段 ${field.code}`, {
      currentValue: {
        raw: currentValue.raw,
        display: currentValue.display,
        fromURL: !!(currentValue.meta && currentValue.meta[FieldValueMeta.FROM_URL])
      },
      hasChildren: !!(field.children && field.children.length > 0),
      childrenCount: field.children?.length || 0,
      initSource: context.initSource
    })

    return hydrateFormField(context)
  }
}
