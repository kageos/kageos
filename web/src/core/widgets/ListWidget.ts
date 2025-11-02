/**
 * ListWidget - 列表组件（表格+表单混合模式 + 事件驱动）
 * 
 * 设计理念：
 * - 已填数据用表格展示（紧凑、清晰）
 * - 新增/编辑用表单展示（明确的编辑状态）
 * - 事件驱动：监听子组件事件，协调回调和聚合
 * - 符合传统 CRUD 的用户习惯
 */

import { h, ref, computed, markRaw } from 'vue'
import { ElButton, ElTable, ElTableColumn, ElForm, ElFormItem, ElIcon, ElMessage } from 'element-plus'
import { Plus, Delete, Edit, Check, Close } from '@element-plus/icons-vue'
import { BaseWidget } from './BaseWidget'
import { widgetFactory } from '../factories/WidgetFactory'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps } from '../types/widget'
import { selectFuzzy } from '@/api/function'  // 🔥 导入回调 API
import { ExpressionParser } from '../utils/ExpressionParser'  // 🔥 导入表达式解析器

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

/**
 * 已保存的行数据
 * 🔥 直接使用系统标准的 FieldValue，保持架构一致性
 */
interface SavedRowData {
  [field_code: string]: FieldValue
}

export class ListWidget extends BaseWidget {
  // List 配置
  private listConfig: ListConfig
  
  // 子字段配置（List 的元素类型）
  private itemFields: FieldConfig[]
  
  // 🔥 已保存的数据（用于表格展示）
  private savedData: any
  
  // 🔥 表单的 Widget 实例（用于新增/编辑）
  private formWidgets: any
  
  // 🔥 编辑状态
  private editingIndex: any  // null 表示不在编辑，数字表示编辑第几行
  private isAdding: any      // 是否正在新增
  
  // 🔥 聚合统计配置（从回调获取）
  private statisticsConfig: any
  
  // 🔥 聚合统计结果（计算后的值）
  private statisticsResult: any

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
    
    // 🔥 初始化状态
    this.savedData = ref<SavedRowData[]>([])
    this.formWidgets = ref<ListItemWidgets>({})
    this.editingIndex = ref<number | null>(null)
    this.isAdding = ref(false)
    this.statisticsConfig = ref<Record<string, string>>({})
    this.statisticsResult = ref<Record<string, any>>({})
    
    // 解析 List 配置
    this.listConfig = (this.field.widget?.config as ListConfig) || {}
    
    // 解析子字段（List 的元素类型）
    this.itemFields = this.parseItemFields()
    
    // 🔥 初始化默认行（如果配置了 default_items）
    const defaultItems = this.listConfig.default_items || 0
    if (defaultItems > 0) {
      // 创建空行数据
      for (let i = 0; i < defaultItems; i++) {
        this.savedData.value.push({})
      }
    }
    
