/**
 * Excel 导入工具函数
 * 用于 TableView 和 TableWidget 的批量导入功能
 */

import * as XLSX from 'xlsx'
import type { FieldConfig } from '@/core/types/field'

export interface ImportError {
  index: number
  field: string
  error: string
}

export interface ImportResult {
  data: any[]
  errors: ImportError[]
}

/**
 * 解析 Excel 文件
 * @param file Excel 文件
 * @param fields 字段配置（用于数据映射和验证）
 * @returns 解析结果
 */
export function parseExcelFile(file: File, fields: FieldConfig[]): Promise<ImportResult> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = (e) => {
      try {
        console.log('[ExcelImport] 开始解析 Excel 文件')
        console.log('[ExcelImport] fields 参数:', fields)
        console.log('[ExcelImport] fields 长度:', fields?.length)
        console.log('[ExcelImport] fields 中是否有 null:', fields?.some(f => f === null || f === undefined))
        
        const data = new Uint8Array(e.target?.result as ArrayBuffer)
        const workbook = XLSX.read(data, { type: 'array' })
        
        // 获取第一个工作表
        const firstSheetName = workbook.SheetNames[0]
        const worksheet = workbook.Sheets[firstSheetName]
        
        // 转换为 JSON（第一行作为键名）
        const jsonData = XLSX.utils.sheet_to_json(worksheet, { header: 1, defval: '' })
        
        if (jsonData.length < 2) {
          reject(new Error('Excel 文件格式错误：至少需要 2 行（字段名称、数据）'))
          return
        }
        
        // 第一行：字段名称（中文）
        // 第二行开始：示例数据（跳过）和数据行
        const fieldNames = jsonData[0] as string[]
        console.log('[ExcelImport] 第一行字段名称:', fieldNames)
        
        // 根据字段名称匹配字段配置，构建字段映射（字段 code -> 列索引）
        const fieldCodeMap = new Map<string, number>()
        fieldNames.forEach((name, index) => {
          if (name) {
            // 根据字段名称（中文）查找对应的字段配置
            console.log(`[ExcelImport] 查找字段名称 "${name}" 的配置`)
            const matchedField = fields.find(field => {
              if (!field) {
                console.log(`[ExcelImport] 发现 null 字段，跳过`)
                return false
              }
              return field.name === name || field.name === String(name).trim()
            })
            console.log(`[ExcelImport] 匹配结果:`, matchedField)
            if (matchedField) {
              if (!matchedField.code) {
                console.error(`[ExcelImport] 警告：字段 "${matchedField.name}" 没有 code 属性`, matchedField)
              } else {
                fieldCodeMap.set(matchedField.code, index)
                console.log(`[ExcelImport] 设置字段映射: ${matchedField.code} -> ${index}`)
              }
            } else {
              console.log(`[ExcelImport] 未找到字段名称 "${name}" 的配置`)
            }
          }
        })
        console.log('[ExcelImport] 字段映射完成:', Array.from(fieldCodeMap.entries()))
        
        // 从第二行开始都是数据行（包括示例数据行，会在后续过滤）
        const dataRows = jsonData.slice(1) as any[][]
        
        // 转换数据
        const convertedData: any[] = []
        const errors: ImportError[] = []
        
        dataRows.forEach((row, rowIndex) => {
          // 跳过空行
          if (row.every(cell => !cell || cell.toString().trim() === '')) {
            return
          }
          
          // Excel 实际行号（从1开始，第一行是表头，所以数据行从2开始）
          const excelRowNumber = rowIndex + 2
          
          // 检查是否是示例数据行：如果这一行的所有值都是示例值，跳过
          const nonEmptyCells = row.filter(cell => cell && cell.toString().trim() !== '')
          if (nonEmptyCells.length > 0) {
            // 检查这一行是否包含示例数据特征
            let exampleCellCount = 0
            let totalCellCount = 0
            
            row.forEach((cell, colIndex) => {
              if (cell !== null && cell !== undefined && cell.toString().trim() !== '') {
                totalCellCount++
                const cellStr = cell.toString().trim()
                // 检查是否是示例文本格式
                const isExampleValue = cellStr.includes('示例文本') || 
                                       (cellStr.includes('选项') && /选项\d+/.test(cellStr)) || 
                                       cellStr === '是' || 
                                       cellStr === '否' ||
                                       cellStr === '2024-01-01' ||
                                       cellStr === '123' ||
                                       /^示例文本\d+$/.test(cellStr) // 匹配"示例文本1"、"示例文本2"等
                
                if (isExampleValue) {
                  exampleCellCount++
                }
              }
            })
            
            // 如果这一行大部分（超过80%）都是示例数据，跳过
            if (totalCellCount > 0 && exampleCellCount / totalCellCount >= 0.8) {
              return
            }
          }
          
          const rowData: any = {}
          let hasError = false
          let hasData = false // 标记是否有实际数据（业务字段）
          let hasBusinessData = false // 标记是否有业务数据（非系统字段）
          
          // 根据字段代码映射数据
          fields.forEach((field, fieldIndex) => {
            // 跳过 null 或 undefined 的字段
            if (!field) {
              console.error(`[ExcelImport] 第 ${excelRowNumber} 行，字段索引 ${fieldIndex} 为 null 或 undefined`)
              return
            }
            if (!field.code) {
              console.error(`[ExcelImport] 第 ${excelRowNumber} 行，字段 "${field.name || '未知'}" (索引 ${fieldIndex}) 没有 code 属性`, field)
              return
            }
            console.log(`[ExcelImport] 处理字段: ${field.name} (code: ${field.code})`)
            
            // 检查是否是系统字段
            const isSystemField = field.code === 'created_at' || field.code === 'create_by' || 
                                  field.code === 'updated_at' || field.code === 'updated_by'
            
            const colIndex = fieldCodeMap.get(field.code)
            let finalValue: any = null
            
            if (colIndex !== undefined && colIndex < row.length) {
              const cellValue = row[colIndex]
              // 转换数据类型
              const convertedValue = convertFieldValue(field, cellValue)
              
              // 处理 $me 占位符（创建用户/更新用户字段）
              // 注意：$me 应该在提交前处理，这里先保留原值
              finalValue = convertedValue
              // 如果遇到 $me，先保留，在提交前统一处理
              if ((field.code === 'create_by' || field.code === 'updated_by') && 
                  typeof convertedValue === 'string' && convertedValue === '$me') {
                // $me 占位符保留，在提交前处理
                finalValue = '$me'
              }
            }
            
            // 对于系统字段，即使 Excel 中为空，也要添加默认值（在提交前处理）
            // 对于非系统字段，只有非空值才添加
            if (isSystemField) {
              // 系统字段：即使为空也要包含（提交前会设置默认值）
              rowData[field.code] = finalValue !== null && finalValue !== undefined && finalValue !== '' 
                ? finalValue 
                : null // 空值保留为 null，提交前会设置默认值
              // 系统字段不算业务数据
            } else if (finalValue !== null && finalValue !== undefined && finalValue !== '') {
              // 非系统字段：只有非空值才添加
              hasData = true
              hasBusinessData = true // 标记有业务数据
              rowData[field.code] = finalValue
              
              // 所有值都要进行验证（包括示例值）
              const validationError = validateFieldValue(field, finalValue)
              if (validationError) {
                errors.push({
                  index: excelRowNumber, // Excel 行号
                  field: field.name,
                  error: validationError
                })
                hasError = true
              }
            } else if (isFieldRequired(field)) {
              // 必填字段缺失
              errors.push({
                index: excelRowNumber, // Excel 行号
                field: field.name,
                error: '必填字段不能为空'
              })
              hasError = true
            }
          })
          
          // 只有当有业务数据（非系统字段）时才添加到结果中
          // 如果只有系统字段有值，但业务字段都是空的，跳过这一行
          if (hasBusinessData) {
            convertedData.push(rowData)
          } else {
            // 如果没有业务数据，跳过这一行（可能是空行或只有系统字段的行）
            console.log(`[ExcelImport] 跳过 Excel 第 ${excelRowNumber} 行 (无业务数据)`)
          }
        })
        
        resolve({
          data: convertedData,
          errors
        })
      } catch (error: any) {
        reject(new Error(`解析 Excel 文件失败: ${error.message || '未知错误'}`))
      }
    }
    reader.onerror = () => {
      reject(new Error('读取文件失败'))
    }
    reader.readAsArrayBuffer(file)
  })
}

