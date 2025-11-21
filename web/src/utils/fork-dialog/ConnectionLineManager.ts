/**
 * 连接线管理器
 * 负责管理连接线的计算和渲染
 */

import { nextTick, type Ref } from 'vue'
import { MappingManager, type ForkMapping, type MappingColor } from './MappingManager'

export interface ConnectionLine {
  source: string
  target: string
  color: MappingColor
  sourceRect?: DOMRect
  targetRect?: DOMRect
}

export class ConnectionLineManager {
  private lines: ConnectionLine[] = []

  constructor(
    private mappingManager: MappingManager,
    private sourceTreeRef: Ref<HTMLElement | null>,
    private targetTreeRef: Ref<HTMLElement | null>,
    private layoutRef: Ref<HTMLElement | null>
  ) {}

  /**
   * 计算连接线
   */
  calculateLines(): ConnectionLine[] {
    const lines: ConnectionLine[] = []
    const mappings = this.mappingManager.getAllMappings()

    console.log('[ConnectionLineManager] 开始计算连接线，映射数量:', mappings.length)

    if (mappings.length === 0) {
      console.log('[ConnectionLineManager] 没有映射，返回空连接线')
      return []
    }

    // 按颜色索引分组映射
    const mappingsByColor = new Map<number, ForkMapping[]>()
    mappings.forEach(m => {
      const colorIndex = m.colorIndex ?? 0
      if (!mappingsByColor.has(colorIndex)) {
        mappingsByColor.set(colorIndex, [])
      }
      mappingsByColor.get(colorIndex)!.push(m)
    })

    // 为每个颜色组创建连接线
    mappingsByColor.forEach((maps, colorIndex) => {
      if (maps.length === 0 || !maps[0]) return
      const color = this.mappingManager.getMappingColor(maps[0])

      console.log('[ConnectionLineManager] 处理颜色组:', {
        colorIndex,
        mapsCount: maps.length,
        firstMapping: maps[0]
      })

      // 如果是多个映射（目录拖拽），找到共同的源父目录
      if (maps.length > 1) {
        // 对于目录拖拽，找到目录映射（source 是目录路径，不是函数组）
        const directoryMapping = maps.find(m => 
          !this.mappingManager.isFunctionGroupMapping(m.source)
        )
        
        // 如果找不到目录映射，使用第一个映射
        const sourceMapping = directoryMapping || maps[0]
        const sourceDirectoryPath = sourceMapping.source
        console.log('[ConnectionLineManager] 多个映射（目录拖拽），源目录路径:', sourceDirectoryPath)
        
        let sourceElement = this.findNodeElement(
          this.sourceTreeRef.value!,
          sourceDirectoryPath
        )

        // 如果没找到，尝试查找源目录的父目录
        if (!sourceElement) {
          const sourcePathParts = sourceDirectoryPath.split('/').filter(Boolean)
          // 尝试向上查找父目录
          for (let i = sourcePathParts.length - 1; i >= 2; i--) {
            const parentPath = '/' + sourcePathParts.slice(0, i).join('/')
            sourceElement = this.findNodeElement(this.sourceTreeRef.value!, parentPath)
            if (sourceElement) {
              console.log('[ConnectionLineManager] 找到源目录的父目录:', parentPath)
              break
            }
          }
        }

        // 如果还是找不到，尝试使用第一个函数组的源节点
        if (!sourceElement) {
          console.log('[ConnectionLineManager] 未找到源目录元素，尝试使用第一个函数组的源节点')
          const firstFunctionGroupMapping = maps.find(m => 
            this.mappingManager.isFunctionGroupMapping(m.source)
          )
          if (firstFunctionGroupMapping) {
            sourceElement = this.findNodeElement(
              this.sourceTreeRef.value!,
              firstFunctionGroupMapping.source
            )
            if (sourceElement) {
              console.log('[ConnectionLineManager] 使用函数组源节点:', firstFunctionGroupMapping.source)
            }
          }
        }

        if (sourceElement) {
          // 找到目标元素（使用目录映射的 target，即新创建的目录）
          const targetMapping = directoryMapping || maps[0]
          let targetElement = this.findTargetElement(targetMapping)

          // 如果找不到目标目录，尝试查找函数组节点（待 Fork 的虚拟节点）
          if (!targetElement) {
            console.log('[ConnectionLineManager] 未找到目标目录，尝试查找函数组节点')
            const functionGroupMapping = maps.find(m => 
              this.mappingManager.isFunctionGroupMapping(m.source)
            )
            if (functionGroupMapping) {
              targetElement = this.findNodeElementExact(
                this.targetTreeRef.value!,
                functionGroupMapping.source
              )
              if (targetElement) {
                console.log('[ConnectionLineManager] 找到函数组节点作为目标')
              }
            }
          }

          // 如果还是找不到，尝试查找目标目录的父目录
          if (!targetElement && targetMapping) {
            const targetPathParts = targetMapping.target.split('/').filter(Boolean)
            for (let i = targetPathParts.length - 1; i >= 2; i--) {
              const parentPath = '/' + targetPathParts.slice(0, i).join('/')
              targetElement = this.findNodeElementExact(this.targetTreeRef.value!, parentPath)
              if (targetElement) {
                console.log('[ConnectionLineManager] 找到目标目录的父目录作为目标:', parentPath)
                break
              }
            }
          }

          if (targetElement) {
            // 检查元素是否可见
            const sourceRect = sourceElement.getBoundingClientRect()
            const targetRect = targetElement.getBoundingClientRect()
            
            // 验证元素是否可见（宽度和高度大于0，且位置合理）
            // 同时检查元素是否在容器内（避免折叠后元素位置异常）
            const sourceStyle = window.getComputedStyle(sourceElement)
            const targetStyle = window.getComputedStyle(targetElement)
            const isSourceVisible = sourceRect.width > 0 && sourceRect.height > 0 && 
                                   sourceRect.left >= 0 && sourceRect.top >= 0 &&
                                   sourceElement.offsetParent !== null &&
                                   sourceStyle.display !== 'none' &&
                                   sourceStyle.visibility !== 'hidden'
            const isTargetVisible = targetRect.width > 0 && targetRect.height > 0 && 
                                   targetRect.left >= 0 && targetRect.top >= 0 &&
                                   targetElement.offsetParent !== null &&
                                   targetStyle.display !== 'none' &&
                                   targetStyle.visibility !== 'hidden'
            
            // 检查元素是否在容器内（相对于 layoutRef）
            const layoutRect = this.layoutRef.value?.getBoundingClientRect()
            const isSourceInContainer = layoutRect ? 
              (sourceRect.left >= layoutRect.left && sourceRect.top >= layoutRect.top &&
               sourceRect.right <= layoutRect.right && sourceRect.bottom <= layoutRect.bottom) : true
            const isTargetInContainer = layoutRect ?
              (targetRect.left >= layoutRect.left && targetRect.top >= layoutRect.top &&
               targetRect.right <= layoutRect.right && targetRect.bottom <= layoutRect.bottom) : true
            
            if (isSourceVisible && isTargetVisible && isSourceInContainer && isTargetInContainer) {
              console.log('[ConnectionLineManager] 找到源和目标元素，创建连接线')
              lines.push({
                source: sourceDirectoryPath,
                target: targetMapping.target,
                color,
                sourceRect,
                targetRect
              })
            } else {
              console.log('[ConnectionLineManager] 元素不可见或不在容器内，跳过连接线:', {
                sourceVisible: isSourceVisible,
                targetVisible: isTargetVisible,
                sourceInContainer: isSourceInContainer,
                targetInContainer: isTargetInContainer,
                sourceRect,
                targetRect
              })
            }
          } else {
            console.log('[ConnectionLineManager] 未找到目标元素:', targetMapping?.target)
          }
        } else {
          console.log('[ConnectionLineManager] 未找到源元素:', sourceDirectoryPath)
        }
      } else {
        // 单个映射（函数组拖拽）
        const mapping = maps[0]
        console.log('[ConnectionLineManager] 单个映射，查找元素:', {
          source: mapping.source,
          target: mapping.target
        })
        const sourceElement = this.findNodeElement(
          this.sourceTreeRef.value!,
          mapping.source
        )
        const targetElement = this.findTargetElement(mapping)

        console.log('[ConnectionLineManager] 元素查找结果:', {
          sourceElement: !!sourceElement,
          targetElement: !!targetElement
        })

        if (sourceElement && targetElement) {
          const sourceRect = sourceElement.getBoundingClientRect()
          const targetRect = targetElement.getBoundingClientRect()
          
          // 验证元素是否可见（宽度和高度大于0，且位置合理）
          // 同时检查元素是否在容器内（避免折叠后元素位置异常）
          const sourceStyle = window.getComputedStyle(sourceElement)
          const targetStyle = window.getComputedStyle(targetElement)
          const isSourceVisible = sourceRect.width > 0 && sourceRect.height > 0 && 
                                 sourceRect.left >= 0 && sourceRect.top >= 0 &&
                                 sourceElement.offsetParent !== null &&
                                 sourceStyle.display !== 'none' &&
                                 sourceStyle.visibility !== 'hidden'
          const isTargetVisible = targetRect.width > 0 && targetRect.height > 0 && 
                                 targetRect.left >= 0 && targetRect.top >= 0 &&
                                 targetElement.offsetParent !== null &&
                                 targetStyle.display !== 'none' &&
                                 targetStyle.visibility !== 'hidden'
          
          // 检查元素是否在容器内（相对于 layoutRef）
          const layoutRect = this.layoutRef.value?.getBoundingClientRect()
          const isSourceInContainer = layoutRect ? 
            (sourceRect.left >= layoutRect.left && sourceRect.top >= layoutRect.top &&
             sourceRect.right <= layoutRect.right && sourceRect.bottom <= layoutRect.bottom) : true
          const isTargetInContainer = layoutRect ?
            (targetRect.left >= layoutRect.left && targetRect.top >= layoutRect.top &&
             targetRect.right <= layoutRect.right && targetRect.bottom <= layoutRect.bottom) : true
          
          if (isSourceVisible && isTargetVisible && isSourceInContainer && isTargetInContainer) {
            lines.push({
              source: mapping.source,
              target: mapping.target,
              color,
              sourceRect,
              targetRect
            })
            console.log('[ConnectionLineManager] 连接线已添加')
          } else {
            console.log('[ConnectionLineManager] 元素不可见或不在容器内，跳过连接线:', {
              sourceVisible: isSourceVisible,
              targetVisible: isTargetVisible,
              sourceInContainer: isSourceInContainer,
              targetInContainer: isTargetInContainer,
              sourceRect,
              targetRect
            })
          }
        } else {
          console.log('[ConnectionLineManager] 元素查找失败，无法创建连接线')
        }
      }
    })

    this.lines = lines
    console.log('[ConnectionLineManager] 连接线计算完成，总数:', lines.length)
    return lines
  }

