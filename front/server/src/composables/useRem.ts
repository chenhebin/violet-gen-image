import { onBeforeUnmount } from 'vue'
import { REM_CONFIG } from '@/config'

export function calculateRootFontSize(width: number): number {
  const safeWidth = Math.max(0, width)

  if (safeWidth <= REM_CONFIG.mobileScaleMaxWidth) {
    return (
      (REM_CONFIG.rootValue * safeWidth) / REM_CONFIG.mobileDesignWidth
    )
  }

  if (safeWidth <= REM_CONFIG.tabletFixedMaxWidth) {
    return REM_CONFIG.rootValue
  }

  if (safeWidth < REM_CONFIG.compactDesktopMinWidth) {
    const progress =
      (safeWidth - REM_CONFIG.tabletFixedMaxWidth) /
      (REM_CONFIG.compactDesktopMinWidth - REM_CONFIG.tabletFixedMaxWidth)
    return (
      REM_CONFIG.rootValue -
      progress *
        (REM_CONFIG.rootValue - REM_CONFIG.compactRootFontSize)
    )
  }

  if (safeWidth <= REM_CONFIG.compactDesktopMaxWidth) {
    return REM_CONFIG.compactRootFontSize
  }

  if (safeWidth < REM_CONFIG.desktopDesignWidth) {
    return (
      (REM_CONFIG.rootValue * safeWidth) / REM_CONFIG.desktopDesignWidth
    )
  }

  return REM_CONFIG.maxRootFontSize
}

export function initRem(targetWindow: Window = window): () => void {
  let frameId: number | null = null

  const applyRootFontSize = (): void => {
    frameId = null
    const fontSize = calculateRootFontSize(targetWindow.innerWidth)
    targetWindow.document.documentElement.style.fontSize = `${fontSize}px`
  }

  const scheduleUpdate = (): void => {
    if (frameId !== null) return
    frameId = targetWindow.requestAnimationFrame(applyRootFontSize)
  }

  applyRootFontSize()
  targetWindow.addEventListener('resize', scheduleUpdate, { passive: true })
  targetWindow.addEventListener('orientationchange', scheduleUpdate, {
    passive: true,
  })

  return () => {
    targetWindow.removeEventListener('resize', scheduleUpdate)
    targetWindow.removeEventListener('orientationchange', scheduleUpdate)
    if (frameId !== null) targetWindow.cancelAnimationFrame(frameId)
    frameId = null
  }
}

export function useRem(): void {
  const cleanup = initRem()
  onBeforeUnmount(cleanup)
}
