import axios, {
  AxiosError,
  type AxiosRequestConfig,
  type AxiosResponse,
} from 'axios'
import { NETWORK_CONFIG } from '@/config'
import type { ApiErrorResponse, ApiSuccessResponse } from '@/types/api'
import { AppError, ErrorCode } from '@/types/api'
import { createId } from '@/utils/id'

type AuthenticationFailureListener = (error: AppError) => void

const authenticationFailureListeners =
  new Set<AuthenticationFailureListener>()

function isSuccessResponse<T>(value: unknown): value is ApiSuccessResponse<T> {
  return (
    typeof value === 'object' &&
    value !== null &&
    'code' in value &&
    value.code === 0 &&
    'data' in value
  )
}

function toAppError(value: unknown): AppError {
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
    })
  }

  return new AppError({
    code: ErrorCode.Unknown,
    message: '服务暂时不可用，请稍后重试',
    details: value,
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

function rejectAppError(value: unknown): never {
  const error = toAppError(value)
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
  config.headers.set('X-Request-Id', createId('req'))
  return config
})

httpClient.interceptors.response.use(
  (response: AxiosResponse<unknown>): AxiosResponse<unknown> => {
    if (isSuccessResponse(response.data)) {
      return response.data.data as AxiosResponse<unknown>
    }

    return rejectAppError(response.data)
  },
  (error: AxiosError<unknown>) => {
    if (error.code === 'ERR_CANCELED') {
      return Promise.reject(error)
    }

    const retryAfter = Number(error.response?.headers?.['retry-after'])
    const appError = toAppError(error.response?.data ?? error.message)
    if (Number.isFinite(retryAfter) && retryAfter > 0) {
      Object.defineProperty(appError, 'retryAfterSeconds', { value: retryAfter })
    }
    notifyAuthenticationFailure(appError)
    return Promise.reject(appError)
  },
)

export function apiRequest<T>(config: AxiosRequestConfig): Promise<T> {
  return httpClient.request<ApiSuccessResponse<T>, T>(config)
}

export function createIdempotencyKey(scope: string): string {
  return createId(`idem_${scope}`)
}

export function onAuthenticationFailure(
  listener: AuthenticationFailureListener,
): () => void {
  authenticationFailureListeners.add(listener)
  return () => authenticationFailureListeners.delete(listener)
}