  /**
   * 更新连接线（异步，等待 DOM 更新）
   */
  async updateLines(): Promise<void> {
    await nextTick()
    
    // 延迟确保 DOM 完全渲染（特别是新创建的目录节点）
    return new Promise<void>((resolve) => {
      setTimeout(() => {
        this.calculateLines()
        resolve()
      }, 300)
    })
  }

  /**
   * 获取当前连接线
   */
  getLines(): ConnectionLine[] {
    return this.lines
  }

  /**
   * 查找目标元素
   */
  private findTargetElement(mapping: ForkMapping): HTMLElement | null {
    if (!this.targetTreeRef.value) {
      console.log('[ConnectionLineManager] 目标树引用不存在')
      return null
    }

    // 判断是函数组映射还是目录映射
    const isFunctionGroup = this.mappingManager.isFunctionGroupMapping(mapping.source)

    console.log('[ConnectionLineManager] 查找目标元素:', {
      source: mapping.source,
      target: mapping.target,
      isFunctionGroup
    })

    if (isFunctionGroup) {
      // 函数组映射：先查找函数组节点（待 Fork 的虚拟节点）
      let element = this.findNodeElement(this.targetTreeRef.value, mapping.source)
      console.log('[ConnectionLineManager] 函数组节点查找结果:', !!element, mapping.source)
      
      // 如果没找到，查找目标目录
      if (!element) {
        element = this.findNodeElement(this.targetTreeRef.value, mapping.target)
        console.log('[ConnectionLineManager] 目标目录查找结果:', !!element, mapping.target)
      }

      return element
    } else {
      // 目录映射：直接查找目标目录（新创建的目录）
      // 注意：对于目录映射，只查找精确匹配，不要向上查找父目录
      const element = this.findNodeElementExact(this.targetTreeRef.value, mapping.target)
      console.log('[ConnectionLineManager] 目录节点查找结果:', !!element, mapping.target)
      return element
    }
  }

