import type {
  GenerationSettings,
  PromptSections,
  ReferenceRole,
  RetouchTicketStatus,
  TaskStatus,
} from '@/types/domain'

export const REM_CONFIG = {
  rootValue: 100,
  mobileDesignWidth: 375,
  mobileScaleMaxWidth: 375,
  tabletFixedMaxWidth: 768,
  compactDesktopMinWidth: 1200,
  compactDesktopMaxWidth: 1440,
  desktopDesignWidth: 1920,
  compactRootFontSize: 75,
  maxRootFontSize: 100,
} as const

export const LAYOUT_CONFIG = {
  mobileBreakpoint: 760,
  stackedWorkspaceBreakpoint: 900,
  authBreakpoint: 840,
  compactMobileBreakpoint: 560,
  headerHeight: 64,
  mobileNavigationHeight: 58,
  mobileSafeGap: 12,
} as const

export const REFERENCE_ROLE_OPTIONS = [
  { value: 'style', label: '风格' },
  { value: 'composition', label: '构图' },
  { value: 'person', label: '人物/服装' },
  { value: 'detail', label: '局部细节' },
] as const satisfies ReadonlyArray<{ value: ReferenceRole; label: string }>

export const PROMPT_SECTION_OPTIONS = [
  { key: 'subject', label: '主体' },
  { key: 'scene', label: '场景' },
  { key: 'style', label: '风格' },
  { key: 'composition', label: '构图' },
  { key: 'details', label: '细节' },
  { key: 'negative', label: '禁止内容' },
  { key: 'output', label: '输出规格' },
] as const satisfies ReadonlyArray<{ key: keyof PromptSections; label: string }>

export const TASK_FILTER_OPTIONS = [
  { value: 'all', label: '全部' },
  { value: 'running', label: '进行中' },
  { value: 'completed', label: '已结算' },
] as const

export type TaskFilter = (typeof TASK_FILTER_OPTIONS)[number]['value']

export const TASK_STATUS_LABELS = {
  draft: '草稿',
  queued: '排队中',
  processing: '处理中',
  completed: '已完成',
  partial: '部分完成',
  'failed-refunded': '失败并退款',
  cancelled: '已取消',
} as const satisfies Record<TaskStatus, string>

export const TASK_STAGE_STATUS_LABELS = {
  ...TASK_STATUS_LABELS,
  queued: '任务正在排队',
  processing: '正在生成画面',
  completed: '生成完成',
  partial: '部分图片已完成',
  'failed-refunded': '生成失败，次数已退回',
  cancelled: '任务已取消',
} as const satisfies Record<TaskStatus, string>

export const TASK_RUNNING_STATUSES = ['queued', 'processing'] as const
export const TASK_FINAL_STATUSES = [
  'completed',
  'partial',
  'failed-refunded',
  'cancelled',
] as const

export const TASK_TIMING = {
  monitorPollMs: 1_000,
  listRefreshMs: 1_500,
} as const

export const RETOUCH_TICKET_STATUS_LABELS = {
  submitted: '已提交',
  quote_pending: '待确认报价',
  accepted: '已接受报价',
  processing: '处理中',
  awaiting_confirmation: '待确认',
  delivered: '已交付',
  rejected: '已拒绝',
  cancelled: '已取消',
} as const satisfies Record<RetouchTicketStatus, string>

export const RETOUCH_TICKET_FINAL_STATUSES = [
  'delivered',
  'rejected',
  'cancelled',
] as const satisfies ReadonlyArray<RetouchTicketStatus>

export const RETOUCH_TICKET_CONFIG = {
  quoteCredits: 3,
  maxSelectedResults: 4,
  maxSupplementalAssets: 4,
  requirementMaxLength: 1_000,
  revisionMaxLength: 500,
  maxRevisionCount: 1,
} as const

export const RETOUCH_TICKET_TIMING = {
  quoteDelayMs: 1_000,
  acceptedDelayMs: 800,
  processingDurationMs: 2_000,
  revisionProcessingMs: 1_800,
  listPollMs: 1_500,
  detailPollMs: 1_000,
} as const

export const ASSET_CONFIG = {
  maxCount: 8,
  maxFileSize: 15 * 1024 * 1024,
  maxFileSizeLabel: '15MB',
  acceptedMimeTypes: ['image/jpeg', 'image/png', 'image/webp'],
  acceptAttribute: 'image/jpeg,image/png,image/webp',
} as const

export const PROMPT_CONFIG = {
  minLength: 6,
  maxLength: 600,
  unchangedText: '保持不变',
} as const

export const DEFAULT_GENERATION_SETTINGS = {
  aspectRatio: '3:4',
  outputCount: 1,
  referenceStrength: 68,
} as const satisfies GenerationSettings

export const NETWORK_CONFIG = {
  requestTimeoutMs: 20_000,
  uploadTimeoutMs: 120_000,
  promptTimeoutMs: 135_000,
} as const

export const UI_TIMING = {
  toastDurationMs: 4_200,
} as const

export const AUTH_CONFIG = {
  minimumPasswordLength: 8,
} as const

export const MOCK_REDEMPTION_CODES = [
  { code: 'YINGYAN-START-10', credits: 10, state: 'active' },
  { code: 'YINGYAN-PRO-30', credits: 30, state: 'active' },
  { code: 'YINGYAN-USED-10', credits: 10, state: 'used' },
  { code: 'YINGYAN-EXPIRED-10', credits: 10, state: 'expired' },
] as const

export const STAGE_VIEW_OPTIONS = [
  { value: 'source', label: '原图' },
  { value: 'result', label: '结果' },
] as const

export type StageView = (typeof STAGE_VIEW_OPTIONS)[number]['value']

export function isRunningTaskStatus(status: TaskStatus): boolean {
  return TASK_RUNNING_STATUSES.some((item) => item === status)
}

export function isFinalTaskStatus(status: TaskStatus): boolean {
  return TASK_FINAL_STATUSES.some((item) => item === status)
}

export function isFinalRetouchTicketStatus(
  status: RetouchTicketStatus,
): boolean {
  return RETOUCH_TICKET_FINAL_STATUSES.some((item) => item === status)
}
