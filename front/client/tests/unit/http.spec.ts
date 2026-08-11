import type { AxiosRequestConfig, AxiosResponse } from 'axios'
import { describe, expect, it } from 'vitest'
import {
  apiRequest,
  httpClient,
  onAuthenticationFailure,
} from '@/services/http'
import { AppError, ErrorCode } from '@/types/api'

function response(
  config: AxiosRequestConfig,
  data: unknown,
  status = 200,
): AxiosResponse {
  return {
    data,
    status,
    statusText: status < 400 ? 'OK' : 'Error',
    headers: {},
    config: config as AxiosResponse['config'],
  }
}

describe('http client', () => {
  it('uses the shared /api base URL and credentials', () => {
    expect(httpClient.defaults.baseURL).toBe('/api')
    expect(httpClient.defaults.withCredentials).toBe(true)
  })

  it('adds a request ID and unwraps code 0 data', async () => {
    let requestId = ''
    const result = await apiRequest<{ value: number }>({
      method: 'GET',
      url: '/probe',
      adapter: async (config) => {
        requestId = String(config.headers.get('X-Request-Id'))
        return response(config, { code: 0, data: { value: 42 } })
      },
    })

    expect(result).toEqual({ value: 42 })
    expect(requestId).toMatch(/^req_/)
  })

  it('converts an HTTP 200 business error into AppError', async () => {
    const request = apiRequest({
      method: 'GET',
      url: '/business-error',
      adapter: async (config) =>
        response(config, {
          code: ErrorCode.InsufficientCredits,
          message: '剩余次数不足',
          details: { balance: 0 },
        }),
    })

    await expect(request).rejects.toMatchObject({
      name: 'AppError',
      code: ErrorCode.InsufficientCredits,
      message: '剩余次数不足',
      details: { balance: 0 },
    })
  })

  it('uses a clear fallback for malformed or message-less errors', async () => {
    const missingMessage = apiRequest({
      method: 'GET',
      url: '/missing-message',
      adapter: async (config) => response(config, { code: 6001 }),
    })
    await expect(missingMessage).rejects.toMatchObject({
      code: 6001,
      message: '请求未能完成',
    })

    const malformed = apiRequest({
      method: 'GET',
      url: '/malformed',
      adapter: async (config) => response(config, { unexpected: true }, 500),
    })
    await expect(malformed).rejects.toMatchObject({
      code: ErrorCode.Unknown,
      message: '服务暂时不可用，请稍后重试',
    })
  })

  it('preserves Axios cancellation semantics', async () => {
    const controller = new AbortController()
    controller.abort()

    try {
      await apiRequest({
        method: 'GET',
        url: '/cancelled',
        signal: controller.signal,
        adapter: async (config) =>
          response(config, { code: 0, data: { unreachable: true } }),
      })
      throw new Error('Expected request to be cancelled')
    } catch (caught) {
      expect(caught).not.toBeInstanceOf(AppError)
      expect(caught).toMatchObject({ code: 'ERR_CANCELED' })
    }
  })

  it('notifies the application when an authenticated session is invalid', async () => {
    const codes: number[] = []
    const unsubscribe = onAuthenticationFailure((error) => {
      codes.push(error.code)
    })
    try {
      await expect(
        apiRequest({
          method: 'GET',
          url: '/expired-session',
          adapter: async (config) =>
            response(config, {
              code: ErrorCode.AuthRequired,
              message: '登录状态已失效',
            }),
        }),
      ).rejects.toMatchObject({ code: ErrorCode.AuthRequired })
      expect(codes).toEqual([ErrorCode.AuthRequired])
    } finally {
      unsubscribe()
    }
  })
})
