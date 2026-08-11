import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { entitlementApi } from '@/services/api'
import { createIdempotencyKey } from '@/services/http'
import { AppError, ErrorCode } from '@/types/api'
import type {
  Entitlement,
  LedgerEntry,
  RedemptionResult,
  UsageQuote,
} from '@/types/domain'

const redemptionKeys = new Map<string, string>()

export const useEntitlementStore = defineStore('entitlement', () => {
  const entitlement = ref<Entitlement | null>(null)
  const ledger = ref<LedgerEntry[]>([])
  const quote = ref<UsageQuote | null>(null)
  const loading = ref(false)
  const quoting = ref(false)
  const error = ref('')
  let quoteController: AbortController | null = null

  const balance = computed(() => entitlement.value?.balance ?? 0)
  const canCreate = computed(() => entitlement.value?.canCreate ?? false)

  async function load(): Promise<void> {
    loading.value = true
    error.value = ''
    try {
      const [nextEntitlement, nextLedger] = await Promise.all([
        entitlementApi.get(),
        entitlementApi.ledger(),
      ])
      entitlement.value = nextEntitlement
      ledger.value = nextLedger
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : '权益信息加载失败'
      throw caught
    } finally {
      loading.value = false
    }
  }

  async function redeem(code: string): Promise<RedemptionResult> {
    const normalized = code.trim().toUpperCase()
    const key =
      redemptionKeys.get(normalized) ?? createIdempotencyKey('redemption')
    redemptionKeys.set(normalized, key)
    try {
      const result = await entitlementApi.redeem(normalized, key)
      redemptionKeys.delete(normalized)
      entitlement.value = result.entitlement
      await refreshLedger()
      return result
    } catch (caught) {
      if (!(caught instanceof AppError && caught.code === ErrorCode.Unknown)) {
        redemptionKeys.delete(normalized)
      }
      throw caught
    }
  }

  async function refreshLedger(): Promise<void> {
    ledger.value = await entitlementApi.ledger()
  }

  async function requestQuote(outputCount: number): Promise<UsageQuote | null> {
    quoteController?.abort()
    quoteController = new AbortController()
    quoting.value = true
    try {
      const result = await entitlementApi.quote(
        outputCount,
        quoteController.signal,
      )
      quote.value = result
      return result
    } catch (caught) {
      if (caught instanceof DOMException && caught.name === 'AbortError') return null
      if (
        typeof caught === 'object' &&
        caught !== null &&
        'code' in caught &&
        caught.code === 'ERR_CANCELED'
      ) {
        return null
      }
      error.value = caught instanceof Error ? caught.message : '报价获取失败'
      throw caught
    } finally {
      quoting.value = false
    }
  }

  function applyBalance(nextBalance: number): void {
    if (!entitlement.value) return
    entitlement.value = {
      balance: nextBalance,
      canCreate: nextBalance > 0,
      status: nextBalance > 0 ? 'active' : 'empty',
    }
  }

  function reset(): void {
    quoteController?.abort()
    entitlement.value = null
    ledger.value = []
    quote.value = null
    error.value = ''
    redemptionKeys.clear()
  }

  return {
    entitlement,
    ledger,
    quote,
    loading,
    quoting,
    error,
    balance,
    canCreate,
    load,
    redeem,
    refreshLedger,
    requestQuote,
    applyBalance,
    reset,
  }
})
