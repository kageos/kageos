/**
 * ListWidget - 列表组件
 * 支持添加/删除行、递归渲染子组件、聚合统计
 */

import { h, ref, computed, markRaw } from 'vue'
import { ElButton, ElCard, ElIcon } from 'element-plus'
import { Plus, Delete } from '@element-plus/icons-vue'
import { BaseWidget } from './BaseWidget'
import { widgetFactory } from '../factories/WidgetFactory'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps } from '../types/widget'

/**
 * List 配置
 */
export interface ListConfig {
  min_items?: number
  max_items?: number
  default_items?: number
  [key: string]: any
}

/**
 * List 子元素的 Widget 实例
 */
interface ListItemWidgets {
  [field_code: string]: BaseWidget
}

/**
 * List 组件数据（用于快照）
 */
interface ListComponentData {
  item_count: number
}

export class ListWidget extends BaseWidget {
  // List 配置
  private listConfig: ListConfig
  
  // 子字段配置（List 的元素类型）
  private itemFields: FieldConfig[]
  
  // 每一行的 Widget 实例 [行索引 -> { field_code -> Widget }]
  private itemWidgets: any
  
  // 当前行数
  private itemCount: any

  /**
   * ListWidget 的默认值是空数组
   */
  static getDefaultValue(field: FieldConfig): FieldValue {
    return {
      raw: [],
      display: '[]',
      meta: {}
    }
  }

  constructor(props: WidgetRenderProps) {
    super(props)
    
    // 🔥 在构造函数中初始化 ref（避免类属性初始化问题）
    this.itemWidgets = ref<Map<number, ListItemWidgets>>(new Map())
    this.itemCount = ref(0)
    
    // 解析 List 配置
    this.listConfig = (this.field.widget?.config as ListConfig) || {}
    
    // 解析子字段（List 的元素类型）
    this.itemFields = this.parseItemFields()
    
    // 初始化默认行数
    const defaultItems = this.listConfig.default_items || 1
    for (let i = 0; i < defaultItems; i++) {
      this.addItem()
    }
    
    // 🔥 订阅子组件的事件（Select/MultiSelect 的搜索事件）
    this.subscribeChildEvents()
  }

  /**
   * 解析子字段配置
   */
  private parseItemFields(): FieldConfig[] {
    // 检查是否是 object 类型（结构体）
    // 注意：后端返回的是 "children"，不是 "properties"
    if (this.field.children && Array.isArray(this.field.children)) {
      return this.field.children
    }
    
    // 如果是简单类型的列表（如 list<string>）
    // 这里先不处理，后续扩展
    return []
  }

  /**
   * 订阅子组件事件
   */
  private subscribeChildEvents(): void {
    // TODO: 事件系统尚未实现，暂时注释
    // 🔥 监听 field:search 事件（Select/MultiSelect 触发）
    // this.formManager.on('field:search', (eventData: any) => {
    //   console.log(`[ListWidget] 收到子组件搜索事件:`, eventData)
    //   
    //   // 检查是否是当前 List 的子组件
    //   if (eventData.field_path?.startsWith(this.fieldPath + '[')) {
    //     this.handleChildSearch(eventData)
    //   }
    // })
    // 
    // 🔥 监听 field:change 事件（用于触发聚合计算）
    // this.formManager.on('field:change', (eventData: any) => {
    //   // 检查是否是当前 List 的子组件
    //   if (eventData.field_path?.startsWith(this.fieldPath + '[')) {
    //     this.recalculateAggregation()
    //   }
    // })
  }

  /**
   * 处理子组件搜索（调用回调 API）
   */
  private async handleChildSearch(eventData: any): Promise<void> {
    console.log(`[ListWidget] 处理子组件搜索:`, eventData)
    
    // TODO: 调用实际的回调 API
    // 目前先模拟返回数据
    
    // 模拟异步请求
    setTimeout(() => {
      console.log(`[ListWidget] 搜索完成，更新子组件选项`)
      // 实际项目中，这里会通过事件将结果传递给子组件
    }, 300)
  }

