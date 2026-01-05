/**
 * 组织架构管理 API
 * 
 * 需求：
 * - 获取部门树
 * - 查看部门下的用户
 * - 给用户分配组织架构（部门和 Leader）
 * - 分配 Leader
 * 
 * 设计思路：
 * - 使用树形结构展示部门层级
 * - 支持点击部门查看该部门下的用户
 * - 支持批量分配用户到部门
 * - 支持为用户分配 Leader
 */

import { get, post, put, del } from '@/utils/request'
import type { UserInfo } from '@/types'

// ==================== 类型定义 ====================

export interface Department {
  id: number
  name: string
  code: string
  parent_id: number | null // NULL 表示根部门
  full_code_path: string
  full_name_path?: string // 完整名称路径（如：技术部/后端组）
  managers: string // 部门负责人（多个用户名逗号分隔）
  description: string
  status: 'active' | 'inactive'
  sort: number
  is_system_default?: boolean // 是否为系统默认组织（不可删除）
  created_at: string
  updated_at: string
  children?: Department[] // 子部门
}

export interface CreateDepartmentReq {
  name: string
  code: string
  parent_id: number
  description?: string
}

export interface CreateDepartmentResp {
  department: Department
}

export interface UpdateDepartmentReq {
  name?: string
  description?: string
  managers?: string // 多个用户名逗号分隔
  status?: 'active' | 'inactive'
  sort?: number
}

export interface UpdateDepartmentResp {
  department: Department
}

export interface GetDepartmentTreeResp {
  departments: Department[]
}

export interface GetDepartmentResp {
  department: Department
}

// ==================== 用户分配相关 ====================

export interface AssignUserReq {
  username: string
  department_full_path?: string // 部门完整路径（可选，为空表示移除部门）
  leader_username?: string // Leader 用户名（可选，为空表示移除 Leader）
}

export interface AssignUserResp {
  user: UserInfo
}

// ==================== API 调用 ====================

/**
 * 获取部门树
 */
export function getDepartmentTree() {
  return get<GetDepartmentTreeResp>('/hr/api/v1/department/tree')
}

/**
 * 根据ID获取部门详情
 */
export function getDepartmentById(id: number) {
  return get<GetDepartmentResp>(`/hr/api/v1/department/${id}`)
}

/**
 * 根据完整路径获取部门详情
 */
export interface GetDepartmentByPathReq {
  full_code_path: string
}

export function getDepartmentByPath(fullCodePath: string) {
  // 先获取部门树，然后查找对应的部门
  return getDepartmentTree().then(res => {
    const findDepartment = (depts: Department[], path: string): Department | null => {
      for (const dept of depts) {
        if (dept.full_code_path === path) {
          return dept
        }
        if (dept.children) {
          const found = findDepartment(dept.children, path)
          if (found) return found
        }
      }
      return null
    }
    
    const department = findDepartment(res.departments, fullCodePath)
    if (!department) {
      throw new Error(`部门不存在: ${fullCodePath}`)
    }
    return { department }
  })
}

/**
 * 创建部门
 */
export function createDepartment(data: CreateDepartmentReq) {
  return post<CreateDepartmentResp>('/hr/api/v1/department', data)
}

/**
 * 更新部门
 */
export function updateDepartment(id: number, data: UpdateDepartmentReq) {
  return put<UpdateDepartmentResp>(`/hr/api/v1/department/${id}`, data)
}

/**
 * 删除部门
 */
export function deleteDepartment(id: number) {
  return del(`/hr/api/v1/department/${id}`)
}

// ==================== 用户分配 API ====================

/**
 * 分配用户组织架构（部门和 Leader）
 */
export function assignUser(data: AssignUserReq) {
  return post<AssignUserResp>('/hr/api/v1/user/assign', data)
}


/**
 * 根据部门完整路径获取用户列表
 */
export interface GetUsersByDepartmentResp {
  users: UserInfo[]
}

export function getUsersByDepartment(departmentFullPath: string) {
  return get<GetUsersByDepartmentResp>('/hr/api/v1/user/department', {
    department_full_path: departmentFullPath
  })
}

