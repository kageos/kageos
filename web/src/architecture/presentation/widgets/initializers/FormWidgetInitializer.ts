import type { IWidgetInitializer, WidgetInitContext } from '@/architecture/presentation/widgets/interfaces/IWidgetInitializer'
import type { FieldValue } from '@/architecture/domain/types'
import { hydrateFormField } from './nestedFieldHydrator'

export class FormWidgetInitializer implements IWidgetInitializer {
  async initialize(context: WidgetInitContext): Promise<FieldValue | null> {
    return hydrateFormField(context)
  }
}
