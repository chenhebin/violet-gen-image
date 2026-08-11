import { authHandlers } from '@/mocks/handlers/auth'
import { contentHandlers } from '@/mocks/handlers/content'
import { dashboardHandlers } from '@/mocks/handlers/dashboard'
import { providerHandlers } from '@/mocks/handlers/providers'
import { redemptionHandlers } from '@/mocks/handlers/redemption'
import { retouchHandlers } from '@/mocks/handlers/retouch'
import { userHandlers } from '@/mocks/handlers/users'

export const handlers = [
  ...authHandlers,
  ...dashboardHandlers,
  ...redemptionHandlers,
  ...providerHandlers,
  ...userHandlers,
  ...contentHandlers,
  ...retouchHandlers,
]

