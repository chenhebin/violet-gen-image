import type {
  Asset,
  AssetKind,
  AuthPayload,
  Entitlement,
  GenerationCreateResult,
  GenerationSettings,
  GenerationTask,
  LedgerEntry,
  PromptSections,
  PromptReferenceAsset,
  PromptVersion,
  RedemptionResult,
  RedemptionPreview,
  ReferenceRole,
  RegisterPayload,
  RetouchTicket,
  TaskStatus,
  UsageQuote,
  User,
  WorkspaceMode,
} from '@/types/domain'
import { NETWORK_CONFIG } from '@/config'
import type { PageResult } from '@/types/api'
import { apiRequest, createIdempotencyKey } from './http'
import {
  cacheAssetFile,
  loadAssetUrl,
  removeCachedAsset,
} from './asset-cache'

export const authApi = {
  register(payload: RegisterPayload) {
    return apiRequest<User>({
      method: 'POST',
      url: '/auth/register',
      data: payload,
    })
  },
  login(payload: AuthPayload) {
    return apiRequest<User>({
      method: 'POST',
      url: '/auth/login',
      data: payload,
    })
  },
  session() {
    return apiRequest<User>({ method: 'GET', url: '/auth/session' })
  },
  logout() {
    return apiRequest<null>({ method: 'POST', url: '/auth/logout' })
  },
}

export const entitlementApi = {
  get() {
    return apiRequest<Entitlement>({ method: 'GET', url: '/entitlements' })
  },
  ledger() {
    return apiRequest<LedgerEntry[]>({ method: 'GET', url: '/usage/ledger' })
  },
  previewRedemption(code: string, signal?: AbortSignal) {
    return apiRequest<RedemptionPreview>({
      method: 'POST',
      url: '/redemptions/preview',
      data: { code },
      signal,
    })
  },
  redeem(code: string, idempotencyKey = createIdempotencyKey('redeem')) {
    return apiRequest<RedemptionResult>({
      method: 'POST',
      url: '/redemptions/claim',
      data: { code },
      headers: { 'Idempotency-Key': idempotencyKey },
    })
  },
  quote(outputCount: number, signal?: AbortSignal) {
    return apiRequest<UsageQuote>({
      method: 'POST',
      url: '/usage/quote',
      data: { action: 'generate', outputCount },
      signal,
    })
  },
}

export interface AIProcessingNotice {
  version: string
  title: string
  providerDisclosure: string
  securitySummary: string
  purpose: string
  processingScope: string[]
  retentionDays: number
  stopUseDescription: string
  acknowledged: boolean
  acknowledgedAt?: string
}

export const aiNoticeApi = {
  get(signal?: AbortSignal) {
    return apiRequest<AIProcessingNotice>({
      method: 'GET',
      url: '/notices/ai-processing',
      signal,
    })
  },
  acknowledge(version: string, idempotencyKey = createIdempotencyKey('ai-notice-ack')) {
    return apiRequest<AIProcessingNotice>({
      method: 'POST',
      url: '/notices/ai-processing/ack',
      data: { version },
      headers: { 'Idempotency-Key': idempotencyKey },
    })
  },
}

