import { CLAIM_CONFIG } from '@/config'

export function normalizeClaimCode(value: string): string {
  return value.trim().toUpperCase()
}

export function isClaimCodeFormatValid(value: string): boolean {
  const normalized = normalizeClaimCode(value)
  return (
    normalized.length <= CLAIM_CONFIG.codeMaxLength &&
    /^[A-Z0-9]+(?:-[A-Z0-9]+)+$/.test(normalized)
  )
}

export function maskClaimCode(value: string): string {
  const normalized = normalizeClaimCode(value)
  const parts = normalized.split('-')
  if (parts.length >= 3) {
    return parts
      .map((part, index) => {
        if (index === 0 || index === parts.length - 1) return part
        if (index === 1 && part.length > 2) return `${part.slice(0, 2)}${'*'.repeat(part.length - 2)}`
        return '*'.repeat(part.length)
      })
      .join('-')
  }
  return normalized.length > 6
    ? `${normalized.slice(0, 3)}${'*'.repeat(normalized.length - 6)}${normalized.slice(-3)}`
    : '*'.repeat(normalized.length)
}

export function readPendingClaimCode(): string {
  return sessionStorage.getItem(CLAIM_CONFIG.pendingCodeStorageKey) ?? ''
}

export function savePendingClaimCode(value: string): string {
  const normalized = normalizeClaimCode(value)
  const previous = readPendingClaimCode()
  sessionStorage.setItem(CLAIM_CONFIG.pendingCodeStorageKey, normalized)
  if (previous !== normalized) {
    sessionStorage.removeItem(CLAIM_CONFIG.idempotencyStorageKey)
  }
  return normalized
}

export function clearPendingClaim(): void {
  sessionStorage.removeItem(CLAIM_CONFIG.pendingCodeStorageKey)
  sessionStorage.removeItem(CLAIM_CONFIG.idempotencyStorageKey)
}

export function readClaimIdempotencyKey(): string {
  return sessionStorage.getItem(CLAIM_CONFIG.idempotencyStorageKey) ?? ''
}

export function saveClaimIdempotencyKey(value: string): void {
  sessionStorage.setItem(CLAIM_CONFIG.idempotencyStorageKey, value)
}
