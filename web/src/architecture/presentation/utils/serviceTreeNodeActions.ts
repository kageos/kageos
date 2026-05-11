import type { Component } from 'vue'
import {
  ChatDotRound,
  ChatDotSquare,
  Connection,
  CopyDocument,
  Delete,
  Document,
  DocumentChecked,
  Download,
  Edit,
  Key,
  Plus,
  Upload
} from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'
import { DirectoryPermission, TablePermission, WorkflowPermission, hasPermission } from '@/utils/permission'
import { isRootNode } from '@/utils/tree-utils'

export type ServiceTreeNodeActionCommand =
  | 'apply-permission'
  | 'create-directory'
  | 'create-docs'
  | 'create-board'
  | 'create-workflow'
  | 'open-workstation'
  | 'delete-directory'
  | 'rename'
  | 'copy'
  | 'export-json'
  | 'import-json'
  | 'paste'
  | 'delete-function'
  | 'delete-doc'
  | 'delete-board'
  | 'delete-workflow'
  | 'publish-to-hub'
  | 'push-to-hub'
  | 'update-history'
  | 'approve-permission'
  | 'manage-permission'

export interface ServiceTreeNodeAction {
  command: ServiceTreeNodeActionCommand
  label: string
  icon: Component
  visible: boolean
}

export interface ServiceTreeNodeActionOptions {
  hasCopiedDirectory?: boolean
  hasCopiedHubLink?: boolean
}

export function getServiceTreeNodeActions(
  data: ServiceTree,
  options: ServiceTreeNodeActionOptions = {}
): ServiceTreeNodeAction[] {
  const actions: ServiceTreeNodeAction[] = [
    { command: 'apply-permission', label: '申请权限', icon: Key, visible: true },
    {
      command: 'create-directory',
      label: '添加服务目录',
      icon: Plus,
      visible: data.type === 'package' && hasPermission(data, DirectoryPermission.write)
    },
    {
      command: 'create-docs',
      label: '创建文档',
      icon: Document,
      visible: data.type === 'package' && hasPermission(data, DirectoryPermission.write)
    },
    {
      command: 'create-board',
      label: '新增讨论区',
      icon: ChatDotSquare,
      visible: data.type === 'package' && hasPermission(data, DirectoryPermission.write)
    },
    {
      command: 'create-workflow',
      label: '新增工作流',
      icon: Connection,
      visible: data.type === 'package' && hasPermission(data, DirectoryPermission.write)
    },
    {
      command: 'open-workstation',
      label: '打开工作台',
      icon: ChatDotRound,
      visible: data.type === 'package'
    },
    {
      command: 'delete-directory',
      label: '删除目录',
      icon: Delete,
      visible: data.type === 'package' && !isRootNode(data) && hasPermission(data, DirectoryPermission.delete)
    },
    {
      command: 'rename',
      label: '重命名',
      icon: Edit,
      visible: (data.type === 'package' && hasPermission(data, DirectoryPermission.update))
        || (data.type === 'workflow' && hasPermission(data, WorkflowPermission.update))
    },
    {
      command: 'copy',
      label: '复制',
      icon: CopyDocument,
      visible: data.type === 'package' && hasPermission(data, DirectoryPermission.read)
    },
    {
      command: 'export-json',
      label: '导出能力包',
      icon: Download,
      visible: data.type === 'package' && hasPermission(data, DirectoryPermission.read)
    },
    {
      command: 'import-json',
      label: '导入能力包',
      icon: Upload,
      visible: data.type === 'package' && hasPermission(data, DirectoryPermission.write)
    },
    {
      command: 'paste',
      label: '粘贴',
      icon: DocumentChecked,
      visible: data.type === 'package'
        && (Boolean(options.hasCopiedDirectory) || Boolean(options.hasCopiedHubLink))
        && hasPermission(data, DirectoryPermission.write)
    },
    {
      command: 'delete-function',
      label: '删除函数',
      icon: Delete,
      visible: data.type === 'function' && hasPermission(data, TablePermission.delete)
    },
    {
      command: 'delete-doc',
      label: '删除文档',
      icon: Delete,
      visible: data.type === 'docs' && hasPermission(data, DirectoryPermission.delete)
    },
    {
      command: 'delete-board',
      label: '删除讨论区',
      icon: Delete,
      visible: data.type === 'board' && hasPermission(data, DirectoryPermission.delete)
    },
    {
      command: 'delete-workflow',
      label: '删除工作流',
      icon: Delete,
      visible: data.type === 'workflow' && hasPermission(data, WorkflowPermission.delete)
    },
    {
      command: 'publish-to-hub',
      label: '发布到 Hub',
      icon: Upload,
      visible: data.type === 'package' && !data.hub_full_code_path && hasPermission(data, DirectoryPermission.read)
    },
    {
      command: 'push-to-hub',
      label: '推送到 Hub',
      icon: Upload,
      visible: data.type === 'package' && Boolean(data.hub_full_code_path) && hasPermission(data, DirectoryPermission.write)
    }
  ]

  return actions.filter((action) => action.visible)
}

export function buildServiceTreeNodeActionTestId(
  command: ServiceTreeNodeActionCommand,
  data: ServiceTree
): string {
  return `service-tree-action-${command}-${data.id}`
}
