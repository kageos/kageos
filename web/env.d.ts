/// <reference types="vite/client" />

declare module 'vue' {
  interface GlobalComponents {
    ElButton: typeof import('element-plus')['ElButton']
    ElTable: typeof import('element-plus')['ElTable']
    ElTableColumn: typeof import('element-plus')['ElTableColumn']
    ElForm: typeof import('element-plus')['ElForm']
    ElFormItem: typeof import('element-plus')['ElFormItem']
    ElInput: typeof import('element-plus')['ElInput']
    ElSelect: typeof import('element-plus')['ElSelect']
    ElOption: typeof import('element-plus')['ElOption']
    ElDialog: typeof import('element-plus')['ElDialog']
    ElCard: typeof import('element-plus')['ElCard']
    ElContainer: typeof import('element-plus')['ElContainer']
    ElHeader: typeof import('element-plus')['ElHeader']
    ElAside: typeof import('element-plus')['ElAside']
    ElMain: typeof import('element-plus')['ElMain']
    ElMenu: typeof import('element-plus')['ElMenu']
    ElMenuItem: typeof import('element-plus')['ElMenuItem']
    ElSubMenu: typeof import('element-plus')['ElSubMenu']
    ElPagination: typeof import('element-plus')['ElPagination']
    ElMessage: typeof import('element-plus')['ElMessage']
    ElMessageBox: typeof import('element-plus')['ElMessageBox']
    ElNotification: typeof import('element-plus')['ElNotification']
    ElLoading: typeof import('element-plus')['ElLoading']
  }
}