  /**
   * 查找节点元素（精确匹配，不向上查找）
   */
  private findNodeElementExact(container: HTMLElement, identifier: string): HTMLElement | null {
    // 方式1: 使用 data-node-id 属性（转义特殊字符）
    let element = container.querySelector(
      `[data-node-id="${this.escapeSelector(identifier)}"]`
    ) as HTMLElement

    if (element) {
      console.log('[ConnectionLineManager] 找到元素（方式1-转义）:', identifier)
      return element
    }

    // 方式2: 使用 data-node-id 属性（不转义）
    element = container.querySelector(
      `[data-node-id="${identifier}"]`
    ) as HTMLElement

    if (element) {
      console.log('[ConnectionLineManager] 找到元素（方式2-不转义）:', identifier)
      return element
    }

    // 方式3: 查找所有带有 data-node-id 的元素，然后匹配
    const allNodes = container.querySelectorAll('[data-node-id]')
    console.log('[ConnectionLineManager] 查找元素，identifier:', identifier, '总节点数:', allNodes.length)
    
    // 收集所有 node-id 用于调试
    const nodeIds: string[] = []
    for (const node of allNodes) {
      const nodeId = (node as HTMLElement).getAttribute('data-node-id')
      if (nodeId) {
        nodeIds.push(nodeId)
        if (nodeId === identifier) {
          console.log('[ConnectionLineManager] 找到元素（方式3-遍历）:', identifier)
          return node as HTMLElement
        }
      }
    }
    
    // 如果找不到，输出所有 node-id 用于调试
    if (nodeIds.length > 0) {
      console.log('[ConnectionLineManager] 未找到元素，所有 node-id:', nodeIds.slice(0, 20)) // 只显示前20个
    }

    return null
  }

