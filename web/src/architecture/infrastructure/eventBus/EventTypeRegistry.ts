/**
 * 事件类型注册表
 * 
 * 🔥 统一管理事件类型，支持事件类型的元数据和验证
 * 
 * 职责：
 * - 注册事件类型及其元数据
 * - 提供事件类型查询和验证
 * - 支持事件类型的分类和描述
 */

import { WorkspaceEvent, FormEvent, TableEvent, RouteEvent } from '../../domain/interfaces/IEventBus'
import { Logger } from '@/core/utils/logger'

/**
 * 事件类型元数据
 */
export interface EventTypeMetadata {
  /** 事件名称 */
  name: string
  
  /** 事件描述 */
  description?: string
  
  /** 事件分类（如 'workspace', 'form', 'table', 'route'） */
  category?: string
  
  /** 事件版本（用于兼容性管理） */
  version?: string
  
  /** 事件负载类型（TypeScript 类型提示） */
  payloadType?: string
}

/**
 * 事件类型注册表
 */
export class EventTypeRegistry {
  private eventTypes: Map<string, EventTypeMetadata> = new Map()
  
  constructor() {
    // 注册默认事件类型
    this.registerDefaultEvents()
  }
  
  /**
   * 注册默认事件类型
   */
  private registerDefaultEvents(): void {
    // Workspace 事件
    this.register(WorkspaceEvent.nodeClicked, {
      name: '节点点击',
      description: '工作空间节点被点击',
      category: 'workspace',
      payloadType: 'NodeClickPayload'
    })
    
    this.register(WorkspaceEvent.appSwitched, {
      name: '应用切换',
      description: '工作空间应用切换',
      category: 'workspace',
      payloadType: 'AppSwitchPayload'
    })
    
    this.register(WorkspaceEvent.serviceTreeLoaded, {
      name: '服务树加载完成',
      description: '服务树加载完成',
      category: 'workspace',
      payloadType: 'ServiceTreeLoadedPayload'
    })
    
    this.register(WorkspaceEvent.functionLoaded, {
      name: '函数加载完成',
      description: '函数详情加载完成',
      category: 'workspace',
      payloadType: 'FunctionLoadedPayload'
    })
    
    // Form 事件
    this.register(FormEvent.initialized, {
      name: '表单初始化完成',
      description: '表单初始化完成',
      category: 'form',
      payloadType: 'FormInitializedPayload'
    })
    
    this.register(FormEvent.fieldValueUpdated, {
      name: '字段值更新',
      description: '表单字段值更新',
      category: 'form',
      payloadType: 'FieldValueUpdatedPayload'
    })
    
    this.register(FormEvent.validated, {
      name: '表单验证完成',
      description: '表单验证完成',
      category: 'form',
      payloadType: 'FormValidatedPayload'
    })
    
    this.register(FormEvent.submitted, {
      name: '表单提交',
      description: '表单提交',
      category: 'form',
      payloadType: 'FormSubmittedPayload'
    })
    
    this.register(FormEvent.responseReceived, {
      name: '响应数据接收',
      description: '表单响应数据接收',
      category: 'form',
      payloadType: 'FormResponseReceivedPayload'
    })
    
    // Table 事件
    this.register(TableEvent.dataLoaded, {
      name: '表格数据加载完成',
      description: '表格数据加载完成',
      category: 'table',
      payloadType: 'TableDataLoadedPayload'
    })
    
    this.register(TableEvent.searchChanged, {
      name: '搜索条件变化',
      description: '表格搜索条件变化',
      category: 'table',
      payloadType: 'TableSearchChangedPayload'
    })
    
    this.register(TableEvent.sortChanged, {
      name: '排序变化',
      description: '表格排序变化',
      category: 'table',
      payloadType: 'TableSortChangedPayload'
    })
    
    this.register(TableEvent.pageChanged, {
      name: '分页变化',
      description: '表格分页变化',
      category: 'table',
      payloadType: 'TablePageChangedPayload'
    })
    
    this.register(TableEvent.rowAdded, {
      name: '行新增',
      description: '表格行新增',
      category: 'table',
      payloadType: 'TableRowAddedPayload'
    })
    
    this.register(TableEvent.rowUpdated, {
      name: '行更新',
      description: '表格行更新',
      category: 'table',
      payloadType: 'TableRowUpdatedPayload'
    })
    
    this.register(TableEvent.rowDeleted, {
      name: '行删除',
      description: '表格行删除',
      category: 'table',
      payloadType: 'TableRowDeletedPayload'
    })
    
    // Route 事件
    this.register(RouteEvent.updateRequested, {
      name: '路由更新请求',
      description: '请求更新路由',
      category: 'route',
      payloadType: 'RouteUpdateRequestedPayload'
    })
    
    this.register(RouteEvent.updateCompleted, {
      name: '路由更新完成',
      description: '路由更新完成',
      category: 'route',
      payloadType: 'RouteUpdateCompletedPayload'
    })
    
    this.register(RouteEvent.pathChanged, {
      name: '路径变化',
      description: '路由路径变化',
      category: 'route',
      payloadType: 'RoutePathChangedPayload'
    })
    
    this.register(RouteEvent.queryChanged, {
      name: '查询参数变化',
      description: '路由查询参数变化',
      category: 'route',
      payloadType: 'RouteQueryChangedPayload'
    })
    
    this.register(RouteEvent.routeChanged, {
      name: '路由变化',
      description: '路由变化（path + query）',
      category: 'route',
      payloadType: 'RouteChangedPayload'
    })
  }
  
  /**
   * 注册事件类型
   * 
   * @param eventType 事件类型（字符串）
   * @param metadata 事件元数据
   */
  register(eventType: string, metadata: EventTypeMetadata): void {
    if (this.eventTypes.has(eventType)) {
      Logger.warn('[EventTypeRegistry]', `事件类型 "${eventType}" 已存在，将被覆盖`, {
        oldMetadata: this.eventTypes.get(eventType),
        newMetadata: metadata
      })
    }
    
    this.eventTypes.set(eventType, metadata)
  }
  
  /**
   * 获取事件类型元数据
   * 
   * @param eventType 事件类型
   * @returns 事件元数据，如果不存在返回 undefined
   */
  getMetadata(eventType: string): EventTypeMetadata | undefined {
    return this.eventTypes.get(eventType)
  }
  
  /**
   * 检查事件类型是否已注册
   * 
   * @param eventType 事件类型
   * @returns 是否已注册
   */
  hasEventType(eventType: string): boolean {
    return this.eventTypes.has(eventType)
  }
  
  /**
   * 根据分类获取事件类型
   * 
   * @param category 事件分类
   * @returns 事件类型列表
   */
  getEventsByCategory(category: string): string[] {
    return Array.from(this.eventTypes.entries())
      .filter(([_, metadata]) => metadata.category === category)
      .map(([eventType]) => eventType)
  }
  
  /**
   * 获取所有已注册的事件类型
   * 
   * @returns 事件类型列表
   */
  getAllEventTypes(): string[] {
    return Array.from(this.eventTypes.keys())
  }
  
  /**
   * 获取所有事件类型的元数据
   * 
   * @returns 事件类型和元数据的映射
   */
  getAllMetadata(): Map<string, EventTypeMetadata> {
    return new Map(this.eventTypes)
  }
  
  /**
   * 验证事件类型
   * 
   * @param eventType 事件类型
   * @returns 是否有效
   */
  validateEventType(eventType: string): boolean {
    return this.hasEventType(eventType)
  }
}

// 导出全局单例
export const eventTypeRegistry = new EventTypeRegistry()

