import type {
  AIModel,
  AIProvider,
  AdminSession,
  AdjustmentLedger,
  AuditEvent,
  ManagedAsset,
  ManagedGenerationTask,
  ManagedUser,
  ManageRetouchTicket,
  PlatformModelBindings,
  RedemptionBatch,
} from '@/types/domain'

export interface MockAdmin extends Omit<AdminSession, 'csrfToken'> {
  password: string
}

export type MockRedemptionBatch = Omit<
  RedemptionBatch,
  'counts' | 'usageRate'
>

export interface MockRedemptionCode {
  id: string
  fullCode: string
  batchId: string
  productCode: string
  credits: number
  expiresAt: string | null
  redeemedBy?: string
  redeemedAt?: string
  disabledBy?: string
  disabledAt?: string
  disabledReason?: string
  createdAt: string
  operationHistory: Array<{
    action: string
    operator: string
    reason?: string
    createdAt: string
  }>
}

export interface MockProvider extends AIProvider {
  apiKey: string
}

export type MockModel = Omit<
  AIModel,
  'providerName' | 'isPlatformModel'
>

export interface MockUser extends ManagedUser {
  password: string
  mustChangePassword?: boolean
}

export type MockTask = Omit<
  ManagedGenerationTask,
  'ownerEmail' | 'providerName' | 'modelName' | 'assets' | 'results'
> & {
  assetIds: string[]
  resultAssetIds: string[]
}

export type MockTicket = Omit<
  ManageRetouchTicket,
  'user' | 'sourceTaskDetail' | 'supplementalAssets'
> & {
  userId: string
  supplementalAssetIds: string[]
}

export interface MockIdempotencyEntry {
  operatorId: string
  path: string
  fingerprint: string
  result: unknown
  createdAt: string
}

export interface MockDb {
  admins: MockAdmin[]
  batches: MockRedemptionBatch[]
  codes: MockRedemptionCode[]
  providers: MockProvider[]
  models: MockModel[]
  bindings: PlatformModelBindings
  users: MockUser[]
  ledger: AdjustmentLedger[]
  assets: ManagedAsset[]
  tasks: MockTask[]
  tickets: MockTicket[]
  audits: AuditEvent[]
  idempotency: Record<string, MockIdempotencyEntry>
}
