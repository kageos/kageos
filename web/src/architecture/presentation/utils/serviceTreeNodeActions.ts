import type { Component } from 'vue'
import {
  ChatDotRound,
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
import type { ServiceTree } from '@/architecture/domain/types'
import { isRootNode } from '@/architecture/domain/utils/tree-utils'
import { featureFlags } from '@/architecture/shared/config/features'
import { canAdmin } from '../composables/useAccessControl'
import { translate } from '@/architecture/shared/i18n'

export type ServiceTreeNodeActionCommand =
  | 'create-directory'
  | 'create-docs'
  | 'open-workstation'
  | 'delete-directory'
  | 'rename'
  | 'copy'
  | 'export-json'
  | 'import-directory'
  | 'paste'
  | 'delete-function'
  | 'delete-doc'
  | 'manage-access'
  | 'update-history'

export interface ServiceTreeNodeAction {
  command: ServiceTreeNodeActionCommand
  label: string
  icon: Component
  visible: boolean
}

export interface ServiceTreeNodeActionOptions {
  hasCopiedDirectory?: boolean
}

export function getServiceTreeNodeActions(
  data: ServiceTree,
  options: ServiceTreeNodeActionOptions = {}
): ServiceTreeNodeAction[] {
  const actions: ServiceTreeNodeAction[] = [
    {
      command: 'create-directory',
      label: translate('serviceTree.addDirectory'),
      icon: Plus,
      visible: data.type === 'package'
    },
    {
      command: 'create-docs',
      label: translate('serviceTree.createDocs'),
      icon: Document,
      visible: data.type === 'package'
    },
    {
      command: 'open-workstation',
      label: translate('serviceTree.openWorkbench'),
      icon: ChatDotRound,
      visible: data.type === 'package'
    },
    {
      command: 'manage-access',
      label: translate('serviceTree.manageAccess'),
      icon: Key,
      visible: Boolean(data.full_code_path) && canAdmin(data)
    },
    {
      command: 'delete-directory',
      label: translate('serviceTree.deleteDirectory'),
      icon: Delete,
      visible: data.type === 'package' && !isRootNode(data)
    },
    {
      command: 'rename',
      label: translate('serviceTree.rename'),
      icon: Edit,
      visible: data.type === 'package'
    },
    {
      command: 'copy',
      label: translate('serviceTree.copy'),
      icon: CopyDocument,
      visible: data.type === 'package'
    },
    {
      command: 'export-json',
      label: translate('serviceTree.exportBundle'),
      icon: Download,
      visible: featureFlags.capabilityBundle && data.type === 'package'
    },
    {
      command: 'import-directory',
      label: translate('serviceTree.importBundle'),
      icon: Upload,
      visible: featureFlags.capabilityBundle && data.type === 'package'
    },
    {
      command: 'paste',
      label: translate('serviceTree.paste'),
      icon: DocumentChecked,
      visible: data.type === 'package'
        && Boolean(options.hasCopiedDirectory)
    },
    {
      command: 'delete-function',
      label: translate('serviceTree.deleteFunction'),
      icon: Delete,
      visible: data.type === 'function'
    },
    {
      command: 'delete-doc',
      label: translate('serviceTree.deleteDoc'),
      icon: Delete,
      visible: data.type === 'docs'
    },
  ]

  return actions.filter((action) => action.visible)
}

export function buildServiceTreeNodeActionTestId(
  command: ServiceTreeNodeActionCommand,
  data: ServiceTree
): string {
  return `service-tree-action-${command}-${data.id}`
}
