type XlsxModule = typeof import('xlsx')

let xlsxModulePromise: Promise<XlsxModule> | null = null

export function loadXlsx(): Promise<XlsxModule> {
  xlsxModulePromise ??= import('xlsx')
  return xlsxModulePromise
}
