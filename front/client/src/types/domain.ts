export type WorkspaceMode = 'text-to-image' | 'image-to-image'
export type AssetKind = 'source' | 'reference' | 'retouch-reference'
export type ReferenceRole = 'style' | 'composition' | 'person' | 'detail'
export type TaskStatus =
  | 'draft'
  | 'queued'
  | 'processing'
  | 'completed'
  | 'partial'
  | 'failed-refunded'
  | 'cancelled'
export type RetouchTicketStatus =
  | 'submitted'
  | 'quote_pending'
  | 'accepted'
  | 'processing'
  | 'awaiting_confirmation'
  | 'delivered'
  | 'rejected'
  | 'cancelled'

export interface User {
  id: string
  email: string
  createdAt: string
  status: 'active' | 'disabled'
}

export interface AuthPayload {
  email: string
  password: string
  remember?: boolean
}

export interface RegisterPayload extends AuthPayload {
  acceptedTerms: boolean
}

export interface Entitlement {
  balance: number
  canCreate: boolean
  status: 'unredeemed' | 'active' | 'empty'
}

export interface LedgerEntry {
  id: string
  type: 'redemption' | 'reserve' | 'release' | 'refund' | 'adjustment'
  amount: number
  balanceAfter: number
  description: string
  createdAt: string
}

export interface RedemptionResult {
  added: number
  entitlement: Entitlement
}

export interface RedemptionPreview {
  valid: true
  credits: number
  productName: string
  maskedCode: string
  expiresAt: string | null
}

export interface Asset {
  id: string
  name: string
  kind: AssetKind
  role?: ReferenceRole
  mimeType: string
	size: number
	previewUrl?: string
	previewUrlExpiresAt?: string
	uploadProgress: number
}

export interface PromptSections {
  subject: string
  scene: string
  style: string
  composition: string
  details: string
  negative: string
  output: string
  referencePrompt?: string
}

export type PromptSectionBackups = Partial<PromptSections>

export interface PromptVersion {
  id: string
  source: string
  sections: PromptSections
  confirmedAt?: string
}

export interface PromptReferenceAsset {
  assetId: string
  role: ReferenceRole
}

export interface GenerationSettings {
  aspectRatio: '1:1' | '3:4' | '4:3' | '9:16' | '16:9'
  outputCount: 1 | 2 | 3 | 4
  referenceStrength: number
}

export interface UsageQuote {
  action: 'generate'
  cost: number
  balance: number
  canSubmit: boolean
}

export interface GenerationResult {
  id: string
  url: string
  downloadUrl?: string
  urlExpiresAt?: string
  width: number
  height: number
}

export interface RetouchTicketSummary {
  id: string
  ticketNo: string
  status: RetouchTicketStatus
  updatedAt: string
  quoteCredits?: number
}

export interface RetouchTicketTimelineEntry {
  status: RetouchTicketStatus
  note?: string
  createdAt: string
}

export interface RetouchQuote {
  id: string
  credits: number
  createdAt: string
  status: 'active' | 'accepted' | 'invalidated' | 'expired'
  expiresAt: string
  remainingSeconds: number
}

export interface RetouchSLA {
  stage: 'quote' | 'first-delivery' | 'revision' | 'completed'
  dueAt: string | null
  overdue: boolean
  remainingSeconds: number | null
}

export interface RetouchRevision {
  message: string
  requestedAt: string
}

export interface RetouchTicket {
  id: string
  ticketNo: string
  taskId: string
  taskTitle: string
  status: RetouchTicketStatus
  selectedResults: GenerationResult[]
  requirement: string
  supplementalAssets: Asset[]
  quote?: RetouchQuote
  timeline: RetouchTicketTimelineEntry[]
  reservedCredits: number
  spentCredits: number
  refundedCredits: number
  revision?: RetouchRevision
  deliverables: GenerationResult[]
  sla: RetouchSLA
  createdAt: string
  updatedAt: string
}

export interface GenerationTask {
  id: string
  mode: WorkspaceMode
  title: string
  status: TaskStatus
  prompt: PromptVersion
  settings: GenerationSettings
  assets: Asset[]
  requestedCount: number
  successfulCount: number
  reservedCredits: number
  spentCredits: number
  refundedCredits: number
  progress: number
  results: GenerationResult[]
  retouchTicket?: RetouchTicketSummary
  createdAt: string
  updatedAt: string
}

export interface GenerationCreateResult extends GenerationTask {
  entitlement?: Entitlement
}

export interface WorkspaceDraft {
  mode: WorkspaceMode
  sourcePrompt: string
  assets: Asset[]
  referencePrompt: string
  promptVersion: PromptVersion | null
  promptSectionBackups: PromptSectionBackups
  settings: GenerationSettings
}
