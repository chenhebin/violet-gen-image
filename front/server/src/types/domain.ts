import type { PageQuery } from '@/types/api'

export type AdminRole = 'platform_admin' | 'retouch_operator'
export type AdminPermission = 'platform:manage' | 'retouch:manage'
export type UserStatus = 'active' | 'disabled'
export type RedemptionCodeStatus =
  | 'unused'
  | 'redeemed'
  | 'expired'
  | 'disabled'
export type ConnectionStatus = 'untested' | 'healthy' | 'error'
export type ModelType = 'chat' | 'image'
export type WorkspaceMode = 'text-to-image' | 'image-to-image'
export type TaskStatus =
  | 'queued'
  | 'processing'
  | 'completed'
  | 'partial'
  | 'failed'
  | 'cancelled'
export type AssetKind =
  | 'source'
  | 'reference'
  | 'retouch-reference'
  | 'ai-result'
  | 'retouch-result'
export type ReferenceRole = 'style' | 'composition' | 'person' | 'detail'
export type RetouchTicketStatus =
  | 'submitted'
  | 'quote_pending'
  | 'accepted'
  | 'processing'
  | 'awaiting_confirmation'
  | 'delivered'
  | 'rejected'
  | 'cancelled'
export type LedgerType =
  | 'redemption'
  | 'reserve'
  | 'release'
  | 'refund'
  | 'adjustment'
export type AuditResult = 'success' | 'failure'

export interface AdminSession {
  id: string
  email: string
  name: string
  role: AdminRole
  permissions: AdminPermission[]
  status: UserStatus
  csrfToken: string
  createdAt: string
}

export interface AdminLoginPayload {
  email: string
  password: string
  remember?: boolean
}

export interface DashboardMetric {
  key:
    | 'unusedCodes'
    | 'expiringCodes'
    | 'redeemedToday'
    | 'creditsGrantedToday'
    | 'failedTasks'
    | 'pendingTickets'
    | 'overdueTickets'
    | 'dueSoonTickets'
  label: string
  value: number
  tone: 'neutral' | 'positive' | 'warning' | 'danger'
}

export interface ModelSummary {
  id: string
  displayName: string
  modelId: string
  providerName: string
  type: ModelType
}

export interface DashboardAlert {
  id: string
  type: 'provider' | 'task' | 'redemption' | 'retouch'
  title: string
  description: string
  tone: 'warning' | 'danger'
  href: string
}

export interface DashboardData {
  metrics: DashboardMetric[]
  currentModels: {
    chat?: ModelSummary
    image?: ModelSummary
  }
  alerts: DashboardAlert[]
  pendingTickets: ManageRetouchTicketSummary[]
  recentBatches: RedemptionBatch[]
}

export interface RedemptionBatch {
  id: string
  name: string
  productCode: string
  quantity: number
  creditsPerCode: number
  expiresAt: string | null
  neverExpires: boolean
  note?: string
  createdBy: string
  createdAt: string
  counts: Record<RedemptionCodeStatus, number>
  usageRate: number
}

export interface RedemptionCode {
  id: string
  maskedCode: string
  batchId: string
  batchName: string
  productCode: string
  credits: number
  status: RedemptionCodeStatus
  expiresAt: string | null
  redeemedBy?: string
  redeemedByEmail?: string
  redeemedAt?: string
  disabledBy?: string
  disabledAt?: string
  disabledReason?: string
  createdAt: string
  expiringSoon: boolean
}

export interface RedemptionCodeDetail extends RedemptionCode {
  operationHistory: Array<{
    action: string
    operator: string
    reason?: string
    createdAt: string
  }>
}

export interface CreateRedemptionBatchPayload {
  name: string
  quantity: number
  creditsPerCode: number
  productCode: string
  expiresAt?: string | null
  neverExpires?: boolean
  note?: string
}

export interface CreateRedemptionBatchResult {
  batch: RedemptionBatch
  codes: Array<{
    id: string
    fullCode: string
    maskedCode: string
  }>
}

export interface UpdateRedemptionBatchPayload {
  name: string
}

export interface RedemptionCodeQuery extends PageQuery {
  status?: RedemptionCodeStatus
  batchId?: string
  productCode?: string
  redeemedBy?: string
  expiringSoon?: boolean
}

