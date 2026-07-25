const COMMENT_RELATIONSHIP = /<Relationship\b(?=[^>]*\bType="[^"]*\/(?:comments|vmlDrawing)")[^>]*\/>/g
const LEGACY_DRAWING = /<legacyDrawing\b[^>]*\/>/g
const COMMENT_CONTENT_TYPE = /<Override\b(?=[^>]*\bPartName="\/xl\/comments(?:\/[^"]+|\d+\.xml)")[^>]*\/>/g

const isCommentPart = (path: string): boolean => {
  return /^xl\/comments(?:\/|\d+\.xml$)/.test(path)
    || /^xl\/drawings\/(?:commentsDrawing|vmlDrawing)\d+\.vml$/.test(path)
}

/**
 * ExcelJS 的浏览器构建会把旧式批注写到一个自己无法再次读取的路径。
 * 表格导入只关心单元格值，因此失败重试时在内存副本中移除批注关系，源文件不会被修改。
 */
export const sanitizeXlsxCommentsForImport = async (source: ArrayBuffer): Promise<Uint8Array> => {
  const { default: JSZip } = await import('jszip')
  const zip = await JSZip.loadAsync(source)

  for (const path of Object.keys(zip.files)) {
    if (isCommentPart(path)) zip.remove(path)
  }

  const relationshipPaths = Object.keys(zip.files).filter((path) => (
    /^xl\/worksheets\/_rels\/[^/]+\.rels$/.test(path)
  ))
  for (const path of relationshipPaths) {
    const file = zip.file(path)
    if (!file) continue
    const xml = await file.async('string')
    zip.file(path, xml.replace(COMMENT_RELATIONSHIP, ''))
  }

  const worksheetPaths = Object.keys(zip.files).filter((path) => (
    /^xl\/worksheets\/[^/]+\.xml$/.test(path)
  ))
  for (const path of worksheetPaths) {
    const file = zip.file(path)
    if (!file) continue
    const xml = await file.async('string')
    zip.file(path, xml.replace(LEGACY_DRAWING, ''))
  }

  const contentTypes = zip.file('[Content_Types].xml')
  if (contentTypes) {
    const xml = await contentTypes.async('string')
    zip.file('[Content_Types].xml', xml.replace(COMMENT_CONTENT_TYPE, ''))
  }

  return zip.generateAsync({ type: 'uint8array' })
}