export const assetApi = {
	getUrl(assetId: string, purpose: 'preview' | 'download' = 'preview') {
		return apiRequest<{ url: string; expiresAt: string }>({
			method: 'GET',
			url: `/assets/${assetId}/url`,
			params: { purpose },
		})
	},
  async upload(
    file: File,
    kind: AssetKind,
    role?: ReferenceRole,
    onProgress?: (progress: number) => void,
    signal?: AbortSignal,
  ): Promise<Asset> {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('kind', kind)
    if (role) formData.append('role', role)

    const asset = await apiRequest<Asset>({
      method: 'POST',
      url: '/assets',
      data: formData,
      headers: { 'Idempotency-Key': createIdempotencyKey('asset-upload') },
      signal,
      timeout: NETWORK_CONFIG.uploadTimeoutMs,
      onUploadProgress(event) {
        if (!event.total) return
        onProgress?.(Math.round((event.loaded / event.total) * 100))
      },
    })

    const cachedPreviewUrl = await cacheAssetFile(asset.id, file)
    return {
      ...asset,
      previewUrl: asset.previewUrl || cachedPreviewUrl,
      uploadProgress: 100,
    }
  },
	async hydrate(asset: Asset): Promise<Asset> {
		const expiresAt = asset.previewUrlExpiresAt ? Date.parse(asset.previewUrlExpiresAt) : 0
		if (asset.previewUrl && (!expiresAt || expiresAt - Date.now() > 60_000)) return asset
		try {
			const signed = await assetApi.getUrl(asset.id)
			return { ...asset, previewUrl: signed.url, previewUrlExpiresAt: signed.expiresAt }
		} catch {
			if (asset.previewUrl) return asset
			return { ...asset, previewUrl: await loadAssetUrl(asset.id) }
		}
	},
  async remove(assetId: string): Promise<void> {
    try {
      await apiRequest<null>({
        method: 'DELETE',
        url: `/assets/${assetId}`,
        headers: { 'Idempotency-Key': createIdempotencyKey('asset-delete') },
      })
    } finally {
      await removeCachedAsset(assetId)
    }
  },
}

export interface OptimizePromptPayload {
  source: string
  mode: WorkspaceMode
  sourceAssetIds: string[]
  referenceAssets: PromptReferenceAsset[]
  referencePrompt?: string
}

export interface ReferencePromptResult {
  prompt: string
  referenceAssets: PromptReferenceAsset[]
}

export const promptApi = {
  describeReferences(
    referenceAssets: PromptReferenceAsset[],
    signal?: AbortSignal,
  ): Promise<ReferencePromptResult> {
    return apiRequest<ReferencePromptResult>({
      method: 'POST',
      url: '/prompts/reference-prompt',
      data: { referenceAssets },
      signal,
      timeout: NETWORK_CONFIG.promptTimeoutMs,
    })
  },
  optimize(
    payload: OptimizePromptPayload,
    signal?: AbortSignal,
  ): Promise<PromptVersion> {
    return apiRequest<PromptVersion>({
      method: 'POST',
      url: '/prompts/optimize',
      data: payload,
      signal,
      timeout: NETWORK_CONFIG.promptTimeoutMs,
    })
  },
  confirm(
    id: string,
    source: string,
    sections: PromptSections,
    idempotencyKey = createIdempotencyKey('prompt-confirm'),
  ): Promise<PromptVersion> {
    return apiRequest<PromptVersion>({
      method: 'POST',
      url: '/prompts/confirm',
      data: { id, source, sections },
      headers: { 'Idempotency-Key': idempotencyKey },
    })
  },
}

export interface CreateGenerationPayload {
  promptVersionId?: string
  source?: string
  referenceAssets?: PromptReferenceAsset[]
  assetIds: string[]
  settings: GenerationSettings
}

export interface CreateRetouchTicketPayload {
  selectedResultIds: string[]
  requirement: string
  supplementalAssetIds: string[]
}

export interface RetouchTicketBalanceResult {
  ticket: RetouchTicket
  entitlement: Entitlement
}

export const generationApi = {
  async create(
    payload: CreateGenerationPayload,
    idempotencyKey = createIdempotencyKey('generation'),
  ): Promise<GenerationCreateResult> {
    const result = await apiRequest<{
      task: ApiGenerationTask
      entitlement: Entitlement
    }>({
      method: 'POST',
      url: '/generations',
      data: payload,
      headers: { 'Idempotency-Key': idempotencyKey },
    })
    const normalized = normalizeTask(result.task)
    return {
      ...normalized,
      assets: await Promise.all(normalized.assets.map(assetApi.hydrate)),
      entitlement: result.entitlement,
    } satisfies GenerationCreateResult
  },
}

type ApiGenerationTask = Omit<GenerationTask, 'status'> & {
  status: TaskStatus | 'failed'
}

function normalizeTask(task: ApiGenerationTask): GenerationTask {
  return {
    ...task,
    status: task.status === 'failed' ? 'failed-refunded' : task.status,
  }
}

