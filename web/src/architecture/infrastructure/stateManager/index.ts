/**
 * StateManager 导出
 */

import { StateManagerImpl } from './StateManagerImpl'
import { FormStateManager } from './FormStateManager'
import type { IStateManager } from '../../domain/interfaces/IStateManager'

// 导出接口
export type { IStateManager } from '../../domain/interfaces/IStateManager'

// 导出实现
export { StateManagerImpl, FormStateManager }

