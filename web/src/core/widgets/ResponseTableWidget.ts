/**
 * ResponseTableWidget - 返回值表格组件
 * 用于渲染返回值中的 table/list 类型字段（只读展示）
 * 
 * 功能特性：
 * - 点击 ID 列查看详情
 * - 详情抽屉导航（上一个/下一个）
 * - 只读展示，无编辑功能
 */

import { h, ref, computed, toRaw } from 'vue'
import { ElTable, ElTableColumn, ElDescriptions, ElDescriptionsItem } from 'element-plus'
import { ArrowLeft, ArrowRight, Close } from '@element-plus/icons-vue'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig, FieldValue } from '../types/field'
import { WidgetBuilder } from '../factories/WidgetBuilder'
import { convertToFieldValue } from '../../utils/field'
import { WidgetType } from '../constants/widget'
import {
  createFormDrawerState,
  handleFormFieldClick,
  handleCloseFormDetail,
  renderFormFieldButton,
  renderFormDetailDrawer,
  createDrawerContentComputed,
  type FormDrawerState,
  type DrawerContentRenderer
} from './utils/TableFormDrawerHelper'
import { ResponseFormWidget } from './ResponseFormWidget'
import { Logger } from '../utils/logger'

export class ResponseTableWidget extends BaseWidget {
  // 🔥 详情抽屉状态（用于 ID 列点击）
  private showDetailDrawer = ref(false)
  private currentDetailRow = ref<any>(null)
  private currentDetailIndex = ref<number>(-1)
  private tableData = ref<any[]>([])
  
  // 🔥 Form 字段详情抽屉状态（使用工具类管理）
  private formDrawerState: FormDrawerState = createFormDrawerState()
  
  // 🔥 使用 computed 包装抽屉渲染，确保响应式更新（作为实例属性）
  private drawerContent = createDrawerContentComputed(
    this.formDrawerState,
    () => this.renderFormDetailDrawer(),
    'ResponseTableWidget'
  )
  /**
   * 🔥 判断是否是 ID 列
   */
  private isIdColumn(field: FieldConfig): boolean {
    const code = field.code.toLowerCase()
    return code === 'id' || code === 'ID' || code.endsWith('_id') || code.endsWith('Id')
  }

  /**
   * 🔥 处理 ID 列点击
   */
  private handleIdClick(row: any, index: number): void {
    this.currentDetailRow.value = row
    this.currentDetailIndex.value = index
    this.showDetailDrawer.value = true
  }

  /**
   * 🔥 处理导航（上一个/下一个）
   */
  private handleNavigate(direction: 'prev' | 'next'): void {
    const data = this.tableData.value
    if (!data || data.length === 0) return

    if (direction === 'prev' && this.currentDetailIndex.value > 0) {
      this.currentDetailIndex.value--
      this.currentDetailRow.value = data[this.currentDetailIndex.value]
    } else if (direction === 'next' && this.currentDetailIndex.value < data.length - 1) {
      this.currentDetailIndex.value++
      this.currentDetailRow.value = data[this.currentDetailIndex.value]
    }
  }

  /**
   * 🔥 关闭详情抽屉
   */
  private handleCloseDetail(): void {
    this.showDetailDrawer.value = false
    this.currentDetailRow.value = null
    this.currentDetailIndex.value = -1
  }

  /**
   * 获取列宽
   */
  private getColumnWidth(field: FieldConfig): number {
    if (field.widget?.type === 'timestamp') return 180
    if (field.data?.type === 'float' || field.widget?.type === 'float') return 120
    return 100
  }

  /**
   * 格式化时间戳
   */
  private formatTimestamp(timestamp: number | string | null | undefined, format?: string): string {
    if (!timestamp) return '-'
    const date = new Date(typeof timestamp === 'string' ? parseInt(timestamp, 10) : timestamp)
    if (isNaN(date.getTime())) return '-'
    
    const formatStr = format || 'YYYY-MM-DD HH:mm:ss'
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hours = String(date.getHours()).padStart(2, '0')
    const minutes = String(date.getMinutes()).padStart(2, '0')
    const seconds = String(date.getSeconds()).padStart(2, '0')
    
    return formatStr
      .replace('YYYY', String(year))
      .replace('MM', month)
      .replace('DD', day)
      .replace('HH', hours)
      .replace('mm', minutes)
      .replace('ss', seconds)
  }

