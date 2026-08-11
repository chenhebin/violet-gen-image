import { http } from 'msw'
import {
  clearSession,
  getSessionAdminId,
  publicAdmin,
  readDb,
  resetDb,
  setSessionAdminId,
} from '@/mocks/db'
import {
  dbAndAdmin,
  MockApiError,
  readJson,
  respond,
} from '@/mocks/helpers'
import { ErrorCode } from '@/types/api'
import type { AdminLoginPayload } from '@/types/domain'

export const authHandlers = [
  http.post('/api/manage/auth/login', ({ request }) =>
    respond(async () => {
      const payload = await readJson<AdminLoginPayload>(request)
      const db = readDb()
      const email = payload.email.trim().toLowerCase()
      const admin = db.admins.find(
        (item) => item.email.toLowerCase() === email,
      )
      if (!admin || admin.password !== payload.password) {
        throw new MockApiError(
          401,
          ErrorCode.AuthRequired,
          '邮箱或密码不正确',
        )
      }
      if (admin.status !== 'active') {
        throw new MockApiError(
          403,
          ErrorCode.AccountDisabled,
          '管理员账号已停用',
        )
      }
      setSessionAdminId(admin.id, Boolean(payload.remember))
      return publicAdmin(admin)
    }),
  ),

  http.get('/api/manage/auth/session', () =>
    respond(() => {
      const db = readDb()
      const admin = db.admins.find(
        (item) => item.id === getSessionAdminId(),
      )
      if (!admin) {
        throw new MockApiError(
          401,
          ErrorCode.AuthRequired,
          '登录状态已失效',
        )
      }
      if (admin.status !== 'active') {
        clearSession()
        throw new MockApiError(
          403,
          ErrorCode.AccountDisabled,
          '管理员账号已停用',
        )
      }
      return publicAdmin(admin)
    }),
  ),

  http.post('/api/manage/auth/logout', () =>
    respond(() => {
      clearSession()
      return null
    }),
  ),

  http.post('/api/manage/mock/reset', () =>
    respond(() => {
      dbAndAdmin('platform:manage')
      resetDb()
      return null
    }),
  ),
]
