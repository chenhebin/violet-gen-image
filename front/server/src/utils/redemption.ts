import { REDEMPTION_CONFIG } from '@/config'
import type { RedemptionCodeStatus } from '@/types/domain'

export interface RedemptionStateSource {
  redeemedAt?: string
  disabledAt?: string
  expiresAt: string | null
}

export function deriveRedemptionStatus(
  code: RedemptionStateSource,
  now = Date.now(),
): RedemptionCodeStatus {
  if (code.redeemedAt) return 'redeemed'
  if (code.disabledAt) return 'disabled'
  if (code.expiresAt && new Date(code.expiresAt).getTime() <= now) {
    return 'expired'
  }
  return 'unused'
}

export function isExpiringSoon(
  code: RedemptionStateSource,
  now = Date.now(),
): boolean {
  if (deriveRedemptionStatus(code, now) !== 'unused' || !code.expiresAt) {
    return false
  }
  const remaining = new Date(code.expiresAt).getTime() - now
  return remaining <= REDEMPTION_CONFIG.expiringSoonDays * 86_400_000
}

export function maskRedemptionCode(code: string): string {
  const parts = code.split('-')
  return parts.length >= 4
    ? `${parts[0]}-****-****-${parts.at(-1)}`
    : `****${code.slice(-4)}`
}

export function normalizeRedemptionCode(code: string): string {
  return code.trim().toUpperCase()
}

