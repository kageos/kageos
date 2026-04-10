import type { IWidgetInitializer, WidgetInitContext } from '@/architecture/presentation/widgets/interfaces/IWidgetInitializer'
import type { FieldValue } from '@/architecture/domain/types'
import { hydrateTableField } from './nestedFieldHydrator'

export class TableWidgetInitializer implements IWidgetInitializer {
  async initialize(context: WidgetInitContext): Promise<FieldValue | null> {
    return hydrateTableField(context)
  }
}
