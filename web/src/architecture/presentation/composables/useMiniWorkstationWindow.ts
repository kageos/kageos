import { computed, ref, type Ref } from 'vue'

type ResizeDir = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw'

export interface UseMiniWorkstationWindowOptions {
  maximized: Ref<boolean>
  initialOffset?: number
  initialPosition?: 'center'
}

const MIN_W = 220
const MIN_H = 120
const DEFAULT_W = 320
const DEFAULT_H = 220

export function useMiniWorkstationWindow(options: UseMiniWorkstationWindowOptions) {
  const { maximized, initialOffset, initialPosition } = options

  const rootRef = ref<HTMLElement>()
  const posX = ref<number | null>(null)
  const posY = ref<number | null>(null)
  const winW = ref(DEFAULT_W)
  const winH = ref(DEFAULT_H)

  let dragStartX = 0
  let dragStartY = 0
  let dragOriginX = 0
  let dragOriginY = 0
  let dragging = false

  let resizeDir: ResizeDir = 's'
  let resizeStartX = 0
  let resizeStartY = 0
  let resizeOriginX = 0
  let resizeOriginY = 0
  let resizeOriginW = 0
  let resizeOriginH = 0
  let resizeAspect = 1
  let resizing = false

  const windowStyle = computed(() => {
    if (maximized.value) {
      return {
        left: '0',
        top: '0',
        right: '0',
        bottom: '0',
        width: '100vw',
        height: '100vh',
        borderRadius: '0',
        transform: 'none'
      }
    }

    const base: Record<string, string> = {
      '--mini-ws-base-width': `${winW.value}px`,
      '--mini-ws-base-height': `${winH.value}px`,
      width: `${winW.value}px`,
      height: `${winH.value}px`
    }

    if (posX.value !== null && posY.value !== null) {
      return { ...base, left: `${posX.value}px`, top: `${posY.value}px`, right: 'auto', bottom: 'auto' }
    }

    if (initialPosition === 'center') {
      return { ...base, left: '50%', top: '50%', transform: 'translate(-50%, -50%)', right: 'auto', bottom: 'auto' }
    }

    const offset = initialOffset || 0
    if (offset > 0) {
      return { ...base, right: `${24 + offset}px`, bottom: `${80 + offset}px` }
    }

    return base
  })

  function startDrag(event: MouseEvent) {
    if (maximized.value) {
      return
    }

    dragging = true
    dragStartX = event.clientX
    dragStartY = event.clientY
    const element = (event.currentTarget as HTMLElement).parentElement!
    const rect = element.getBoundingClientRect()
    dragOriginX = rect.left
    dragOriginY = rect.top
    document.addEventListener('mousemove', onDrag)
    document.addEventListener('mouseup', stopDrag)
  }

  function onDrag(event: MouseEvent) {
    if (!dragging) {
      return
    }
    posX.value = dragOriginX + (event.clientX - dragStartX)
    posY.value = dragOriginY + (event.clientY - dragStartY)
  }

  function stopDrag() {
    dragging = false
    document.removeEventListener('mousemove', onDrag)
    document.removeEventListener('mouseup', stopDrag)
  }

  function startResize(event: MouseEvent, dir: ResizeDir) {
    event.preventDefault()
    resizing = true
    resizeDir = dir
    resizeStartX = event.clientX
    resizeStartY = event.clientY
    const element = (event.target as HTMLElement).closest('.mini-ws') as HTMLElement
    if (element) {
      const rect = element.getBoundingClientRect()
      resizeOriginX = rect.left
      resizeOriginY = rect.top
      resizeOriginW = rect.width
      resizeOriginH = rect.height
      if (posX.value === null) {
        posX.value = rect.left
        posY.value = rect.top
      }
    } else {
      resizeOriginW = winW.value
      resizeOriginH = winH.value
    }
    resizeAspect = resizeOriginW / resizeOriginH
    document.addEventListener('mousemove', onResize)
    document.addEventListener('mouseup', stopResize)
  }

  function onResize(event: MouseEvent) {
    if (!resizing) {
      return
    }

    const dx = event.clientX - resizeStartX
    const dy = event.clientY - resizeStartY
    const isEdge = resizeDir.length === 1

    if (isEdge) {
      if (resizeDir === 'e' || resizeDir === 'w') {
        const rawW = resizeDir === 'e' ? resizeOriginW + dx : resizeOriginW - dx
        const newW = Math.max(MIN_W, rawW)
        const newH = Math.max(MIN_H, Math.round(newW / resizeAspect))
        winW.value = newW
        winH.value = newH
        if (resizeDir === 'w') {
          posX.value = resizeOriginX + (resizeOriginW - newW)
        }
      } else {
        const rawH = resizeDir === 's' ? resizeOriginH + dy : resizeOriginH - dy
        const newH = Math.max(MIN_H, rawH)
        const newW = Math.max(MIN_W, Math.round(newH * resizeAspect))
        winW.value = newW
        winH.value = newH
        if (resizeDir === 'n') {
          posY.value = resizeOriginY + (resizeOriginH - newH)
        }
      }
      return
    }

    if (resizeDir.includes('e')) {
      winW.value = Math.max(MIN_W, resizeOriginW + dx)
    }
    if (resizeDir.includes('w')) {
      const newW = Math.max(MIN_W, resizeOriginW - dx)
      posX.value = resizeOriginX + (resizeOriginW - newW)
      winW.value = newW
    }
    if (resizeDir.includes('s')) {
      winH.value = Math.max(MIN_H, resizeOriginH + dy)
    }
    if (resizeDir.includes('n')) {
      const newH = Math.max(MIN_H, resizeOriginH - dy)
      posY.value = resizeOriginY + (resizeOriginH - newH)
      winH.value = newH
    }
  }

  function stopResize() {
    resizing = false
    document.removeEventListener('mousemove', onResize)
    document.removeEventListener('mouseup', stopResize)
  }

  function captureWindowRect() {
    const element = rootRef.value
    if (element) {
      const rect = element.getBoundingClientRect()
      return { x: rect.left, y: rect.top, w: rect.width, h: rect.height }
    }
    return { x: posX.value ?? 0, y: posY.value ?? 0, w: winW.value, h: winH.value }
  }

  function restoreWindowRect(rect: { x: number; y: number; w: number; h: number } | null) {
    if (!rect) {
      return
    }
    posX.value = rect.x
    posY.value = rect.y
    winW.value = rect.w
    winH.value = rect.h
  }

  function dispose() {
    stopDrag()
    stopResize()
  }

  return {
    rootRef,
    windowStyle,
    startDrag,
    startResize,
    captureWindowRect,
    restoreWindowRect,
    dispose
  }
}
