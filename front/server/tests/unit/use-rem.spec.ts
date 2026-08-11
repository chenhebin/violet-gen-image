import { describe, expect, it, vi } from 'vitest'
import {
  calculateRootFontSize,
  initRem,
} from '@/composables/useRem'

describe('calculateRootFontSize', () => {
  it.each([
    [360, 96],
    [375, 100],
    [768, 100],
    [1024, 85.185185],
    [1200, 75],
    [1440, 75],
    [1920, 100],
    [2560, 100],
  ])('calculates the root size at %ipx', (width, expected) => {
    expect(calculateRootFontSize(width)).toBeCloseTo(expected, 5)
  })

  it('does not return a negative size for invalid widths', () => {
    expect(calculateRootFontSize(-100)).toBe(0)
  })
})

describe('initRem', () => {
  it('applies immediately, merges resize frames, and cleans up listeners', () => {
    const callbacks: FrameRequestCallback[] = []
    const requestFrame = vi
      .spyOn(window, 'requestAnimationFrame')
      .mockImplementation((callback) => {
        callbacks.push(callback)
        return callbacks.length
      })
    const cancelFrame = vi.spyOn(window, 'cancelAnimationFrame')
    const addListener = vi.spyOn(window, 'addEventListener')
    const removeListener = vi.spyOn(window, 'removeEventListener')

    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 1440,
      writable: true,
    })

    const cleanup = initRem(window)
    expect(document.documentElement.style.fontSize).toBe('75px')
    expect(addListener).toHaveBeenCalledWith(
      'resize',
      expect.any(Function),
      { passive: true },
    )
    expect(addListener).toHaveBeenCalledWith(
      'orientationchange',
      expect.any(Function),
      { passive: true },
    )

    window.dispatchEvent(new Event('resize'))
    window.dispatchEvent(new Event('resize'))
    expect(requestFrame).toHaveBeenCalledTimes(1)

    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 375,
      writable: true,
    })
    callbacks[0]?.(0)
    expect(document.documentElement.style.fontSize).toBe('100px')

    window.dispatchEvent(new Event('orientationchange'))
    cleanup()
    expect(removeListener).toHaveBeenCalledWith('resize', expect.any(Function))
    expect(removeListener).toHaveBeenCalledWith(
      'orientationchange',
      expect.any(Function),
    )
    expect(cancelFrame).toHaveBeenCalledTimes(1)
  })
})
