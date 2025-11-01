/**
 * TextAreaWidget - 多行文本输入组件
 */

import { h } from 'vue'
import { ElInput } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig } from '../types/field'
import type { WidgetRenderProps } from '../types/widget'

/**
 * TextArea 配置
 */
export interface TextAreaConfig {
  placeholder?: string
  rows?: number
  autosize?: boolean | { minRows?: number; maxRows?: number }
  maxlength?: number
  showWordLimit?: boolean
  [key: string]: any
}

export class TextAreaWidget extends BaseWidget {
  private textAreaConfig: TextAreaConfig

  constructor(props: WidgetRenderProps) {
    super(props)
    
    // 解析 TextArea 配置
    this.textAreaConfig = (this.field.widget?.config as TextAreaConfig) || {}
  }

  /**
   * 处理值变化
   */
  private handleInput(value: string): void {
    this.updateRawValue(value)
  }

  /**
   * 渲染组件
   */
  render() {
    const currentValue = this.getValue()
    
    return h(ElInput, {
      type: 'textarea',
      modelValue: currentValue?.raw || '',
      placeholder: this.textAreaConfig.placeholder || `请输入${this.field.name}`,
      rows: this.textAreaConfig.rows || 3,
      autosize: this.textAreaConfig.autosize,
      maxlength: this.textAreaConfig.maxlength,
      showWordLimit: this.textAreaConfig.showWordLimit,
      'onUpdate:modelValue': (value: string) => this.handleInput(value),
      disabled: this.field.table_permission === 'read'
    })
  }

  /**
   * 捕获组件数据（TextArea 没有额外数据）
   */
  protected captureComponentData(): null {
    return null
  }

  /**
   * 恢复组件数据（TextArea 没有额外数据）
   */
  protected restoreComponentData(data: any): void {
    // 无需恢复
  }
}

