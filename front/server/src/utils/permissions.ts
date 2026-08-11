import type {
  AdminPermission,
  AdminSession,
  AdminRole,
} from '@/types/domain'

export function hasPermission(
  session: AdminSession | null,
  permission: AdminPermission,
): boolean {
  return Boolean(
    session?.status === 'active' && session.permissions.includes(permission),
  )
}

export function permissionsForRole(role: AdminRole): AdminPermission[] {
  return role === 'platform_admin'
    ? ['platform:manage', 'retouch:manage']
    : ['retouch:manage']
}

