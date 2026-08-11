const SENSITIVE_KEY =
  /(api.?key|password|secret|token|authorization|full.?code|signed.?url|preview.?url)/i

export function redactAuditValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(redactAuditValue)
  if (!value || typeof value !== 'object') return value

  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>).map(([key, nested]) => [
      key,
      SENSITIVE_KEY.test(key) ? '[已脱敏]' : redactAuditValue(nested),
    ]),
  )
}

export function formatAuditSnapshot(
  value?: Record<string, unknown>,
): string {
  if (!value) return '无'
  return JSON.stringify(redactAuditValue(value), null, 2)
}

