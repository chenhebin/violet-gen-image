import { HttpResponse, delay, http } from 'msw'
import { setupServer } from 'msw/node'
import {
  afterAll,
  afterEach,
  beforeAll,
  describe,
  expect,
  it,
} from 'vitest'
import {
  apiRequest,
  clearCsrfToken,
  getCsrfToken,
  httpClient,
  onAuthenticationFailure,
} from '@/services/http'
import { AppError, ErrorCode } from '@/types/api'

const originalBaseUrl = httpClient.defaults.baseURL
let capturedRequestId: string | null = null
let capturedCsrfToken: string | null = null

const server = setupServer(
  http.get('http://localhost/api/test/success', ({ request }) => {
    capturedRequestId = request.headers.get('X-Request-Id')
    return HttpResponse.json({ code: 0, data: { value: 42 } })
  }),
  http.get('http://localhost/api/test/business-error', () =>
    HttpResponse.json(
      { code: 6001, message: '参数不正确', details: { field: 'name' } },
      { status: 200 },
    ),
  ),
  http.get('http://localhost/api/test/http-error', () =>
    HttpResponse.json(
      { code: 1003, message: '没有权限' },
      { status: 403 },
    ),
  ),
  http.get('http://localhost/api/test/auth-error', () =>
    HttpResponse.json(
      { code: 1001, message: '登录状态已失效' },
      { status: 401 },
    ),
  ),
  http.get('http://localhost/api/test/malformed', () =>
    HttpResponse.json({ value: 'unexpected' }),
  ),
  http.get('http://localhost/api/test/slow', async () => {
    await delay(200)
    return HttpResponse.json({ code: 0, data: true })
  }),
  http.get('http://localhost/api/manage/auth/session', () =>
    HttpResponse.json({
      code: 0,
      data: {
        id: 'admin_test',
        csrfToken: 'csrf_session_token',
      },
    }),
  ),
  http.post(
    'http://localhost/api/manage/users/user_test/status',
    ({ request }) => {
      capturedCsrfToken = request.headers.get('X-CSRF-Token')
      return HttpResponse.json({ code: 0, data: null })
    },
  ),
)

beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' })
  httpClient.defaults.baseURL = 'http://localhost/api'
})

afterEach(() => {
  server.resetHandlers()
  capturedRequestId = null
  capturedCsrfToken = null
  clearCsrfToken()
})

afterAll(() => {
  server.close()
  httpClient.defaults.baseURL = originalBaseUrl
})

describe('http client', () => {
  it('uses the shared API configuration', () => {
    expect(originalBaseUrl).toBe('/api')
    expect(httpClient.defaults.withCredentials).toBe(true)
    expect(httpClient.defaults.headers.common.Accept).toContain(
      'application/json',
    )
  })

  it('unwraps code 0 data and adds a request id', async () => {
    const result = await apiRequest<{ value: number }>({
      method: 'GET',
      url: '/test/success',
    })

    expect(result).toEqual({ value: 42 })
    expect(capturedRequestId).toMatch(/^req_/)
  })

  it.each([
    ['/test/business-error', 6001, '参数不正确'],
    ['/test/http-error', 1003, '没有权限'],
  ])('normalizes %s as AppError', async (url, code, message) => {
    await expect(
      apiRequest({ method: 'GET', url }),
    ).rejects.toMatchObject({
      name: 'AppError',
      code,
      message,
    })
  })

  it('falls back safely for malformed response data', async () => {
    await expect(
      apiRequest({ method: 'GET', url: '/test/malformed' }),
    ).rejects.toEqual(
      expect.objectContaining<AppError>({
        name: 'AppError',
        code: ErrorCode.Unknown,
        message: '服务暂时不可用，请稍后重试',
      }),
    )
  })

  it('notifies the application when the management session expires', async () => {
    const failures: AppError[] = []
    const unsubscribe = onAuthenticationFailure((error) => {
      failures.push(error)
    })

    await expect(
      apiRequest({ method: 'GET', url: '/test/auth-error' }),
    ).rejects.toMatchObject({ code: ErrorCode.AuthRequired })
    unsubscribe()

    expect(failures).toHaveLength(1)
    expect(failures[0]?.code).toBe(ErrorCode.AuthRequired)
  })

  it('preserves request cancellation', async () => {
    const controller = new AbortController()
    const request = apiRequest({
      method: 'GET',
      url: '/test/slow',
      signal: controller.signal,
    })
    controller.abort()

    await expect(request).rejects.toMatchObject({ code: 'ERR_CANCELED' })
  })

  it('keeps the management csrf token in memory and sends it on writes', async () => {
    await apiRequest({
      method: 'GET',
      url: '/manage/auth/session',
    })

    expect(getCsrfToken()).toBe('csrf_session_token')

    await apiRequest({
      method: 'POST',
      url: '/manage/users/user_test/status',
      data: { status: 'disabled' },
    })

    expect(capturedCsrfToken).toBe('csrf_session_token')
  })
})