    // 🔥 订阅子组件事件
    this.subscribeChildEvents()
  }

  /**
   * 解析子字段配置
   */
  private parseItemFields(): FieldConfig[] {
    if (this.field.children && Array.isArray(this.field.children)) {
      return this.field.children
    }
    return []
  }

  /**
   * 🔥 订阅子组件事件（核心方法）
   */
  private subscribeChildEvents(): void {
    console.log(`[ListWidget] ${this.fieldPath} 开始订阅子组件事件`)
    
    // 找出所有 select/multiselect 字段
    const selectFields = this.itemFields.filter(field => 
      field.widget?.type === 'select' || field.widget?.type === 'multiselect'
    )
    
    console.log(`[ListWidget] ${this.fieldPath} 发现 ${selectFields.length} 个 select/multiselect 字段`)
    
    selectFields.forEach(field => {
      // 订阅搜索事件（如果配置了回调）
      if (field.callbacks?.includes('OnSelectFuzzy')) {
        this.subscribeSearchEvent(field)
      }
      
      // 订阅变化事件（用于聚合统计）
      this.subscribeChangeEvent(field)
    })
  }

  /**
   * 🔥 订阅搜索事件（核心：调用后端回调）
   */
  private subscribeSearchEvent(field: FieldConfig): void {
    // 监听所有行的该字段的搜索事件
    // 例如：field:search:products[].product_id
    const eventPattern = `field:search:${this.fieldPath}[].${field.code}`
    
    console.log(`[ListWidget] 订阅搜索事件: ${eventPattern}`)
    
    this.formManager.on(eventPattern, async (event: any) => {
      console.log(`[ListWidget] 收到子组件搜索事件:`, event)
      console.log(`[ListWidget]   触发字段: ${event.fieldPath}`)
      console.log(`[ListWidget]   查询关键词: "${event.query}"`)
      
      try {
        // 1. 获取函数的 method 和 router
        if (!this.formRenderer?.getFunctionMethod || !this.formRenderer?.getFunctionRouter) {
          console.error(`[ListWidget] formRenderer 不完整，无法调用回调`)
          if (event.callback) event.callback([])
          return
        }
        
        const method = this.formRenderer.getFunctionMethod()
        const router = this.formRenderer.getFunctionRouter()
        
        if (!router) {
          console.error(`[ListWidget] 无法获取函数路由`)
          if (event.callback) event.callback([])
          return
        }
        
        // 2. 构建回调请求体
        const queryType: 'by_keyword' | 'by_value' = event.isByValue ? 'by_value' : 'by_keyword'
        const requestBody = {
          code: field.code,
          type: queryType,
          value: event.query,
          request: this.formRenderer.getSubmitData?.() || {},  // 🔥 获取完整表单数据
          value_type: field.data?.type || 'string'
        }
        
        console.log(`[ListWidget] 调用后端回调:`, {
          method,
          router,
          field: field.code,
          query: event.query
        })
        
        // 3. 调用回调 API
        const response = await selectFuzzy(method, router, requestBody)
        
        console.log(`[ListWidget] 回调响应:`, response)
        
        // 4. 解析响应
        const { items, error_msg, statistics } = response || {}
        
        if (error_msg) {
          ElMessage.error(error_msg)
          if (event.callback) event.callback([])
          return
        }
        
        // 5. 保存聚合配置
        if (statistics && typeof statistics === 'object') {
          this.statisticsConfig.value = statistics
          console.log(`[ListWidget] 保存聚合配置:`, statistics)
          
          // 🔥 立即触发一次计算（如果已有数据）
          if (this.savedData.value.length > 0) {
            this.recalculateStatistics()
          }
        }
        
        // 6. 转换选项格式并返回给 SelectWidget
        const options = (items || []).map((item: any) => ({
          label: item.label || String(item.value),
          value: item.value,
          disabled: false,
          displayInfo: item.display_info,
          icon: item.icon
        }))
        
        console.log(`[ListWidget] 返回 ${options.length} 个选项给子组件`)
        
        // 7. 通过回调函数返回选项
        if (event.callback) {
          event.callback(options)
        }
        
      } catch (error: any) {
        console.error(`[ListWidget] 回调失败:`, error)
        ElMessage.error(error?.message || '查询失败')
        if (event.callback) event.callback([])
      }
    })
  }

  /**
   * 🔥 订阅变化事件（触发聚合计算）
   */
  private subscribeChangeEvent(field: FieldConfig): void {
    // 监听所有行的该字段的变化事件
    // 例如：field:change:products[].product_id
    const eventPattern = `field:change:${this.fieldPath}[].${field.code}`
    
    console.log(`[ListWidget] 订阅变化事件: ${eventPattern}`)
    
    this.formManager.on(eventPattern, (event: any) => {
      console.log(`[ListWidget] 收到子组件变化事件:`, event)
      console.log(`[ListWidget]   触发字段: ${event.fieldPath}`)
      
      // 🔥 重新计算聚合统计
      this.recalculateStatistics()
    })
  }

  /**
   * 🔥 获取所有行的数据（用于聚合计算）
   * 包含：raw 值 + displayInfo
   */
  private getAllRowsData(): any[] {
    return this.savedData.value.map(row => {
      const merged: Record<string, any> = {}
      
      for (const [fieldCode, fieldValue] of Object.entries(row)) {
        // 保存 raw 值
        merged[fieldCode] = fieldValue.raw
        
        // 🔥 合并 displayInfo（来自 Select 回调）
        if (fieldValue.meta?.displayInfo) {
          Object.assign(merged, fieldValue.meta.displayInfo)
        }
        
        // 🔥 合并行内聚合统计（来自 MultiSelect，场景 4 二层聚合）
        if (fieldValue.meta?.rowStatistics) {
          Object.assign(merged, fieldValue.meta.rowStatistics)
        }
      }
      
      return merged
    })
  }

  /**
   * 🔥 重新计算聚合统计（核心方法）
   */
  private recalculateStatistics(): void {
    // 检查是否有聚合配置
    if (!this.statisticsConfig.value || Object.keys(this.statisticsConfig.value).length === 0) {
      console.log(`[ListWidget] ${this.fieldPath} 无聚合配置，跳过计算`)
      return
    }
    
    console.log(`[ListWidget] ${this.fieldPath} 开始计算聚合统计`)
    
    // 1. 获取所有行的数据
    const allRows = this.getAllRowsData()
    
    console.log(`[ListWidget] 数据行数: ${allRows.length}`)
    console.log(`[ListWidget] 聚合配置:`, this.statisticsConfig.value)
    console.log(`[ListWidget] 第一行数据示例:`, allRows[0])
    
    // 2. 遍历聚合配置，计算每个统计项
    const result: Record<string, any> = {}
    
    for (const [label, expression] of Object.entries(this.statisticsConfig.value)) {
      try {
        console.log(`\n[ListWidget] 计算: ${label} = ${expression}`)
        // 使用表达式解析器计算
        const value = ExpressionParser.evaluate(expression, allRows)
        result[label] = value
        
        console.log(`[ListWidget]   ✓ 结果: ${value}`)
      } catch (error) {
        console.error(`[ListWidget] ✗ 计算失败: ${label} = ${expression}`, error)
        result[label] = 0
      }
    }
    
    // 3. 更新统计结果
    this.statisticsResult.value = result
    
    console.log(`\n[ListWidget] 聚合统计完成:`, result)
    
    // 4. 发出 List 聚合完成事件（如果父组件需要）
    this.emit('list:statistics:updated', {
      statistics: result
    })
  }

  /**
   * 🔥 创建表单的 Widget 实例
   */
  private createFormWidgets(initialData?: SavedRowData): void {
    const widgets: ListItemWidgets = {}
    
    for (const itemField of this.itemFields) {
      // 🔥 表单的 fieldPath 使用临时路径（不加索引）
      const tempFieldPath = `${this.fieldPath}._form_.${itemField.code}`
      
      // 获取初始值（编辑时使用已有值，新增时使用默认值）
      const WidgetClass = widgetFactory.getWidgetClass(itemField.widget.type)
      const defaultValue = WidgetClass.getDefaultValue(itemField)
      
      // 🔥 直接使用 FieldValue，无需转换（已经是标准格式）
      const fieldValue = initialData?.[itemField.code] || defaultValue
      
      // 初始化到 FormDataManager
      this.formManager.setValue(tempFieldPath, fieldValue)
      
      // 创建子 Widget
      const childProps: WidgetRenderProps = {
        field: itemField,
        currentFieldPath: tempFieldPath,
        value: this.formManager.getValue(tempFieldPath),
        onChange: (newValue: FieldValue) => {
          this.formManager.setValue(tempFieldPath, newValue)
        },
        formManager: this.formManager,
        formRenderer: this.formRenderer,
        depth: this.depth + 1
      }
      
      const widget = new WidgetClass(childProps)
      
      if (widget) {
        widgets[itemField.code] = markRaw(widget)
      }
    }
    
    this.formWidgets.value = widgets
  }

  /**
   * 🔥 清空表单的 Widget 实例
   */
  private clearFormWidgets(): void {
    // 清空 FormDataManager 中的数据
    for (const itemField of this.itemFields) {
      const tempFieldPath = `${this.fieldPath}._form_.${itemField.code}`
      this.formManager.setValue(tempFieldPath, {
        raw: null,
        display: '',
        meta: {}
      })
    }
    
    this.formWidgets.value = {}
  }

  /**
   * 🔥 开始新增
   */
  private startAdding(): void {
    const maxItems = this.listConfig.max_items
    if (maxItems && this.savedData.value.length >= maxItems) {
      ElMessage.warning(`已达到最大行数 ${maxItems}`)
      return
    }
    
    this.isAdding.value = true
    this.editingIndex.value = null
    this.createFormWidgets()
  }

  /**
   * 🔥 开始编辑
   */
  private startEditing(index: number): void {
    this.editingIndex.value = index
    this.isAdding.value = false
    const rowData = this.savedData.value[index]
    this.createFormWidgets(rowData)
  }

  /**
   * 🔥 保存（新增或编辑）
   */
  private handleSave(): void {
    // 🔥 直接使用 Widget 的 FieldValue，无需重构数据
    const rowData: SavedRowData = {}
    
    for (const [fieldCode, widget] of Object.entries(this.formWidgets.value)) {
      const rawWidget = widget as any
      // 直接获取完整的 FieldValue（包含 raw、display、meta）
      rowData[fieldCode] = rawWidget.getValue()
    }
    
    if (this.isAdding.value) {
      // 新增
      this.savedData.value.push(rowData)
      console.log(`[ListWidget] 新增行:`, rowData)
    } else if (this.editingIndex.value !== null) {
      // 编辑
      this.savedData.value[this.editingIndex.value] = rowData
      console.log(`[ListWidget] 编辑行 ${this.editingIndex.value}:`, rowData)
    }
    
    // 清空状态
    this.handleCancel()
    
    // 触发外部的 onChange（通知父组件数据已变化）
    this.updateParentValue()
    
    // 🔥 重新计算聚合统计（数据已变化）
    this.recalculateStatistics()
  }

  /**
   * 🔥 取消（新增或编辑）
   */
  private handleCancel(): void {
    this.isAdding.value = false
    this.editingIndex.value = null
    this.clearFormWidgets()
  }

  /**
   * 🔥 删除一行
   */
  private handleDelete(index: number): void {
    const minItems = this.listConfig.min_items || 0
    if (this.savedData.value.length <= minItems) {
      ElMessage.warning(`已达到最小行数 ${minItems}`)
      return
    }
    
    this.savedData.value.splice(index, 1)
    console.log(`[ListWidget] 删除行 ${index}`)
    
    // 触发外部的 onChange
    this.updateParentValue()
    
    // 🔥 重新计算聚合统计（数据已变化）
    this.recalculateStatistics()
  }

  /**
   * 🔥 更新父组件的值
   */
  private updateParentValue(): void {
    const newValue: FieldValue = {
      raw: this.savedData.value,
      display: `共 ${this.savedData.value.length} 条`,
      meta: {}
    }
    
    // 调用 onChange 通知父组件
    if (this.onChange) {
      this.onChange(newValue)
    }
  }

  /**
   * 🔥 渲染表格（展示已有数据）
   */
  private renderTable(): any {
    if (this.savedData.value.length === 0) {
      return h('div', {
        style: {
          padding: '20px',
          textAlign: 'center',
          color: 'var(--el-text-color-secondary)',
          backgroundColor: 'var(--el-fill-color-lighter)',
          borderRadius: '4px',
          marginBottom: '12px'
        }
      }, '暂无数据，点击下方"添加"按钮开始')
    }
    
    // 渲染表格
    return h(ElTable, {
      data: this.savedData.value,
      border: true,
      stripe: true,
      style: { width: '100%', marginBottom: '12px' }
    }, {
      default: () => [
        // 序号列
        h(ElTableColumn, {
          type: 'index',
          label: '#',
          width: 60,
          align: 'center'
        }),
        
        // 数据列
        ...this.itemFields.map(field => 
          h(ElTableColumn, {
            key: field.code,
            prop: field.code,
            label: field.name,
            minWidth: this.getColumnWidth(field)
          }, {
            default: ({ row }: { row: SavedRowData }) => {
              const value = row[field.code]
              return this.formatCellValue(value, field)
            }
          })
        ),
        
        // 操作列
        h(ElTableColumn, {
          label: '操作',
          width: 150,
          align: 'center',
          fixed: 'right'
        }, {
          default: ({ $index }: { $index: number }) => {
            return h('div', { style: { display: 'flex', gap: '8px', justifyContent: 'center' } }, [
              h(ElButton, {
                link: true,
                type: 'primary',
                icon: Edit,
                onClick: () => this.startEditing($index)
              }, { default: () => '编辑' }),
              
              h(ElButton, {
                link: true,
                type: 'danger',
                icon: Delete,
                onClick: () => this.handleDelete($index)
              }, { default: () => '删除' })
            ])
          }
        })
      ]
    })
  }

  /**
   * 🔥 获取列宽
   */
  private getColumnWidth(field: FieldConfig): number {
    if (field.widget?.type === 'timestamp') return 180
    if (field.widget?.type === 'textarea' || field.widget?.type === 'text_area') return 200
    return 120
  }

  /**
   * 🔥 格式化单元格值
   */
  private formatCellValue(fieldValue: FieldValue, field: FieldConfig): string {
    if (!fieldValue) return '-'
    
    // 🔥 直接使用 FieldValue 的 display 属性
    if (fieldValue.display) {
      return fieldValue.display
    }
    
    // 降级：如果 display 为空，尝试格式化 raw 值
    const raw = fieldValue.raw
    if (raw === null || raw === undefined) return '-'
    
    // 根据字段类型格式化 raw 值
    if (field.widget?.type === 'timestamp') {
      return this.formatTimestamp(raw)
    }
    
    if (Array.isArray(raw)) {
      return raw.join(', ')
    }
    
    return String(raw)
  }

  /**
   * 格式化时间戳
   */
  private formatTimestamp(timestamp: number | string): string {
    if (!timestamp) return '-'
    const date = new Date(timestamp)
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hours = String(date.getHours()).padStart(2, '0')
    const minutes = String(date.getMinutes()).padStart(2, '0')
    const seconds = String(date.getSeconds()).padStart(2, '0')
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
  }

  /**
   * 🔥 渲染表单（新增/编辑）
   */
  private renderForm(): any {
    if (!this.isAdding.value && this.editingIndex.value === null) {
      return null
    }
    
    const title = this.isAdding.value ? '新增' : `编辑第 ${this.editingIndex.value! + 1} 行`
    
    return h('div', {
      style: {
        padding: '20px',
        backgroundColor: 'var(--el-fill-color-lighter)',
        borderRadius: '8px',
        marginBottom: '12px',
        border: '2px solid var(--el-color-primary)'
      }
    }, [
      // 表单标题
      h('div', {
        style: {
          fontSize: '14px',
          fontWeight: 'bold',
          color: 'var(--el-color-primary)',
          marginBottom: '16px'
        }
      }, title),
      
      // 表单字段
      h(ElForm, {
        labelWidth: '100px',
        labelPosition: 'right'
      }, {
        default: () => this.itemFields.map(field => {
          const widget = this.formWidgets.value[field.code]
          if (!widget) return null
          
          return h(ElFormItem, {
            key: field.code,
            label: field.name,
            style: { marginBottom: '18px' }
          }, {
            default: () => h('div', {
              style: { width: '100%' }
            }, [(widget as any).render()])
          })
        })
      }),
      
      // 操作按钮（保存在左，取消在右，占满宽度）
      h('div', {
        style: {
          display: 'flex',
          gap: '12px',
          marginTop: '16px',
          width: '100%'
        }
      }, [
        h(ElButton, {
          type: 'primary',
          icon: Check,
          onClick: () => this.handleSave(),
          style: { flex: 1 }
        }, { default: () => '保存' }),
        
        h(ElButton, {
          onClick: () => this.handleCancel(),
          style: { flex: 1 }
        }, { default: () => '取消' })
      ])
    ])
  }

  /**
   * 🔥 渲染聚合统计结果
   */
  private renderStatistics() {
    // 如果没有统计结果，不渲染
    if (!this.statisticsResult.value || Object.keys(this.statisticsResult.value).length === 0) {
      console.log(`[ListWidget] renderStatistics: 无统计结果`)
      return null
    }
    
    console.log(`[ListWidget] renderStatistics: 渲染统计结果`, this.statisticsResult.value)
    
    return h('div', {
      class: 'list-statistics',
      style: {
        width: '100%',
        marginTop: '12px',
        padding: '12px 16px',
        backgroundColor: 'var(--el-fill-color-light)',
        borderRadius: '4px',
        border: '1px solid var(--el-border-color-lighter)',
        display: 'flex',
        flexWrap: 'wrap',
        gap: '16px'
      }
    }, 
      // 渲染每个统计项
      Object.entries(this.statisticsResult.value).map(([label, value]) => {
        // 🔥 判断是数值还是文本
        const isNumeric = typeof value === 'number'
        const displayValue = isNumeric ? ExpressionParser.formatNumber(value) : String(value)
        
        return h('div', {
          key: label,
          class: 'statistics-item',
          style: {
            display: 'flex',
            alignItems: 'center',
            gap: '8px'
          }
        }, [
          // 标签
          h('span', {
            style: {
              fontSize: '13px',
              color: 'var(--el-text-color-secondary)',
              fontWeight: '500'
            }
          }, `${label}:`),
          
          // 数值或文本
          h('span', {
            style: {
              fontSize: '14px',
              color: isNumeric ? 'var(--el-color-primary)' : 'var(--el-text-color-primary)',
              fontWeight: isNumeric ? 'bold' : 'normal'
            }
          }, displayValue)
        ])
      })
    )
  }

  /**
   * 🔥 渲染组件（主入口）
   */
  render() {
    return h('div', { class: 'list-widget', style: { width: '100%' } }, [
      // 列表标题
      h('div', {
        style: {
          marginBottom: '12px',
          fontSize: '14px',
          fontWeight: 'bold',
          color: 'var(--el-text-color-primary)',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }
      }, [
        h('span', this.field.name),
        h('span', {
          style: {
            fontSize: '12px',
            color: 'var(--el-text-color-secondary)',
            fontWeight: 'normal'
          }
        }, `共 ${this.savedData.value.length} 条`)
      ]),
      
      // 表格展示
      this.renderTable(),
      
      // 🔥 聚合统计结果
      this.renderStatistics(),
      
      // 新增/编辑表单
      this.renderForm(),
      
      // 添加按钮（仅在不处于编辑状态时显示，宽度与表格一致）
      (!this.isAdding.value && this.editingIndex.value === null) ? h('div', {
        style: { width: '100%' }
      }, [
        h(ElButton, {
          type: 'primary',
          icon: Plus,
          onClick: () => this.startAdding(),
          style: { width: '100%' }
        }, { default: () => '添加' })
      ]) : null
    ])
  }

  /**
   * 捕获组件数据（用于快照）
   */
  protected captureComponentData(): ListComponentData {
    return {
      item_count: this.savedData.value.length
    }
  }

  /**
   * 恢复组件数据（从快照）
   */
  protected restoreComponentData(data: ListComponentData): void {
    console.log(`[ListWidget] 恢复组件数据:`, data)
  }

  /**
   * 🔥 获取提交时的原始值
   * 从 FieldValue 中提取 raw 值（后端不需要 display 和 meta）
   */
  getRawValueForSubmit(): any[] {
    const result = this.savedData.value.map(row => {
      const rowRaw: Record<string, any> = {}
      
      for (const [fieldCode, fieldValue] of Object.entries(row)) {
        // 🔥 提取 FieldValue 的 raw 属性
        rowRaw[fieldCode] = fieldValue.raw
      }
      
      return rowRaw
    })
    
    console.log(`[ListWidget] ${this.fieldPath} 提交数据:`, result)
    return result
  }
}
