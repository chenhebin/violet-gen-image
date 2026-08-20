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

export const ErrorCode = {
  AuthRequired: 1001,
  AccountDisabled: 1002,
  RedemptionRequired: 2001,
  InsufficientCredits: 2002,
  AINoticeRequired: 2003,
  CodeInvalid: 3001,
  CodeUsed: 3002,
  CodeExpired: 3003,
  ProductMismatch: 3004,
  RateLimited: 4001,
  DuplicateRequest: 4002,
  TaskFailedRefunded: 5001,
  InvalidPayload: 6001,
  AssetNotFound: 6002,
  RetouchTaskNotEligible: 7001,
  RetouchTicketAlreadyExists: 7002,
  RetouchTicketNotFound: 7003,
  RetouchInvalidStatus: 7004,
  RetouchQuoteInvalid: 7005,
  RetouchRevisionLimitReached: 7006,
  Unknown: 9999,
} as const

export class AppError extends Error {
  readonly code: number
  readonly details?: unknown
  readonly retryAfterSeconds?: number

  constructor(response: ApiErrorResponse, retryAfterSeconds?: number) {
    super(response.message)
    this.name = 'AppError'
    this.code = response.code
    this.details = response.details
    this.retryAfterSeconds = retryAfterSeconds
  }
}
