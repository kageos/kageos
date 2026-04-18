/**
 * 从工具调用的 arguments + result 中提取需要在前端直接展示的字段。
 *
 * 约定：LLM 在调用 run_form_submit 等工具时可传 output_display 参数，
 * 格式为 { "展示标签": "result字段名" }。
 * 前端从 arguments 解析 output_display 映射，再从 result 中提取对应值，
 * 生成 OutputDisplayField[] 供 UI 组件渲染（可复制、可展开）。
 *
 * 同时兼容 result 自带 _display_outputs 元数据（方案 B 预留）。
 */

export interface OutputDisplayField {
  label: string
  fieldKey: string
  value: string
  type: 'text' | 'number' | 'json'
}

function inferType(val: unknown): 'text' | 'number' | 'json' {
  if (typeof val === 'number') return 'number'
  if (typeof val === 'string') return 'text'
  return 'json'
}

function valueToString(val: unknown): string {
  if (val == null) return ''
  if (typeof val === 'string') return val
  if (typeof val === 'number' || typeof val === 'boolean') return String(val)
  try {
    return JSON.stringify(val, null, 2)
  } catch {
    return String(val)
  }
}

/**
 * 从单次工具调用中提取需要展示的字段。
 * @param arguments_ 工具调用参数 JSON 字符串（含 output_display）
 * @param result 工具调用结果 JSON 字符串
 */
export function extractDisplayFieldsFromToolCall(
  arguments_?: string,
  result?: string,
  resultData?: unknown
): OutputDisplayField[] {
  let resultObj: Record<string, unknown>
  if (resultData && typeof resultData === 'object' && !Array.isArray(resultData)) {
    resultObj = resultData as Record<string, unknown>
  } else {
    if (!result) return []
    try {
      resultObj = JSON.parse(result) as Record<string, unknown>
    } catch {
      return []
    }
  }

  const fields: OutputDisplayField[] = []

  // 来源 1：result 自带 _display_outputs 元数据（方案 B 预留）
  const builtinMeta = resultObj['_display_outputs']
  if (builtinMeta && typeof builtinMeta === 'object' && !Array.isArray(builtinMeta)) {
    const meta = builtinMeta as Record<string, unknown>
    for (const [fieldKey, cfg] of Object.entries(meta)) {
      if (!cfg || typeof cfg !== 'object') continue
      const c = cfg as Record<string, unknown>
      const label = (typeof c.label === 'string' ? c.label : fieldKey) as string
      const val = resultObj[fieldKey]
      if (val != null) {
        fields.push({ label, fieldKey, value: valueToString(val), type: inferType(val) })
      }
    }
  }

  // 来源 2：LLM 参数中的 output_display
  if (arguments_) {
    let argsObj: Record<string, unknown>
    try {
      argsObj = JSON.parse(arguments_) as Record<string, unknown>
    } catch {
      return fields
    }
    const outputDisplay = argsObj['output_display']
    if (outputDisplay && typeof outputDisplay === 'object' && !Array.isArray(outputDisplay)) {
      const mapping = outputDisplay as Record<string, unknown>
      const existingKeys = new Set(fields.map(f => f.fieldKey))
      for (const [label, fieldKeyRaw] of Object.entries(mapping)) {
        const fieldKey = typeof fieldKeyRaw === 'string' ? fieldKeyRaw : String(fieldKeyRaw)
        if (existingKeys.has(fieldKey)) continue
        const val = resultObj[fieldKey]
        if (val != null) {
          fields.push({ label, fieldKey, value: valueToString(val), type: inferType(val) })
        }
      }
    }
  }

  return fields
}

/**
 * 批量：从多个 tool_call 中提取所有需要展示的字段。
 */
export function extractAllDisplayFields(
  calls: Array<{ arguments?: string; result?: string; result_data?: unknown }>
): OutputDisplayField[] {
  const all: OutputDisplayField[] = []
  for (const tc of calls) {
    all.push(...extractDisplayFieldsFromToolCall(tc.arguments, tc.result, tc.result_data))
  }
  return all
}
