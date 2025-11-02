/**
 * ResponseTableWidget - 返回值表格组件
 * 用于渲染返回值中的 table/list 类型字段（只读展示）
 */

import { h } from 'vue'
import { ElTable, ElTableColumn } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig } from '../types/field'

export class ResponseTableWidget extends BaseWidget {
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
   */
  render(): any {
    const currentValue = this.getValue()
    const tableData = Array.isArray(currentValue?.raw) ? currentValue.raw : []
    
    // 获取子字段配置
    const fields: FieldConfig[] = this.field.children || []
    
    // 如果没有数据，显示空状态
    if (tableData.length === 0) {
      return h('div', {
        style: {
          padding: '20px',
          textAlign: 'center',
          color: 'var(--el-text-color-placeholder)'
        }
      }, '暂无数据')
    }
    
    // 渲染表格
    return h(ElTable, {
      data: tableData,
      border: true,
      style: { width: '100%' },
      maxHeight: 400
    }, {
      default: () => fields.map(field => 
        h(ElTableColumn, {
          key: field.code,
          prop: field.code,
          label: field.name,
          minWidth: this.getColumnWidth(field)
        }, {
          default: ({ row }: { row: any }) => {
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

