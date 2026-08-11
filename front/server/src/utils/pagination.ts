import { APP_CONFIG } from '@/config'
import type { PageResult } from '@/types/api'

export function paginate<T>(
  items: T[],
  page: number = 1,
  pageSize: number = APP_CONFIG.defaultPageSize,
): PageResult<T> {
  const safePage = Math.max(1, Math.floor(page))
  const safePageSize = Math.min(
    APP_CONFIG.maxPageSize,
    Math.max(1, Math.floor(pageSize)),
  )
  const start = (safePage - 1) * safePageSize
  const pageItems = items.slice(start, start + safePageSize)

  return {
    items: pageItems,
    page: safePage,
    pageSize: safePageSize,
    total: items.length,
    hasMore: start + pageItems.length < items.length,
  }
}
