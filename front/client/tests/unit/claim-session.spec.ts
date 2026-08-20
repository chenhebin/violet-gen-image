import { describe, expect, it } from 'vitest'
import { CLAIM_CONFIG } from '@/config'
import {
  clearPendingClaim,
  isClaimCodeFormatValid,
  maskClaimCode,
  readClaimIdempotencyKey,
  readPendingClaimCode,
  saveClaimIdempotencyKey,
  savePendingClaimCode,
} from '@/services/claim-session'

describe('claim session', () => {
  it('normalizes and keeps pending values only in session storage', () => {
    expect(savePendingClaimCode(' yy-akk9-xd9m-9dsm ')).toBe('YY-AKK9-XD9M-9DSM')
    saveClaimIdempotencyKey('idem-claim-flow')

    expect(readPendingClaimCode()).toBe('YY-AKK9-XD9M-9DSM')
    expect(readClaimIdempotencyKey()).toBe('idem-claim-flow')
    expect(localStorage.getItem(CLAIM_CONFIG.pendingCodeStorageKey)).toBeNull()
  })

  it('resets the idempotency key when the code changes', () => {
    savePendingClaimCode('YY-AAAA-BBBB-CCCC')
    saveClaimIdempotencyKey('idem-first')
    savePendingClaimCode('YY-DDDD-EEEE-FFFF')
    expect(readClaimIdempotencyKey()).toBe('')
  })

  it('validates format, masks UI output, and clears both values', () => {
    expect(isClaimCodeFormatValid('YY-AKK9-XD9M-9DSM')).toBe(true)
    expect(isClaimCodeFormatValid('invalid code')).toBe(false)
    expect(maskClaimCode('YY-AKK9-XD9M-9DSM')).toBe('YY-AK**-****-9DSM')

    savePendingClaimCode('YY-AKK9-XD9M-9DSM')
    saveClaimIdempotencyKey('idem-clear')
    clearPendingClaim()
    expect(readPendingClaimCode()).toBe('')
    expect(readClaimIdempotencyKey()).toBe('')
  })
})
