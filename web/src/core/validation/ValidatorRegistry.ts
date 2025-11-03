/**
 * 验证器注册表
 * 
 * 使用策略模式管理所有验证器
 */

import type { Validator } from './types'

export class ValidatorRegistry {
  private validators = new Map<string, Validator>()
  
  /**
   * 注册验证器
   */
  register(validator: Validator): void {
    this.validators.set(validator.name, validator)
  }
  
  /**
   * 获取验证器
   */
  get(name: string): Validator | undefined {
    return this.validators.get(name)
  }
  
  /**
   * 获取所有已注册的验证器名称
   */
  getRegisteredNames(): string[] {
    return Array.from(this.validators.keys())
  }
  
  /**
   * 检查验证器是否已注册
   */
  has(name: string): boolean {
    return this.validators.has(name)
  }
}

