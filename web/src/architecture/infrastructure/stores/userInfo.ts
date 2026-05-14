/**
 * 入口文件：统一从 userInfo/index 导出，保证 @/architecture/infrastructure/stores/userInfo 与 ./architecture/infrastructure/stores/userInfo 解析一致
 */
export { useUserInfoStore } from './userInfo/index'
