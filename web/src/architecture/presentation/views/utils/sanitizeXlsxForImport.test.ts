import JSZip from 'jszip'
import { describe, expect, it } from 'vitest'
import { sanitizeXlsxCommentsForImport } from './sanitizeXlsxForImport'

describe('sanitizeXlsxCommentsForImport', () => {
  it('removes malformed legacy comments while keeping worksheet content', async () => {
    const zip = new JSZip()
    zip.file('[Content_Types].xml', '<Types><Override PartName="/xl/comments/comment1.xml" ContentType="comments"/></Types>')
    zip.file('xl/comments/comment1.xml', '<comments/>')
    zip.file('xl/drawings/commentsDrawing1.vml', '<xml/>')
    zip.file(
      'xl/worksheets/_rels/sheet1.xml.rels',
      '<Relationships><Relationship Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/comments" Target="/xl/comments/comment1.xml" Id="comments"/><Relationship Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/vmlDrawing" Target="/xl/drawings/commentsDrawing1.vml" Id="vml"/></Relationships>'
    )
    zip.file(
      'xl/worksheets/sheet1.xml',
      '<worksheet><sheetData><row r="1"><c r="A1"><v>1</v></c></row></sheetData><legacyDrawing r:id="vml"/></worksheet>'
    )
    const source = await zip.generateAsync({ type: 'arraybuffer' })

    const sanitized = await sanitizeXlsxCommentsForImport(source)
    const result = await JSZip.loadAsync(sanitized)

    expect(result.file('xl/comments/comment1.xml')).toBeNull()
    expect(result.file('xl/drawings/commentsDrawing1.vml')).toBeNull()
    expect(await result.file('xl/worksheets/_rels/sheet1.xml.rels')!.async('string')).not.toContain('comments')
    expect(await result.file('xl/worksheets/sheet1.xml')!.async('string')).not.toContain('legacyDrawing')
    expect(await result.file('xl/worksheets/sheet1.xml')!.async('string')).toContain('<v>1</v>')
  })
})