  /**
   * 重新计算聚合
   */
  private recalculateAggregation(): void {
    // TODO: 实现聚合计算逻辑
    console.log(`[ListWidget] 重新计算聚合`)
    
    // 检查是否配置了聚合
    const statistics = this.field.widget?.statistics
    if (!statistics) {
      return
    }
    
    // 遍历所有行，收集数据，计算聚合
    // 例如：sum(price * quantity)
    // 实际实现需要 ExpressionParser
  }

  /**
   * 添加一行
   */
  private addItem(): void {
    const maxItems = this.listConfig.max_items
    if (maxItems && this.itemCount.value >= maxItems) {
      console.warn(`[ListWidget] 已达到最大行数 ${maxItems}`)
      return
    }
    
    const newIndex = this.itemCount.value
    this.itemCount.value++
    
    // 为新行创建 Widget 实例
    const rowWidgets: ListItemWidgets = {}
    
    for (const itemField of this.itemFields) {
      const itemFieldPath = `${this.fieldPath}[${newIndex}].${itemField.code}`
      
      // 创建子 Widget
      const childProps: WidgetRenderProps = {
        field: itemField,
        currentFieldPath: itemFieldPath,
        value: this.formManager.getValue(itemFieldPath),
        onChange: (newValue: FieldValue) => {
          this.formManager.setValue(itemFieldPath, newValue)
        },
        formManager: this.formManager,
        formRenderer: this.formRenderer,  // 🔥 传递完整的 formRenderer（包含 getFunctionMethod/Router）
        depth: this.depth + 1
      }
      
      // 🔥 Debug: 检查 formRenderer 是否完整
      if (!this.formRenderer?.getFunctionMethod || !this.formRenderer?.getFunctionRouter) {
        console.warn(`[ListWidget] ${itemFieldPath} formRenderer 不完整:`, {
          hasRegisterWidget: !!this.formRenderer?.registerWidget,
          hasGetFunctionMethod: !!this.formRenderer?.getFunctionMethod,
          hasGetFunctionRouter: !!this.formRenderer?.getFunctionRouter
        })
      }
      
      const WidgetClass = widgetFactory.getWidgetClass(itemField.widget.type)
      const widget = new WidgetClass(childProps)
      
      if (widget) {
        // 🔥 使用 markRaw 标记 widget 为非响应式，避免 Vue 破坏其内部的 ref
        rowWidgets[itemField.code] = markRaw(widget)
        
        // 🔥 注册到父级的 allWidgets（用于快照）
        if (this.formRenderer?.registerWidget) {
          this.formRenderer.registerWidget(itemFieldPath, widget)
        }
      }
    }
    
    this.itemWidgets.value.set(newIndex, rowWidgets)
    
    console.log(`[ListWidget] 添加行 ${newIndex}`, rowWidgets)
  }

  /**
   * 删除一行
   */
  private deleteItem(index: number): void {
    const minItems = this.listConfig.min_items || 0
    if (this.itemCount.value <= minItems) {
      console.warn(`[ListWidget] 已达到最小行数 ${minItems}`)
      return
    }
    
    // 移除 Widget 实例
    this.itemWidgets.value.delete(index)
    
    // 清空该行的数据
    for (const itemField of this.itemFields) {
      const itemFieldPath = `${this.fieldPath}[${index}].${itemField.code}`
      this.formManager.setValue(itemFieldPath, {
        raw: null,
        display: '',
        meta: {}
      })
      
      // 🔥 从父级的 allWidgets 移除
      if (this.formRenderer?.unregisterWidget) {
        this.formRenderer.unregisterWidget(itemFieldPath)
      }
    }
    
    // 重新计算聚合
    this.recalculateAggregation()
    
    console.log(`[ListWidget] 删除行 ${index}`)
  }

