/**
 * StateManager 导出
 */

import { StateManagerImpl } from './StateManagerImpl'
import { FormStateManager } from './FormStateManager'
import { WorkspaceStateManager } from './WorkspaceStateManager'
import { TableStateManager } from './TableStateManager'
import type { IStateManager } from '../../domain/interfaces/IStateManager'

// 导出接口
export type { IStateManager } from '../../domain/interfaces/IStateManager'

// 导出实现
export { StateManagerImpl, FormStateManager, WorkspaceStateManager, TableStateManager }