/**
 * 转换字段值（根据 data.type 转换）
 * 只使用 widget.go 中定义的数据类型：
 * - string
 * - int
 * - bool
 * - []string
 * - []int
 * - []float
 * - float
 * - struct
 * - []struct
 */
function convertFieldValue(field: FieldConfig, value: any): any {
  // 处理空值：null、undefined、空字符串、只有空格的字符串
  if (value === null || value === undefined || value === '') {
    return null
  }
  
  // 如果是字符串，先去除首尾空格
  if (typeof value === 'string') {
    value = value.trim()
    if (value === '') {
      return null
    }
  }
  
  const dataType = (field.data as any)?.type || 'string'
  const widgetType = field.widget?.type || 'input'
  const fieldCode = field.code || ''
  
  // 如果是创建时间/更新时间字段，且是 timestamp widget，需要特殊处理
  if ((fieldCode === 'created_at' || fieldCode === 'updated_at') && widgetType === 'timestamp') {
    if (typeof value === 'string' && value.trim() !== '') {
      // 尝试解析 2006-01-02 15:04:05 格式
      const dateStr = value.trim()
      // 匹配格式：2006-01-02 15:04:05 或 2006-01-02 15:04:05.000
      const dateMatch = dateStr.match(/^(\d{4})-(\d{2})-(\d{2})\s+(\d{2}):(\d{2}):(\d{2})(?:\.(\d{3}))?$/)
      if (dateMatch) {
        try {
          // 手动解析：年-月-日 时:分:秒
          const year = parseInt(dateMatch[1], 10)
          const month = parseInt(dateMatch[2], 10) - 1 // JavaScript 月份从 0 开始
          const day = parseInt(dateMatch[3], 10)
          const hour = parseInt(dateMatch[4], 10)
          const minute = parseInt(dateMatch[5], 10)
          const second = parseInt(dateMatch[6], 10)
          
          // 创建 Date 对象（本地时间）
          const date = new Date(year, month, day, hour, minute, second)
          if (!isNaN(date.getTime())) {
            // 转换为毫秒时间戳
            return date.getTime()
          }
        } catch (e) {
          // 解析失败，继续尝试其他方式
          console.warn('[ExcelImport] 时间解析失败:', dateStr, e)
        }
      }
      // 尝试其他常见格式
      try {
        const date = new Date(dateStr)
        if (!isNaN(date.getTime())) {
          return date.getTime()
        }
      } catch (e) {
        // 解析失败，继续
      }
    }
    // 如果不是标准格式，尝试直接解析为数字（可能是时间戳）
    if (typeof value === 'number') {
      return value
    }
    // 如果都失败了，返回 null（让后续逻辑处理）
  }
  
  // 根据 data.type 转换数据类型（只使用 widget.go 中定义的类型）
  switch (dataType) {
    case 'int':
    case 'int64':
      // 如果是创建时间/更新时间字段，且已经解析成功，直接返回（不会执行到这里）
      // 如果是其他 int 字段，正常转换
      const num = Number(value)
      return isNaN(num) ? null : num
      
    case 'float':
      const floatNum = Number(value)
      return isNaN(floatNum) ? null : floatNum
      
    case 'bool':
      if (typeof value === 'boolean') return value
      if (typeof value === 'string') {
        const lower = value.toLowerCase()
        return lower === 'true' || lower === '1' || lower === '是' || lower === 'yes'
      }
      return Boolean(value)
      
    case '[]string':
      // 字符串数组：支持逗号分隔的字符串
      if (Array.isArray(value)) return value
      if (typeof value === 'string') {
        return value.split(',').map(v => v.trim()).filter(Boolean)
      }
      return [value]
      
    case '[]int':
      // 整数数组：支持逗号分隔的字符串，转换为数字数组
      if (Array.isArray(value)) {
        return value.map(v => {
          const num = Number(v)
          return isNaN(num) ? null : num
        }).filter(v => v !== null)
      }
      if (typeof value === 'string') {
        return value.split(',').map(v => {
          const num = Number(v.trim())
          return isNaN(num) ? null : num
        }).filter(v => v !== null)
      }
      return [Number(value)]
      
    case '[]float':
      // 浮点数数组：支持逗号分隔的字符串，转换为浮点数数组
      if (Array.isArray(value)) {
        return value.map(v => {
          const num = Number(v)
          return isNaN(num) ? null : num
        }).filter(v => v !== null)
      }
      if (typeof value === 'string') {
        return value.split(',').map(v => {
          const num = Number(v.trim())
          return isNaN(num) ? null : num
        }).filter(v => v !== null)
      }
      return [Number(value)]
      
    case 'struct':
    case '[]struct':
      // 结构体类型：保持原样（通常是 JSON 字符串，后端会处理）
      if (typeof value === 'string') {
        try {
          // 尝试解析 JSON
          return JSON.parse(value)
        } catch {
          // 解析失败，返回原字符串
          return value
        }
      }
      return value
      
    case 'string':
    default:
      // 字符串类型：保持原样，但如果是 multiselect widget，需要特殊处理
      if (widgetType === 'multiselect') {
        // multiselect 但 data.type 是 string，需要转换为逗号分隔的字符串
        if (Array.isArray(value)) {
          return value.join(',')
        }
        if (typeof value === 'string') {
          // 已经是字符串，直接返回
          return value
        }
      }
      // timestamp widget：如果是创建时间/更新时间字段，需要转换为毫秒时间戳
      if (widgetType === 'timestamp') {
        if (typeof value === 'string' && value.trim() !== '') {
          // 尝试解析 2006-01-02 15:04:05 格式
          const dateStr = value.trim()
          // 匹配格式：2006-01-02 15:04:05 或 2006-01-02 15:04:05.000
          const dateMatch = dateStr.match(/^(\d{4})-(\d{2})-(\d{2})\s+(\d{2}):(\d{2}):(\d{2})(?:\.(\d{3}))?$/)
          if (dateMatch) {
            try {
              // 手动解析：年-月-日 时:分:秒
              const year = parseInt(dateMatch[1], 10)
              const month = parseInt(dateMatch[2], 10) - 1 // JavaScript 月份从 0 开始
              const day = parseInt(dateMatch[3], 10)
              const hour = parseInt(dateMatch[4], 10)
              const minute = parseInt(dateMatch[5], 10)
              const second = parseInt(dateMatch[6], 10)
              
              // 创建 Date 对象（本地时间）
              const date = new Date(year, month, day, hour, minute, second)
              if (!isNaN(date.getTime())) {
                // 转换为毫秒时间戳
                return date.getTime()
              }
            } catch (e) {
              // 解析失败，返回原值
              console.warn('[ExcelImport] 时间解析失败:', dateStr, e)
            }
          } else {
            // 尝试其他常见格式
            try {
              const date = new Date(dateStr)
              if (!isNaN(date.getTime())) {
                return date.getTime()
              }
            } catch (e) {
              // 解析失败，返回原值
            }
          }
        }
        // 如果不是标准格式，尝试直接解析为数字（可能是时间戳）
        if (typeof value === 'number') {
          return value
        }
        // 其他情况返回原值
        return value.toString()
      }
      // 其他字符串类型保持原样
      return value.toString()
  }
}