export interface RedemptionBatchQuery extends PageQuery {
  productCode?: string
}

export interface BulkMutationResult {
  affected: number
  skipped: number
  failed: number
}

export interface DisableRedemptionPayload {
  codeIds?: string[]
  batchId?: string
  reason: string
}

export interface ExtendRedemptionPayload {
  codeIds?: string[]
  batchId?: string
  expiresAt: string | null
  reason: string
}

export interface AIProvider {
  id: string
  name: string
  code: string
  protocol: 'openai-compatible'
  baseUrl: string
  maskedApiKey: string
  enabled: boolean
  connectionStatus: ConnectionStatus
  lastTest?: {
    testedAt: string
    success: boolean
    message: string
    requestSummary?: ProviderRequestSummary | null
  }
  note?: string
  createdAt: string
  updatedAt: string
}

export interface CreateAIProviderPayload {
  name: string
  code: string
  baseUrl: string
  apiKey: string
  enabled?: boolean
  note?: string
}

export interface UpdateAIProviderPayload {
  name?: string
  baseUrl?: string
  enabled?: boolean
  note?: string
}

export interface AIModelCapabilities {
  promptOptimization?: boolean
  visionInput?: boolean
  textToImage?: boolean
  imageToImage?: boolean
}

export interface AIModel {
  id: string
  providerId: string
  providerName: string
  displayName: string
  modelId: string
  type: ModelType
  enabled: boolean
  connectionStatus: ConnectionStatus
  capabilities: AIModelCapabilities
  lastTestAt?: string
  lastTest?: {
    testedAt: string
    success: boolean
    message: string
    requestSummary?: ProviderRequestSummary | null
  }
  createdAt: string
  updatedAt: string
  isPlatformModel: boolean
}

export interface ProviderRequestSummary {
  operation: string
  method: string
  path: string
  model?: string
  parameterSummary?: Record<string, unknown>
  status?: number
  latencyMs: number
  requestId?: string
  errorKind?: string
}

export interface CreateAIModelPayload {
  providerId: string
  displayName: string
  modelId: string
  type: ModelType
  enabled?: boolean
  capabilities: AIModelCapabilities
}

export interface UpdateAIModelPayload {
  displayName?: string
  modelId?: string
  enabled?: boolean
  capabilities?: AIModelCapabilities
}

export interface PlatformModelBindings {
  chatModelId: string | null
  imageModelId: string | null
}

export interface ManagedUser {
  id: string
  email: string
  status: UserStatus
  balance: number
  totalRedeemed: number
  totalConsumed: number
  taskCount: number
  ticketCount: number
  lastLoginAt?: string
  createdAt: string
  disabledReason?: string
}

export interface ManagedUserDetail extends ManagedUser {
  ledger: AdjustmentLedger[]
  redemptionCodes: RedemptionCode[]
  tasks: ManagedGenerationTaskSummary[]
  tickets: ManageRetouchTicketSummary[]
}

export interface ManagedUserQuery extends PageQuery {
  status?: UserStatus
  minBalance?: number
  maxBalance?: number
  hasTasks?: boolean
  hasTickets?: boolean
}

export interface AdjustmentLedger {
  id: string
  userId: string
  type: LedgerType
  amount: number
  balanceBefore: number
  balanceAfter: number
  description: string
  reason?: string
  referenceNo?: string
  operatorId?: string
  createdAt: string
}

export interface AdjustCreditsPayload {
  amount: number
  reason: string
  referenceNo?: string
}

export interface ResetPasswordResult {
  temporaryPassword: string
  expiresAt: string
}

export interface PromptSections {
  subject: string
  scene: string
  style: string
  composition: string
  details: string
  negative: string
  output: string
}

export interface ManagedAsset {
  id: string
  ownerId: string
  ownerEmail: string
  name: string
  kind: AssetKind
  role?: ReferenceRole
  mimeType: string
  size: number
  width: number
  height: number
	previewUrl?: string
	previewUrlExpiresAt?: string
	taskId?: string
  ticketId?: string
  retained: boolean
  retentionExpiresAt: string | null
  deletedAt?: string
  createdAt: string
}

