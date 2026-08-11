import { AppError, ErrorCode } from '@/types/api'

export function normalizeStoreError(error: unknown): AppError {
  if (error instanceof AppError) return error
  if (error instanceof Error) {
    return new AppError({
      code: ErrorCode.Unknown,
      message: error.message,
    })
  }
  return new AppError({
    code: ErrorCode.Unknown,
    message: '操作未能完成',
    details: error,
  })
}

