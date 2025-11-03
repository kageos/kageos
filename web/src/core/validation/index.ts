/**
 * 验证系统统一导出
 */

import { ValidatorRegistry } from './ValidatorRegistry'
import { ValidationEngine } from './ValidationEngine'
import { RequiredValidator } from './validators/RequiredValidator'
import { MinValidator } from './validators/MinValidator'
import { MaxValidator } from './validators/MaxValidator'
import { OneOfValidator } from './validators/OneOfValidator'
import { RequiredIfValidator } from './validators/RequiredIfValidator'
import { RequiredUnlessValidator } from './validators/RequiredUnlessValidator'
import { RequiredWithValidator } from './validators/RequiredWithValidator'
import { RequiredWithoutValidator } from './validators/RequiredWithoutValidator'
import { EmailValidator } from './validators/EmailValidator'

// 导出类型和类
export { ValidatorRegistry } from './ValidatorRegistry'
export { ValidationEngine } from './ValidationEngine'
export type { Validator, ValidationRule, ValidationResult, ValidationContext } from './types'

// 导出所有验证器
export { RequiredValidator } from './validators/RequiredValidator'
export { MinValidator } from './validators/MinValidator'
export { MaxValidator } from './validators/MaxValidator'
export { OneOfValidator } from './validators/OneOfValidator'
export { RequiredIfValidator } from './validators/RequiredIfValidator'
export { RequiredUnlessValidator } from './validators/RequiredUnlessValidator'
export { RequiredWithValidator } from './validators/RequiredWithValidator'
export { RequiredWithoutValidator } from './validators/RequiredWithoutValidator'
export { EmailValidator } from './validators/EmailValidator'

// 导出验证工具函数
export * from './utils/fieldUtils'

/**
 * 创建默认的验证器注册表（包含所有已实现的验证器）
 */
export function createDefaultValidatorRegistry(): ValidatorRegistry {
  const registry = new ValidatorRegistry()
  
  // 注册基础验证器
  registry.register(new RequiredValidator())
  registry.register(new MinValidator())
  registry.register(new MaxValidator())
  registry.register(new OneOfValidator())
  
  // 注册条件验证器
  registry.register(new RequiredIfValidator())
  registry.register(new RequiredUnlessValidator())
  registry.register(new RequiredWithValidator())
  registry.register(new RequiredWithoutValidator())
  
  // 注册格式验证器
  registry.register(new EmailValidator())
  
  return registry
}