export interface AssetQuery extends PageQuery {
  kind?: AssetKind
  userId?: string
  taskId?: string
  ticketId?: string
  retained?: boolean
}

export interface ManagedGenerationTaskSummary {
  id: string
  ownerId: string
  ownerEmail: string
  title: string
  mode: WorkspaceMode
  status: TaskStatus
  progress: number
  requestedCount: number
  successfulCount: number
  reservedCredits: number
  spentCredits: number
  refundedCredits: number
  providerName: string
  modelName: string
  hasRetouchTicket: boolean
  createdAt: string
  updatedAt: string
}

export interface ManagedGenerationTask extends ManagedGenerationTaskSummary {
  sourceRequirement: string
  optimizedPrompt: PromptSections
  confirmedPrompt: PromptSections
  settings: {
    aspectRatio: string
    outputCount: number
    referenceStrength: number
  }
  assets: ManagedAsset[]
  results: ManagedAsset[]
  executionSnapshot: {
    providerId: string
    providerName: string
    modelId: string
    modelName: string
    configVersion: number
  }
  providerAttempts?: ProviderAttempt[]
  errorMessage?: string
  retouchTicket?: ManageRetouchTicketSummary
}

export interface ProviderAttempt {
  id: string
  jobId: string
  outputIndex?: number
  attemptNo: number
  operation: string
  method: string
  path: string
  model: string
  status: string
  externalRequestId?: string
  responseStatus?: number
  latencyMs: number
  errorCode?: string
  errorKind?: string
  errorSummary?: string
  requestSummary?: Record<string, unknown>
  responseMetadata?: Record<string, unknown>
  startedAt: string
  completedAt?: string
}

export interface ManagedTaskQuery extends PageQuery {
  status?: TaskStatus
  mode?: WorkspaceMode
  userId?: string
  providerId?: string
  modelId?: string
  hasRetouchTicket?: boolean
}

export interface GenerationResult {
  id: string
  url: string
  urlExpiresAt?: string
  width: number
  height: number
}

export interface RetouchTimelineEntry {
  status: RetouchTicketStatus
  action: string
  note?: string
  createdAt: string
}

export interface ManageRetouchTicketSummary {
  id: string
  ticketNo: string
  taskId: string
  taskTitle: string
  status: RetouchTicketStatus
  quoteCredits?: number
  sla: RetouchSLA
  user: Pick<ManagedUser, 'id' | 'email' | 'status'>
  createdAt: string
  updatedAt: string
}

export interface ManageRetouchTicket extends ManageRetouchTicketSummary {
  selectedResults: GenerationResult[]
  requirement: string
  supplementalAssets: ManagedAsset[]
  quote?: {
    id: string
    credits: number
    createdAt: string
    status: 'active' | 'accepted' | 'invalidated' | 'expired'
    expiresAt: string
    remainingSeconds: number
  }
  timeline: RetouchTimelineEntry[]
  reservedCredits: number
  spentCredits: number
  refundedCredits: number
  revision?: {
    message: string
    requestedAt: string
  }
  deliverables: GenerationResult[]
  sourceTaskDetail: Pick<
    ManagedGenerationTask,
    'id' | 'title' | 'mode' | 'status' | 'modelName' | 'sourceRequirement'
  >
}

export interface RetouchTicketQuery extends PageQuery {
  status?: RetouchTicketStatus
  sla?: 'overdue' | 'due-soon'
}

export interface RetouchSLA {
  stage: 'quote' | 'first-delivery' | 'revision' | 'completed'
  dueAt: string | null
  overdue: boolean
  remainingSeconds: number | null
}

export interface AuditEvent {
  id: string
  operatorId: string
  operatorEmail: string
  operatorRole: AdminRole
  action: string
  resourceType: string
  resourceId: string
  before?: Record<string, unknown>
  after?: Record<string, unknown>
  reason?: string
  result: AuditResult
  requestId: string
  ip?: string
  device?: string
  createdAt: string
}

export interface AuditQuery extends PageQuery {
  operatorId?: string
  action?: string
  resourceType?: string
  result?: AuditResult
  startAt?: string
  endAt?: string
}
