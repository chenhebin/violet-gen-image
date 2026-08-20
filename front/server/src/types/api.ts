export interface ApiSuccessResponse<T> {
  code: 0
  data: T
}

export interface ApiErrorResponse {
  code: number
  message: string
  details?: unknown
}

export interface PageResult<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
  hasMore: boolean
}

export interface PageQuery {
  page?: number
  pageSize?: number
  keyword?: string
}

export const ErrorCode = {
  AuthRequired: 1001,
  AccountDisabled: 1002,
  Forbidden: 1003,
  InsufficientCredits: 2002,
  CodeInvalid: 3001,
  RateLimited: 4001,
  DuplicateRequest: 4002,
  InvalidPayload: 6001,
  NotFound: 6004,
  RetouchNotFound: 7003,
  RetouchInvalidStatus: 7004,
  RetouchQuoteInvalid: 7005,
  Unknown: 9999,
} as const

export class AppError extends Error {
  readonly code: number
  readonly details?: unknown
  readonly status?: number
  readonly retryAfterSeconds?: number

  constructor(input: {
    code: number
    message: string
    details?: unknown
    status?: number
    retryAfterSeconds?: number
  }) {
    super(input.message)
    this.name = 'AppError'
    this.code = input.code
    this.details = input.details
    this.status = input.status
    this.retryAfterSeconds = input.retryAfterSeconds
  }
}
