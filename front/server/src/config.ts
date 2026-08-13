import type {
  AdminPermission,
  AdminRole,
  AssetKind,
  AuditResult,
  ConnectionStatus,
  ModelType,
  RedemptionCodeStatus,
  RetouchTicketStatus,
  TaskStatus,
  UserStatus,
  WorkspaceMode,
  PromptSections,
} from '@/types/domain'

export const APP_CONFIG = {
  name: '映研平台管理端',
  productCode: 'yingyan-client',
  defaultPageSize: 20,
  maxPageSize: 100,
  drawerWidthRatio: 2 / 3,
} as const

export const NETWORK_CONFIG = {
  requestTimeoutMs: 20_000,
  aiModelTestTimeoutMs: 210_000,
} as const

export const REM_CONFIG = {
  rootValue: 100,
  mobileDesignWidth: 375,
  mobileScaleMaxWidth: 375,
  tabletFixedMaxWidth: 768,
  compactDesktopMinWidth: 1200,
  compactDesktopMaxWidth: 1440,
  compactRootFontSize: 75,
  desktopDesignWidth: 1920,
  maxRootFontSize: 100,
} as const

export const MOCK_CONFIG = {
  databaseKey: 'yingyan-admin:mock-db:v1',
  rememberedSessionKey: 'yingyan-admin:mock-session',
  tabSessionKey: 'yingyan-admin:mock-tab-session',
  idempotencyTtlMs: 86_400_000,
  latencyMs: 120,
} as const

export const PERMISSIONS = {
  platformManage: 'platform:manage',
  retouchManage: 'retouch:manage',
} as const satisfies Record<string, AdminPermission>

export const ROLE_LABELS: Record<AdminRole, string> = {
  platform_admin: '平台管理员',
  retouch_operator: '修图操作员',
}

export const REDEMPTION_CONFIG = {
  minQuantity: 1,
  maxQuantity: 500,
  minCredits: 1,
  batchNameMaxLength: 60,
  defaultValidityDays: 90,
  expiringSoonDays: 7,
  codePrefix: 'YY',
} as const

export const REDEMPTION_STATUS_LABELS: Record<
  RedemptionCodeStatus,
  string
> = {
  unused: '未使用',
  redeemed: '已兑换',
  expired: '已过期',
  disabled: '已失效',
}

export const USER_STATUS_LABELS: Record<UserStatus, string> = {
  active: '正常',
  disabled: '已停用',
}

export const CONNECTION_STATUS_LABELS: Record<ConnectionStatus, string> = {
  untested: '未测试',
  healthy: '连接正常',
  error: '连接异常',
}

export const MODEL_TEST_STATUS_LABELS: Record<ConnectionStatus, string> = {
  untested: '未测试',
  healthy: '测试正常',
  error: '测试失败',
}

export const MODEL_TYPE_LABELS: Record<ModelType, string> = {
  chat: '对话模型',
  image: '生图模型',
}

export const TASK_STATUS_LABELS: Record<TaskStatus, string> = {
  queued: '排队中',
  processing: '生成中',
  completed: '已完成',
  partial: '部分完成',
  failed: '生成失败',
  cancelled: '已取消',
}

export const WORKSPACE_MODE_LABELS: Record<WorkspaceMode, string> = {
  'text-to-image': '文生图',
  'image-to-image': '图生图',
}

export const PROMPT_SECTION_LABELS: Record<keyof PromptSections, string> = {
  subject: '主体',
  scene: '场景',
  style: '风格',
  composition: '构图',
  details: '细节',
  negative: '规避项',
  output: '输出要求',
}

export const RETOUCH_STATUS_LABELS: Record<RetouchTicketStatus, string> = {
  submitted: '待评估',
  quote_pending: '待用户接受报价',
  accepted: '待开工',
  processing: '处理中',
  awaiting_confirmation: '待用户确认',
  delivered: '已交付',
  rejected: '已拒绝',
  cancelled: '已取消',
}

export const ASSET_KIND_LABELS: Record<AssetKind, string> = {
  source: '用户原图',
  reference: '创作参考图',
  'retouch-reference': '人工修图补充图',
  'ai-result': 'AI 生成结果',
  'retouch-result': '人工修图成片',
}

export const AUDIT_RESULT_LABELS: Record<AuditResult, string> = {
  success: '成功',
  failure: '失败',
}

export const TERMINAL_TASK_STATUSES = new Set<TaskStatus>([
  'completed',
  'partial',
  'failed',
  'cancelled',
])

export const TERMINAL_RETOUCH_STATUSES = new Set<RetouchTicketStatus>([
  'delivered',
  'rejected',
  'cancelled',
])

export const RETOUCH_UPLOAD_CONFIG = {
  maxFiles: 4,
  maxFileSize: 30 * 1024 * 1024,
  allowedTypes: ['image/jpeg', 'image/png', 'image/webp'],
} as const