  /**
   * 格式化浮点数
   */
  private formatFloat(value: number | null | undefined): string {
    if (value === null || value === undefined) return '-'
    return Number(value).toLocaleString('zh-CN', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    })
  }

  /**
   * 🔥 处理 Form 字段点击（打开详情抽屉）
   */
  private handleFormFieldClick(field: FieldConfig, value: FieldValue): void {
    handleFormFieldClick(this.formDrawerState, field, value, 'ResponseTableWidget')
  }

  /**
   * 🔥 关闭 Form 字段详情抽屉
   */
  private handleCloseFormDetail(): void {
    handleCloseFormDetail(this.formDrawerState)
  }

  /**
   * 🔥 渲染表格单元格（使用 Widget 的 renderTableCell 方法）
   * 与 TableRenderer 保持一致，支持复杂组件（如 files、multiselect 等）
   */
  private renderTableCell(field: FieldConfig, rawValue: any): { content: any, isString: boolean } {
    try {
      // 🔥 将原始值转换为 FieldValue 格式
      const value = convertToFieldValue(rawValue, field)
      
      // 🔥 将 field 转换为 core 类型的 FieldConfig（类型兼容）
      const coreField: FieldConfig = {
        ...field,
        widget: field.widget || { type: WidgetType.INPUT, config: {} },
        data: field.data || {}
      } as FieldConfig
      
      // 🔥 如果是 Form 类型，提供可点击的查看按钮
      if (field.widget?.type === WidgetType.FORM) {
        const button = renderFormFieldButton(field, value, (e: MouseEvent) => {
          Logger.info('[ResponseTableWidget]', `点击事件触发: ${field.code}`)
          this.handleFormFieldClick(field, value)
        })
        if (button) {
          return {
            content: button,
            isString: false
          }
        }
      }
      
      // 🔥 创建临时 Widget（不需要 formManager）
      const tempWidget = WidgetBuilder.createTemporary({
        field: coreField,
        value: value
      })
      
      // 🔥 调用 Widget 的 renderTableCell() 方法（组件自治）
      const result = tempWidget.renderTableCell(value)
      
      // 🔥 统一返回格式：区分字符串和 VNode
      const isString = typeof result === 'string'
      return {
        content: result,
        isString
      }
    } catch (error) {
      Logger.error('ResponseTableWidget', `renderTableCell error for ${field.code}`, error)
      const fallbackValue = rawValue !== null && rawValue !== undefined ? String(rawValue) : '-'
      return {
        content: fallbackValue,
        isString: true
      }
    }
  }
  
  /**
   * 🔥 渲染 Form 字段详情抽屉
   * 遵循依赖倒置原则：通过回调函数注入具体的渲染逻辑
   */
  private renderFormDetailDrawer(): any {
    // 🔥 定义渲染内容的回调函数（具体实现）
    const renderContent: DrawerContentRenderer = (field, value, fieldPath) => {
      const responseWidget = new ResponseFormWidget({
        field: field,
        currentFieldPath: fieldPath,
        value: value,
        onChange: () => {},
        formManager: this.formManager,
        formRenderer: this.formRenderer,
        depth: this.depth + 1
      })
      return responseWidget.render()
    }
    
    return renderFormDetailDrawer(
      this.formDrawerState,
      this.fieldPath,
      renderContent,
      'ResponseTableWidget'
    )
  }

  /**
   * 渲染表格
   * 即使没有数据也显示表格框架结构
   */
  render(): any {
    const renderId = Math.random().toString(36).substr(2, 9)
    Logger.info('[ResponseTableWidget]', `render 开始: field=${this.field.code}, renderId=${renderId}`)
    
    const currentValue = this.getValue()
    const tableData = Array.isArray(currentValue?.raw) ? currentValue.raw : []
    
    Logger.info('[ResponseTableWidget]', `render: field=${this.field.code}, tableData.length=${tableData.length}, renderId=${renderId}`)
    
    // 获取子字段配置
    const fields: FieldConfig[] = this.field.children || []
    
    // 判断是否有实际数据
    const hasData = tableData.length > 0
    
    // 🔥 关键修复：使用 toRaw 读取响应式数据，避免触发响应式追踪
    // 这样可以防止在 render 过程中触发响应式更新，从而避免递归更新
    const rawFormDrawerState = toRaw(this.formDrawerState)
    const showDrawer = rawFormDrawerState?.showFormDetailDrawer?.value ?? false
    Logger.info('[ResponseTableWidget]', `render: field=${this.field.code}, showDrawer=${showDrawer}, renderId=${renderId}`)
    
    // 🔥 只在 showDrawer 为 true 时才读取 drawerContent，并且使用 toRaw 避免响应式追踪
    let drawer: any = null
    if (showDrawer) {
      // 🔥 使用 toRaw 读取 computed 值，避免触发响应式追踪
      const rawDrawerContent = toRaw(this.drawerContent)
      drawer = rawDrawerContent?.value ?? null
      Logger.info('[ResponseTableWidget]', `render: field=${this.field.code}, drawer存在=${!!drawer}, renderId=${renderId}`)
    } else {
      Logger.info('[ResponseTableWidget]', `render: field=${this.field.code}, drawer跳过读取(showDrawer=false), renderId=${renderId}`)
    }
    
    // 始终渲染表格（即使没有数据也显示表头结构），以及 Form 字段详情抽屉
    // 🔥 关键修复：给根元素添加稳定的 key，避免 Vue 认为需要重新创建组件
    return h('div', { 
      key: `response_table_${this.field.code}`,  // 🔥 稳定的 key
      style: { width: '100%' } 
    }, [
      h(ElTable, {
        key: `table_${this.field.code}_${tableData.length}`,  // 🔥 基于数据长度的 key
        data: tableData,
        border: true,
        style: { width: '100%' },
        maxHeight: 400,
        emptyText: hasData ? '暂无数据' : '等待数据...'
      }, {
        default: () => fields.map(field => 
          h(ElTableColumn, {
            key: field.code,
            prop: field.code,
            label: field.name,
            minWidth: this.getColumnWidth(field)
          }, {
            default: ({ row }: { row: any }) => {
              // 如果没有数据，不渲染单元格内容
              if (!hasData) return '-'
              
              const rawValue = row[field.code]
              
              // 🔥 使用 Widget 的 renderTableCell 方法（支持复杂组件）
              const cellResult = this.renderTableCell(field, rawValue)
              
              // 🔥 根据返回类型渲染：字符串或 VNode
              if (cellResult.isString) {
                return cellResult.content
              } else {
                // VNode 需要使用 component :is 渲染
                return h('div', { style: 'display: inline-block; width: 100%;' }, cellResult.content)
              }
            }
          })
        )
      }),
      // 🔥 渲染 Form 字段详情抽屉（使用 computed 确保响应式更新）
      drawer
    ])
  }
}