  /**
   * 渲染单行
   */
  private renderItem(index: number): any {
    const rowWidgets = this.itemWidgets.value.get(index)
    if (!rowWidgets) {
      return null
    }
    
    return h('div', { 
      class: 'list-item',
      style: {
        display: 'flex',
        alignItems: 'flex-start',
        gap: '12px',
        marginBottom: '12px',
        padding: '16px',
        border: '1px solid #e4e7ed',
        borderRadius: '4px',
        backgroundColor: '#fafafa'
      }
    }, [
      // 行号
      h('div', {
        style: {
          minWidth: '30px',
          lineHeight: '32px',
          color: '#909399',
          fontWeight: 'bold'
        }
      }, `${index + 1}.`),
      
      // 字段列表
      h('div', {
        style: {
          flex: 1,
          display: 'flex',
          gap: '12px',
          flexWrap: 'wrap'
        }
      }, this.itemFields.map(itemField => {
        const widget = rowWidgets[itemField.code]
        if (!widget) {
          return null
        }
        
        return h('div', {
          style: {
            flex: '1 1 200px',
            minWidth: '200px'
          }
        }, [
          h('label', {
            style: {
              display: 'block',
              marginBottom: '4px',
              fontSize: '12px',
              color: '#606266'
            }
          }, itemField.name),
          widget.render()
        ])
      })),
      
      // 删除按钮
      h(ElButton, {
        type: 'danger',
        text: true,
        icon: Delete,
        onClick: () => this.deleteItem(index),
        style: { marginTop: '24px' }
      }, { default: () => '删除' })
    ])
  }

  /**
   * 渲染组件
   */
  render() {
    const items: any[] = []
    
    // 渲染所有行
    for (let i = 0; i < this.itemCount.value; i++) {
      if (this.itemWidgets.value.has(i)) {
        items.push(this.renderItem(i))
      }
    }
    
    return h('div', { class: 'list-widget' }, [
      // 列表标题
      h('div', {
        style: {
          marginBottom: '12px',
          fontSize: '14px',
          fontWeight: 'bold',
          color: '#303133'
        }
      }, this.field.name),
      
      // 列表项
      ...items,
      
      // 添加按钮
      h('div', { style: { marginTop: '12px' } }, [
        h(ElButton, {
          type: 'primary',
          plain: true,
          icon: Plus,
          onClick: () => this.addItem()
        }, { default: () => '添加一行' })
      ])
    ])
  }

  /**
   * 捕获组件数据（用于快照）
   */
  protected captureComponentData(): ListComponentData {
    return {
      item_count: this.itemCount.value
    }
  }

  /**
   * 恢复组件数据（从快照）
   */
  protected restoreComponentData(data: ListComponentData): void {
    // TODO: 恢复列表行数和子组件
    console.log(`[ListWidget] 恢复组件数据:`, data)
  }

  /**
   * 🔥 重写：获取提交时的原始值（递归收集子组件的值）
   * 
   * ListWidget 不依赖自己的 raw 值，而是主动遍历子组件收集它们的值
   * 这是方案 4 的核心：容器组件负责收集子组件，递归处理嵌套结构
   */
  getRawValueForSubmit(): any[] {
    const result: any[] = []
    
    console.log(`[ListWidget] ${this.fieldPath} 开始收集子组件值，共 ${this.itemCount.value} 行`)
    
    // 遍历每一行
    this.itemWidgets.value.forEach((rowWidgets, index) => {
      const rowData: Record<string, any> = {}
      
      console.log(`[ListWidget] ${this.fieldPath}[${index}] 收集该行的字段`)
      
      // 遍历该行的每个字段
      Object.entries(rowWidgets).forEach(([fieldCode, widget]) => {
        // 🔥 递归调用：子组件可能是基础组件（直接返回值）或容器组件（继续递归）
        const rawWidget = widget as any  // markRaw 后需要转换
        rowData[fieldCode] = rawWidget.getRawValueForSubmit()
        
        console.log(`[ListWidget]   - ${fieldCode}:`, rowData[fieldCode])
      })
      
      result.push(rowData)
    })
    
    console.log(`[ListWidget] ${this.fieldPath} 收集完成:`, result)
    return result
  }
}