export const taskApi = {
  async list(
    query: { page?: number; pageSize?: number } = {},
    signal?: AbortSignal,
  ): Promise<PageResult<GenerationTask>> {
    const page = await apiRequest<PageResult<ApiGenerationTask>>({
      method: 'GET',
      url: '/tasks',
      params: query,
      signal,
    })
    return {
      ...page,
      items: await Promise.all(
        page.items.map(async (task) => {
        const normalized = normalizeTask(task)
        return {
          ...normalized,
          assets: await Promise.all(normalized.assets.map(assetApi.hydrate)),
        }
        }),
      ),
    }
  },
  async get(
    taskId: string,
    signal?: AbortSignal,
  ): Promise<GenerationTask> {
    const task = await apiRequest<ApiGenerationTask>({
      method: 'GET',
      url: `/tasks/${taskId}`,
      signal,
    })
    const normalized = normalizeTask(task)
    return {
      ...normalized,
      assets: await Promise.all(normalized.assets.map(assetApi.hydrate)),
    }
  },
  async cancel(
    taskId: string,
    idempotencyKey = createIdempotencyKey('task-cancel'),
  ): Promise<GenerationTask> {
    const task = await apiRequest<ApiGenerationTask>({
      method: 'POST',
      url: `/tasks/${taskId}/cancel`,
      headers: { 'Idempotency-Key': idempotencyKey },
    })
    return normalizeTask(task)
  },
}

export const retouchTicketApi = {
  list(query: { page?: number; pageSize?: number } = {}, signal?: AbortSignal) {
    return apiRequest<PageResult<RetouchTicket>>({
      method: 'GET',
      url: '/retouch-tickets',
      params: query,
      signal,
    })
  },
  get(ticketId: string, signal?: AbortSignal) {
    return apiRequest<RetouchTicket>({
      method: 'GET',
      url: `/retouch-tickets/${ticketId}`,
      signal,
    })
  },
  create(
    taskId: string,
    payload: CreateRetouchTicketPayload,
    idempotencyKey = createIdempotencyKey('retouch-create'),
    signal?: AbortSignal,
  ) {
    return apiRequest<RetouchTicket>({
      method: 'POST',
      url: `/tasks/${taskId}/retouch-tickets`,
      data: payload,
      headers: { 'Idempotency-Key': idempotencyKey },
      signal,
    })
  },
  acceptQuote(
    ticketId: string,
    quoteId: string,
    idempotencyKey = createIdempotencyKey('retouch-accept'),
    signal?: AbortSignal,
  ) {
    return apiRequest<RetouchTicketBalanceResult>({
      method: 'POST',
      url: `/retouch-tickets/${ticketId}/quote/accept`,
      data: { quoteId },
      headers: { 'Idempotency-Key': idempotencyKey },
      signal,
    })
  },
  cancel(
    ticketId: string,
    idempotencyKey = createIdempotencyKey('retouch-cancel'),
    signal?: AbortSignal,
  ) {
    return apiRequest<RetouchTicketBalanceResult>({
      method: 'POST',
      url: `/retouch-tickets/${ticketId}/cancel`,
      headers: { 'Idempotency-Key': idempotencyKey },
      signal,
    })
  },
  confirm(
    ticketId: string,
    idempotencyKey = createIdempotencyKey('retouch-confirm'),
    signal?: AbortSignal,
  ) {
    return apiRequest<RetouchTicket>({
      method: 'POST',
      url: `/retouch-tickets/${ticketId}/confirm`,
      headers: { 'Idempotency-Key': idempotencyKey },
      signal,
    })
  },
  requestRevision(
    ticketId: string,
    message: string,
    idempotencyKey = createIdempotencyKey('retouch-revision'),
    signal?: AbortSignal,
  ) {
    return apiRequest<RetouchTicket>({
      method: 'POST',
      url: `/retouch-tickets/${ticketId}/revisions`,
      data: { message },
      headers: { 'Idempotency-Key': idempotencyKey },
      signal,
    })
  },
}