  /**
   * 查找节点元素（优化版本，支持向上查找）
   */
  private findNodeElement(container: HTMLElement, identifier: string): HTMLElement | null {
    // 先尝试精确匹配
    let element = this.findNodeElementExact(container, identifier)
    if (element) return element

    // 如果找不到，尝试查找父目录（向上查找）
    if (identifier.startsWith('/')) {
      const pathParts = identifier.split('/').filter(Boolean)
      
      while (pathParts.length > 2 && !element) {
        pathParts.pop()
        const parentPath = '/' + pathParts.join('/')
        element = this.findNodeElementExact(container, parentPath)
        if (element) break
      }
    }

    // 如果是根路径，查找 root 节点
    if (!element && identifier.split('/').filter(Boolean).length <= 2) {
      element = container.querySelector('[data-node-id="root"]') as HTMLElement
    }

    return element
  }

  /**
   * 转义 CSS 选择器中的特殊字符
   */
  private escapeSelector(selector: string): string {
    return selector.replace(/[!"#$%&'()*+,.\/:;<=>?@[\\\]^`{|}~]/g, '\\$&')
  }

  /**
   * 查找多个路径的共同父路径
   */
  private findCommonParentPath(paths: string[]): string {
    if (paths.length === 0) {
      return ''
    }

    if (paths.length === 1) {
      return paths[0]
    }

    // 找到所有路径的共同前缀
    const pathParts = paths.map(p => p.split('/').filter(Boolean))
    const minLength = Math.min(...pathParts.map(p => p.length))

    let commonParts: string[] = []
    for (let i = 0; i < minLength; i++) {
      const part = pathParts[0][i]
      if (pathParts.every(p => p[i] === part)) {
        commonParts.push(part)
      } else {
        break
      }
    }

    return '/' + commonParts.join('/')
  }

  /**
   * 计算连接线的 SVG 路径
   */
  calculateLinePath(line: ConnectionLine): string {
    if (!line.sourceRect || !line.targetRect || !this.layoutRef.value) {
      return ''
    }

    const layoutRect = this.layoutRef.value.getBoundingClientRect()

    // 计算相对于 layout 的坐标
    const sourceX = line.sourceRect.left + line.sourceRect.width / 2 - layoutRect.left
    const sourceY = line.sourceRect.top + line.sourceRect.height / 2 - layoutRect.top
    const targetX = line.targetRect.left + line.targetRect.width / 2 - layoutRect.left
    const targetY = line.targetRect.top + line.targetRect.height / 2 - layoutRect.top

    // 使用贝塞尔曲线连接
    const midX = (sourceX + targetX) / 2
    const controlPoint1X = sourceX + (targetX - sourceX) * 0.5
    const controlPoint1Y = sourceY
    const controlPoint2X = sourceX + (targetX - sourceX) * 0.5
    const controlPoint2Y = targetY

    return `M ${sourceX} ${sourceY} C ${controlPoint1X} ${controlPoint1Y}, ${controlPoint2X} ${controlPoint2Y}, ${targetX} ${targetY}`
  }
}

