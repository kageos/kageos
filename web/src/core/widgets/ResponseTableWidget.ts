/**
 * ResponseTableWidget - 返回值表格组件
 * 用于渲染返回值中的 table/list 类型字段（只读展示）
 * 
 * 功能特性：
 * - 点击 ID 列查看详情
 * - 详情抽屉导航（上一个/下一个）
 * - 只读展示，无编辑功能
 */

import { h, ref } from 'vue'
import { ElTable, ElTableColumn, ElDrawer, ElButton, ElIcon, ElDescriptions, ElDescriptionsItem } from 'element-plus'
import { ArrowLeft, ArrowRight, Close } from '@element-plus/icons-vue'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig } from '../types/field'

export class ResponseTableWidget extends BaseWidget {
  // 🔥 详情抽屉状态
  private showDetailDrawer = ref(false)
  private currentDetailRow = ref<any>(null)
  private currentDetailIndex = ref<number>(-1)
  private tableData = ref<any[]>([])
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
   * 渲染表格
   * 即使没有数据也显示表格框架结构
   */
  render(): any {
    const currentValue = this.getValue()
    const tableData = Array.isArray(currentValue?.raw) ? currentValue.raw : []
    
    // 获取子字段配置
    const fields: FieldConfig[] = this.field.children || []
    
    // 判断是否有实际数据
    const hasData = tableData.length > 0
    
    // 始终渲染表格（即使没有数据也显示表头结构）
    return h(ElTable, {
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
            
            const value = row[field.code]
            
            // 根据字段类型格式化显示
            if (field.widget?.type === 'timestamp') {
              return this.formatTimestamp(value, field.widget.config?.format)
            } else if (field.widget?.type === 'float' || field.data?.type === 'float') {
              return this.formatFloat(value)
            }
            
            return value !== undefined && value !== null ? String(value) : '-'
          }
        })
      )
    })
  }
}

