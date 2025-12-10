/**
 * EventBus 导出
 */

import { EventBusImpl } from './EventBusImpl'
import type { IEventBus } from '../../domain/interfaces/IEventBus'

// 导出接口
export type { IEventBus } from '../../domain/interfaces/IEventBus'
export { WorkspaceEvent, FormEvent, TableEvent, RouteEvent } from '../../domain/interfaces/IEventBus'

// 导出实现
export { EventBusImpl }

// 导出单例实例（可选，也可以在使用时创建新实例）
export const eventBus: IEventBus = new EventBusImpl()

