import axios, {
  AxiosError,
  type AxiosRequestConfig,
  type AxiosResponse,
} from 'axios'
import { NETWORK_CONFIG } from '@/config'
import {
  AppError,
  ErrorCode,
  type ApiErrorResponse,
  type ApiSuccessResponse,
} from '@/types/api'
import {
  createIdempotencyKey,
  createRequestId,
} from '@/utils/id'

type AuthenticationFailureListener = (error: AppError) => void

let csrfToken: string | null = null
const authenticationFailureListeners =
  new Set<AuthenticationFailureListener>()

function captureCsrfToken(value: unknown): void {
  if (
    typeof value === 'object' &&
    value !== null &&
    'csrfToken' in value &&
    typeof value.csrfToken === 'string' &&
    value.csrfToken
  ) {
    csrfToken = value.csrfToken
  }
}

function isManagementMutation(
  method: string | undefined,
  url: string | undefined,
): boolean {
  if (!method || !url) return false
  const mutating = ['post', 'put', 'patch', 'delete'].includes(
    method.toLowerCase(),
  )
  return (
    mutating &&
    (url.startsWith('/manage/') || url.includes('/api/manage/')) &&
    !url.endsWith('/manage/auth/login')
  )
}

function isSuccessResponse<T>(value: unknown): value is ApiSuccessResponse<T> {
  return (
    typeof value === 'object' &&
    value !== null &&
    'code' in value &&
    value.code === 0 &&
    'data' in value
  )
}

function toAppError(value: unknown, status?: number): AppError {
  if (
    typeof value === 'object' &&
    value !== null &&
    'code' in value &&
    typeof value.code === 'number' &&
    value.code !== 0
  ) {
    const response = value as ApiErrorResponse
    return new AppError({
      code: response.code,
      message:
        typeof response.message === 'string' && response.message
          ? response.message
          : '请求未能完成',
      details: response.details,
      status,
    })
  }

  return new AppError({
    code: ErrorCode.Unknown,
    message: '服务暂时不可用，请稍后重试',
    details: value,
    status,
  })
}

function notifyAuthenticationFailure(error: AppError): void {
  if (
    error.code !== ErrorCode.AuthRequired &&
    error.code !== ErrorCode.AccountDisabled
  ) {
    return
  }
  authenticationFailureListeners.forEach((listener) => listener(error))
}

function rejectAppError(value: unknown, status?: number): never {
  const error = toAppError(value, status)
  notifyAuthenticationFailure(error)
  throw error
}

export const httpClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: NETWORK_CONFIG.requestTimeoutMs,
  withCredentials: true,
  headers: {
    Accept: 'application/json',
  },
})

httpClient.interceptors.request.use((config) => {
  if (!config.headers.has('X-Request-Id')) {
    config.headers.set('X-Request-Id', createRequestId())
  }
  if (
    csrfToken &&
    isManagementMutation(config.method, config.url) &&
    !config.headers.has('X-CSRF-Token')
  ) {
    config.headers.set('X-CSRF-Token', csrfToken)
  }
  return config
})

httpClient.interceptors.response.use(
  (response: AxiosResponse<unknown>): AxiosResponse<unknown> => {
    if (isSuccessResponse(response.data)) {
      captureCsrfToken(response.data.data)
      return response.data.data as AxiosResponse<unknown>
    }
    return rejectAppError(response.data, response.status)
  },
  (error: AxiosError<unknown>) => {
    if (error.code === 'ERR_CANCELED') {
      return Promise.reject(error)
    }
    const appError = toAppError(
      error.response?.data ?? error.message,
      error.response?.status,
    )
    notifyAuthenticationFailure(appError)
    return Promise.reject(appError)
  },
)

export function apiRequest<T>(config: AxiosRequestConfig): Promise<T> {
  return httpClient.request(config) as unknown as Promise<T>
}

export function mutationHeaders(scope: string): Record<string, string> {
  return {
    'Idempotency-Key': createIdempotencyKey(scope),
  }
}

export function clearCsrfToken(): void {
  csrfToken = null
}

export function getCsrfToken(): string | null {
  return csrfToken
}

export function onAuthenticationFailure(
  listener: AuthenticationFailureListener,
): () => void {
  authenticationFailureListeners.add(listener)
  return () => authenticationFailureListeners.delete(listener)
}

export { createIdempotencyKey }
