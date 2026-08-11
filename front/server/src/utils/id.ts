export function createId(prefix: string): string {
  const randomPart =
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID().replaceAll('-', '').slice(0, 16)
      : Math.random().toString(36).slice(2, 18)

  return `${prefix}_${Date.now().toString(36)}_${randomPart}`
}

export function createRequestId(): string {
  return createId('req')
}

export function createIdempotencyKey(scope: string): string {
  return createId(`idem_${scope}`)
}

