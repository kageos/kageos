import type { InjectionKey } from 'vue'

export interface PrdPreviewContext {
  interactive: boolean
}

export const prdPreviewContextKey: InjectionKey<PrdPreviewContext> = Symbol('PrdPreviewContext')
