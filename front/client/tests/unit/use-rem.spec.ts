import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  calculateRootFontSize,
  initRem,
} from '@/composables/useRem'

describe('useRem', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    document.documentElement.style.removeProperty('font-size')
  })

  it.each([
    [360, 96],
    [375, 100],
    [768, 100],
    [1024, 85.18519],
    [1200, 75],
    [1440, 75],
    [1920, 100],
    [2560, 100],
  ])('calculates the root font size at %ipx', (width, expected) => {
    expect(calculateRootFontSize(width)).toBeCloseTo(expected, 5)
  })

  it('updates immediately, merges events into one frame, and cleans up', () => {
    const queuedFrames: FrameRequestCallback[] = []
    const addEventListener = vi.spyOn(window, 'addEventListener')
    const removeEventListener = vi.spyOn(window, 'removeEventListener')
    const requestAnimationFrame = vi
      .spyOn(window, 'requestAnimationFrame')
      .mockImplementation((callback) => {
        queuedFrames.push(callback)
        return queuedFrames.length
      })
    const cancelAnimationFrame = vi.spyOn(window, 'cancelAnimationFrame')

    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 360,
      writable: true,
    })

    const cleanup = initRem(window)
    expect(document.documentElement.style.fontSize).toBe('96px')
    expect(addEventListener).toHaveBeenCalledWith(
      'resize',
      expect.any(Function),
      { passive: true },
    )
    expect(addEventListener).toHaveBeenCalledWith(
      'orientationchange',
      expect.any(Function),
      { passive: true },
    )

    window.innerWidth = 1024
    window.dispatchEvent(new Event('resize'))
    window.dispatchEvent(new Event('resize'))
    window.dispatchEvent(new Event('orientationchange'))
    expect(requestAnimationFrame).toHaveBeenCalledTimes(1)

    queuedFrames[0]?.(0)
    expect(Number.parseFloat(document.documentElement.style.fontSize)).toBeCloseTo(
      85.18519,
      5,
    )

    window.dispatchEvent(new Event('resize'))
    cleanup()
    expect(removeEventListener).toHaveBeenCalledWith(
      'resize',
      expect.any(Function),
    )
    expect(removeEventListener).toHaveBeenCalledWith(
      'orientationchange',
      expect.any(Function),
    )
    expect(cancelAnimationFrame).toHaveBeenCalledWith(2)
  })
})