/**
 * 验证字段值
 */
function validateFieldValue(field: FieldConfig, value: any): string | null {
  // 必填验证
  if (isFieldRequired(field)) {
    if (value === null || value === undefined || value === '' || 
        (Array.isArray(value) && value.length === 0)) {
      return '必填字段不能为空'
    }
  }
  
  // 类型验证（只使用 widget.go 中定义的类型）
  const dataType = (field.data as any)?.type || 'string'
  
  if (value !== null && value !== undefined && value !== '') {
    switch (dataType) {
      case 'int':
        if (isNaN(Number(value))) {
          return '必须是整数'
        }
        break
        
      case 'float':
        if (isNaN(Number(value))) {
          return '必须是数字'
        }
        break
        
      case 'bool':
        if (typeof value !== 'boolean') {
          return '必须是布尔值'
        }
        break
        
      case '[]string':
      case '[]int':
      case '[]float':
        if (!Array.isArray(value)) {
          return '必须是数组'
        }
        break
    }
  }
  
  // 长度验证（只对非空值进行验证）
  const validation = field.validation
  if (validation && typeof value === 'string' && value !== '') {
    if (validation.includes('min=')) {
      const minMatch = validation.match(/min=(\d+)/)
      if (minMatch && value.length < Number(minMatch[1])) {
        return `长度不能少于 ${minMatch[1]} 个字符`
      }
    }
    if (validation.includes('max=')) {
      const maxMatch = validation.match(/max=(\d+)/)
      if (maxMatch && value.length > Number(maxMatch[1])) {
        return `长度不能超过 ${maxMatch[1]} 个字符`
      }
    }
  }
  
  return null
}

/**
 * 检查字段是否必填
 */
function isFieldRequired(field: FieldConfig): boolean {
  return field.validation?.includes('required') || false
}

