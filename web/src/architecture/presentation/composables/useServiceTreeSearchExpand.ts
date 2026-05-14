import { computed, nextTick, ref, watch, type ComputedRef } from 'vue'
import { findNodeByPath, findPathToNode, expandPathAndSelect, expandPathOnly } from '@/utils/serviceTreeUtils'
import type { ServiceTree } from '@/architecture/domain/types'
import { Logger } from '@/architecture/runtime/utils/logger'

export interface UseServiceTreeSearchExpandOptions {
  treeData: ComputedRef<ServiceTree[]>
  expandedKeys: ComputedRef<number[] | undefined>
}

function collectVisibleNodeIds(nodes: ServiceTree[], keyword: string): Set<number> {
  const visibleIds = new Set<number>()
  if (!keyword || !nodes.length) {
    return visibleIds
  }

  const normalizedKeyword = keyword.trim().toLowerCase()
  if (!normalizedKeyword) {
    return visibleIds
  }

  const match = (node: ServiceTree): boolean => {
    const name = (node.name || '').toLowerCase()
    const code = (node.code || '').toLowerCase()
    const path = (node.full_code_path || '').toLowerCase()
    return name.includes(normalizedKeyword)
      || code.includes(normalizedKeyword)
      || path.includes(normalizedKeyword)
  }

  const walk = (nodeList: ServiceTree[]): boolean => {
    let hasMatchInSubtree = false
    for (const node of nodeList) {
      const nodeId = Number(node.id)
      const selfMatch = match(node)
      const childMatch = node.children?.length ? walk(node.children) : false
      if (selfMatch || childMatch) {
        visibleIds.add(nodeId)
        hasMatchInSubtree = true
      }
    }
    return hasMatchInSubtree
  }

  walk(nodes)
  return visibleIds
}

export function useServiceTreeSearchExpand(options: UseServiceTreeSearchExpandOptions) {
  const { treeData, expandedKeys } = options

  const treeRef = ref()
  const groupedTreeData = computed(() => treeData.value)
  const searchKeyword = ref('')
  const visibleNodeIdsForFilter = computed(() =>
    collectVisibleNodeIds(groupedTreeData.value, searchKeyword.value)
  )
  const defaultExpandedKeysWithWorkspace = computed(() => expandedKeys.value || [])
  const expandedKeysState = ref<number[]>([])
  const treeKey = ref(0)

  const filterNodeMethod = (value: string, data: ServiceTree) => {
    if (!value || !value.trim()) {
      return true
    }
    return visibleNodeIdsForFilter.value.has(Number(data.id))
  }

  watch(searchKeyword, () => {
    nextTick(() => {
      treeRef.value?.filter(searchKeyword.value)
    })
  })

  watch(expandedKeys, async (newKeys, oldKeys) => {
    if (newKeys && Array.isArray(newKeys) && newKeys.length > 0) {
      const keysArray = [...newKeys]
      const oldKeysArray = oldKeys && Array.isArray(oldKeys) ? [...oldKeys] : []
      const keysChanged = JSON.stringify([...keysArray].sort()) !== JSON.stringify([...oldKeysArray].sort())

      if (keysChanged) {
        expandedKeysState.value = keysArray
        treeKey.value++
        await nextTick()
        await new Promise(resolve => setTimeout(resolve, 200))
        expandedKeysState.value = keysArray

        if (treeRef.value && groupedTreeData.value.length > 0) {
          try {
            for (const nodeId of keysArray) {
              const path = findPathToNode(groupedTreeData.value, nodeId)
              if (path.length > 0) {
                await expandPathOnly(treeRef.value, path)
              }
            }
          } catch (error) {
            Logger.error('[ServiceTreePanel]', 'expandPathOnly 展开失败', { error })
          }
        }
      } else {
        expandedKeysState.value = keysArray
      }
      return
    }

    expandedKeysState.value = []
  }, { immediate: true })

  watch(() => groupedTreeData.value.length, (newLength, oldLength) => {
    if (oldLength === 0 && newLength > 0 && expandedKeys.value?.length) {
      expandedKeysState.value = [...expandedKeys.value]
    }
  })

  const expandPaths = async (paths: string[]) => {
    if (!treeRef.value || !groupedTreeData.value.length) {
      return
    }

    for (const path of paths) {
      const node = findNodeByPath(groupedTreeData.value, path)
      if (!node) {
        continue
      }

      const nodeId = Number(node.id)
      const pathToNode = findPathToNode(groupedTreeData.value, nodeId)
      if (pathToNode.length > 0) {
        await expandPathAndSelect(
          treeRef.value,
          groupedTreeData.value,
          pathToNode,
          nodeId
        )
      }
    }
  }

  return {
    treeRef,
    groupedTreeData,
    searchKeyword,
    filterNodeMethod,
    defaultExpandedKeysWithWorkspace,
    expandedKeysState,
    treeKey,
    expandPaths
  }
}
