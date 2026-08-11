import { apiRequest } from '@/services/http'
import type { PageResult } from '@/types/api'
import type { AuditEvent, AuditQuery } from '@/types/domain'

export const auditApi = {
  list(query: AuditQuery = {}, signal?: AbortSignal) {
    return apiRequest<PageResult<AuditEvent>>({
      method: 'GET',
      url: '/manage/audit-logs',
      params: query,
      signal,
    })
  },
}

