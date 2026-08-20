import type {
  ManagedAsset,
  PromptSections,
  RetouchTicketStatus,
} from '@/types/domain'
import type {
  MockDb,
  MockRedemptionCode,
  MockTask,
  MockTicket,
} from '@/mocks/schema'
import { maskRedemptionCode } from '@/utils/redemption'

const MOCK_ADMIN_ACCOUNTS = {
  admin: {
    email: 'admin@yingyan.local',
    password: 'Admin1234!',
  },
  retouch: {
    email: 'retouch@yingyan.local',
    password: 'Retouch1234!',
  },
} as const

const DAY = 86_400_000

function iso(offsetMs = 0): string {
  return new Date(Date.now() + offsetMs).toISOString()
}

const prompt: PromptSections = {
  subject: '成年女性半身人像，保留人物五官比例与身份特征',
  scene: '低饱和灰色影棚，自然窗边光',
  style: '克制、通透的时尚杂志人像',
  composition: '居中半身构图，留出呼吸空间',
  details: '保留皮肤纹理，整理碎发与衣物褶皱',
  negative: '避免过度磨皮、塑料质感、手指和五官变形',
  output: '3:4 竖图，细节清晰',
}

function asset(
  id: string,
  ownerId: string,
  ownerEmail: string,
  name: string,
  kind: ManagedAsset['kind'],
  previewUrl: string,
  taskId?: string,
  ticketId?: string,
): ManagedAsset {
  return {
    id,
    ownerId,
    ownerEmail,
    name,
    kind,
    role: kind === 'reference' ? 'style' : undefined,
    mimeType: 'image/jpeg',
    size: 428_000,
    width: 1200,
    height: 1800,
    previewUrl,
    taskId,
    ticketId,
    retained: false,
    retentionExpiresAt: iso(82 * DAY),
    createdAt: iso(-8 * DAY),
  }
}

function code(
  id: string,
  fullCode: string,
  batchId: string,
  expiresAt: string | null,
  extra: Partial<MockRedemptionCode> = {},
): MockRedemptionCode {
  return {
    id,
    fullCode,
    batchId,
    productCode: 'yingyan-client',
    credits: batchId === 'batch_vip' ? 30 : 10,
    expiresAt,
    createdAt: iso(-14 * DAY),
    operationHistory: [
      {
        action: 'generated',
        operator: 'admin@yingyan.local',
        createdAt: iso(-14 * DAY),
      },
    ],
    ...extra,
  }
}

function task(
  id: string,
  ownerId: string,
  title: string,
  status: MockTask['status'],
  assetIds: string[],
  resultAssetIds: string[],
  providerId = 'provider_test1',
  modelId = 'model_test1_image',
): MockTask {
  const requestedCount = Math.max(2, resultAssetIds.length)
  return {
    id,
    ownerId,
    title,
    mode: 'image-to-image',
    status,
    progress:
      status === 'completed' || status === 'partial' || status === 'failed'
        ? 100
        : 62,
    requestedCount,
    successfulCount: resultAssetIds.length,
    reservedCredits: requestedCount,
    spentCredits: resultAssetIds.length,
    refundedCredits: requestedCount - resultAssetIds.length,
    hasRetouchTicket: true,
    sourceRequirement:
      '把人物修成自然通透的杂志人像，保留真实皮肤质感，并参考海岸照片的色调。',
    optimizedPrompt: prompt,
    confirmedPrompt: prompt,
    settings: {
      aspectRatio: '3:4',
      outputCount: requestedCount,
      referenceStrength: 68,
    },
    executionSnapshot: {
      providerId,
      providerName: providerId === 'provider_test1' ? 'test1' : 'test2',
      modelId,
      modelName:
        modelId === 'model_test1_image'
          ? 'Photon Studio Image'
          : 'Vision Beta',
      configVersion: 3,
    },
    assetIds,
    resultAssetIds,
    createdAt: iso(-8 * DAY),
    updatedAt: iso(-8 * DAY + 15 * 60_000),
  }
}

function ticket(
  id: string,
  ticketNo: string,
  userId: string,
  taskId: string,
  status: RetouchTicketStatus,
  extra: Partial<MockTicket> = {},
): MockTicket {
  const taskTitle =
    taskId === 'task_anna_completed' ? '自然光人像精修' : '旅行照片风格统一'
  return {
    id,
    ticketNo,
    userId,
    taskId,
    taskTitle,
    status,
    selectedResults: [
      {
        id: `${id}_selected`,
        url: '/demo/result-blue.jpg',
        width: 1200,
        height: 1800,
      },
    ],
    requirement: 'AI 成片肤色略显生硬，请自然处理皮肤纹理和碎发。',
    supplementalAssetIds: [],
    timeline: [
      {
        status: 'submitted',
        action: 'submitted',
        note: '用户提交人工修图需求',
        createdAt: iso(-2 * DAY),
      },
    ],
    reservedCredits: 0,
    spentCredits: 0,
    refundedCredits: 0,
    deliverables: [],
    sla: {
      stage: 'quote',
      dueAt: iso(22 * 60 * 60_000),
      overdue: false,
      remainingSeconds: 22 * 60 * 60,
    },
    createdAt: iso(-2 * DAY),
    updatedAt: iso(-2 * DAY),
    ...extra,
  }
}

function quote(
  id: string,
  credits: number,
  createdAt: string,
  status: 'active' | 'accepted' | 'invalidated' | 'expired' = 'active',
): NonNullable<MockTicket['quote']> {
  const expiresAt = new Date(Date.parse(createdAt) + 48 * 60 * 60_000).toISOString()
  return {
    id,
    credits,
    createdAt,
    status,
    expiresAt,
    remainingSeconds: status === 'active'
      ? Math.max(0, Math.floor((Date.parse(expiresAt) - Date.now()) / 1000))
      : 0,
  }
}

export function createSeedDb(): MockDb {
  const users = [
    {
      id: 'user_anna',
      email: 'anna@example.com',
      password: 'User1234!',
      status: 'active' as const,
      balance: 18,
      totalRedeemed: 40,
      totalConsumed: 22,
      taskCount: 4,
      ticketCount: 3,
      lastLoginAt: iso(-2 * 60_000),
      createdAt: iso(-45 * DAY),
    },
    {
      id: 'user_mia',
      email: 'mia@example.com',
      password: 'User1234!',
      status: 'active' as const,
      balance: 6,
      totalRedeemed: 20,
      totalConsumed: 14,
      taskCount: 2,
      ticketCount: 2,
      lastLoginAt: iso(-DAY),
      createdAt: iso(-28 * DAY),
    },
    {
      id: 'user_lina',
      email: 'lina@example.com',
      password: 'User1234!',
      status: 'disabled' as const,
      balance: 0,
      totalRedeemed: 10,
      totalConsumed: 10,
      taskCount: 1,
      ticketCount: 1,
      disabledReason: '用户主动申请暂停账号',
      lastLoginAt: iso(-20 * DAY),
      createdAt: iso(-70 * DAY),
    },
  ]

  const assets = [
    asset(
      'asset_anna_source',
      'user_anna',
      'anna@example.com',
      '人物原图.jpg',
      'source',
      '/demo/source-portrait.jpg',
      'task_anna_completed',
    ),
    asset(
      'asset_anna_reference',
      'user_anna',
      'anna@example.com',
      '海岸色调参考.jpg',
      'reference',
      '/demo/style-coast.jpg',
      'task_anna_completed',
    ),
    asset(
      'asset_anna_result',
      'user_anna',
      'anna@example.com',
      'AI 成片 01.jpg',
      'ai-result',
      '/demo/result-blue.jpg',
      'task_anna_completed',
    ),
    asset(
      'asset_mia_source',
      'user_mia',
      'mia@example.com',
      '旅行人物原图.jpg',
      'source',
      '/demo/auth-studio.jpg',
      'task_mia_partial',
    ),
    asset(
      'asset_mia_result',
      'user_mia',
      'mia@example.com',
      'AI 成片 01.jpg',
      'ai-result',
      '/demo/result-blue.jpg',
      'task_mia_partial',
    ),
    asset(
      'asset_anna_retouch',
      'user_anna',
      'anna@example.com',
      '人工补充参考.jpg',
      'retouch-reference',
      '/demo/style-coast.jpg',
      'task_anna_completed',
      'ticket_submitted',
    ),
  ]

  const tasks = [
    task(
      'task_anna_completed',
      'user_anna',
      '自然光人像精修',
      'completed',
      ['asset_anna_source', 'asset_anna_reference'],
      ['asset_anna_result'],
    ),
    task(
      'task_mia_partial',
      'user_mia',
      '旅行照片风格统一',
      'partial',
      ['asset_mia_source'],
      ['asset_mia_result'],
    ),
    {
      ...task(
        'task_anna_processing',
        'user_anna',
        '日落氛围人像',
        'processing',
        ['asset_anna_source'],
        [],
      ),
      hasRetouchTicket: false,
      createdAt: iso(-20 * 60_000),
      updatedAt: iso(-4 * 60_000),
    },
    {
      ...task(
        'task_lina_failed',
        'user_lina',
        '复古胶片合成',
        'failed',
        [],
        [],
        'provider_test2',
        'model_test2_image',
      ),
      hasRetouchTicket: false,
      errorMessage: '服务商响应超时，已全额退款',
    },
  ]

  const tickets = [
    ticket(
      'ticket_submitted',
      'YY20260730-A1B2C3',
      'user_anna',
      'task_anna_completed',
      'submitted',
      { supplementalAssetIds: ['asset_anna_retouch'] },
    ),
    ticket(
      'ticket_quote',
      'YY20260729-D4E5F6',
      'user_mia',
      'task_mia_partial',
      'quote_pending',
      {
        quote: quote('quote_seed', 4, iso(-DAY)),
        timeline: [
          {
            status: 'submitted',
            action: 'submitted',
            createdAt: iso(-2 * DAY),
          },
          {
            status: 'quote_pending',
            action: 'quoted',
            note: '包含肤色、碎发与背景细节处理',
            createdAt: iso(-DAY),
          },
        ],
      },
    ),
    ticket(
      'ticket_accepted',
      'YY20260728-G7H8J9',
      'user_anna',
      'task_anna_completed',
      'accepted',
      {
        quote: quote('quote_accepted', 3, iso(-3 * DAY), 'accepted'),
        reservedCredits: 3,
      },
    ),
    ticket(
      'ticket_processing',
      'YY20260727-K2L3M4',
      'user_mia',
      'task_mia_partial',
      'processing',
      {
        quote: quote('quote_processing', 5, iso(-4 * DAY), 'accepted'),
        reservedCredits: 5,
        spentCredits: 5,
      },
    ),
    ticket(
      'ticket_confirmation',
      'YY20260726-N5P6Q7',
      'user_anna',
      'task_anna_completed',
      'awaiting_confirmation',
      {
        quote: quote('quote_confirmation', 3, iso(-5 * DAY), 'accepted'),
        reservedCredits: 3,
        spentCredits: 3,
        deliverables: [
          {
            id: 'delivery_seed',
            url: '/demo/auth-studio.jpg',
            width: 1200,
            height: 1800,
          },
        ],
      },
    ),
    ticket(
      'ticket_delivered',
      'YY20260720-R8S9T1',
      'user_anna',
      'task_anna_completed',
      'delivered',
      {
        quote: quote('quote_delivered', 3, iso(-10 * DAY), 'accepted'),
        reservedCredits: 3,
        spentCredits: 3,
        deliverables: [
          {
            id: 'delivery_done',
            url: '/demo/auth-studio.jpg',
            width: 1200,
            height: 1800,
          },
        ],
      },
    ),
  ]

  return {
    admins: [
      {
        id: 'admin_platform',
        email: MOCK_ADMIN_ACCOUNTS.admin.email,
        password: MOCK_ADMIN_ACCOUNTS.admin.password,
        name: '映研管理员',
        role: 'platform_admin',
        permissions: ['platform:manage', 'retouch:manage'],
        status: 'active',
        createdAt: iso(-180 * DAY),
      },
      {
        id: 'admin_retouch',
        email: MOCK_ADMIN_ACCOUNTS.retouch.email,
        password: MOCK_ADMIN_ACCOUNTS.retouch.password,
        name: '修图操作员',
        role: 'retouch_operator',
        permissions: ['retouch:manage'],
        status: 'active',
        createdAt: iso(-90 * DAY),
      },
    ],
    batches: [
      {
        id: 'batch_xianyu_july',
        name: '咸鱼 7 月标准包',
        productCode: 'yingyan-client',
        quantity: 6,
        creditsPerCode: 10,
        expiresAt: iso(30 * DAY),
        neverExpires: false,
        note: '标准商品发码',
        createdBy: 'admin_platform',
        createdAt: iso(-14 * DAY),
      },
      {
        id: 'batch_vip',
        name: '长期客户 30 次包',
        productCode: 'yingyan-client',
        quantity: 2,
        creditsPerCode: 30,
        expiresAt: null,
        neverExpires: true,
        note: '长期客户专用',
        createdBy: 'admin_platform',
        createdAt: iso(-35 * DAY),
      },
    ],
    codes: [
      code('code_unused', 'YY-A2BC-D3EF-G4HK', 'batch_xianyu_july', iso(5 * DAY)),
      code('code_unused_2', 'YY-J5KM-N6PQ-R7ST', 'batch_xianyu_july', iso(30 * DAY)),
      code('code_redeemed', 'YY-U8VW-X9YZ-A2BC', 'batch_xianyu_july', iso(30 * DAY), {
        redeemedBy: 'user_anna',
        redeemedAt: iso(-8 * DAY),
        operationHistory: [
          {
            action: 'generated',
            operator: 'admin@yingyan.local',
            createdAt: iso(-14 * DAY),
          },
          {
            action: 'redeemed',
            operator: 'anna@example.com',
            createdAt: iso(-8 * DAY),
          },
        ],
      }),
      code('code_expired', 'YY-C3DE-F4GH-J5KM', 'batch_xianyu_july', iso(-DAY)),
      code('code_disabled', 'YY-N6PQ-R7ST-U8VW', 'batch_xianyu_july', iso(30 * DAY), {
        disabledBy: 'admin_platform',
        disabledAt: iso(-2 * DAY),
        disabledReason: '咸鱼订单已取消',
      }),
      code('code_unused_3', 'YY-X9YZ-A3CD-E4FG', 'batch_xianyu_july', iso(30 * DAY)),
      code('code_vip_unused', 'YY-H5JK-M6NP-Q7RS', 'batch_vip', null),
      code('code_vip_redeemed', 'YY-T8UV-W9XY-Z2AB', 'batch_vip', null, {
        redeemedBy: 'user_mia',
        redeemedAt: iso(-20 * DAY),
      }),
    ],
    providers: [
      {
        id: 'provider_test1',
        name: 'test1',
        code: 'test1',
        protocol: 'openai-compatible',
        baseUrl: 'https://api.test1.example/v1',
        apiKey: 'sk-test1-secret-key',
        maskedApiKey: 'sk-••••••••t-key',
        enabled: true,
        connectionStatus: 'healthy',
        lastTest: {
          testedAt: iso(-30 * 60_000),
          success: true,
          message: '认证与响应结构正常',
        },
        note: '主力服务商',
        createdAt: iso(-60 * DAY),
        updatedAt: iso(-30 * 60_000),
      },
      {
        id: 'provider_test2',
        name: 'test2',
        code: 'test2',
        protocol: 'openai-compatible',
        baseUrl: 'https://api.test2.example/v1',
        apiKey: 'sk-test2-secret-key',
        maskedApiKey: 'sk-••••••••t-key',
        enabled: true,
        connectionStatus: 'error',
        lastTest: {
          testedAt: iso(-2 * 60 * 60_000),
          success: false,
          message: '连接超时，请检查 Base URL',
        },
        note: '备用服务商，当前异常',
        createdAt: iso(-30 * DAY),
        updatedAt: iso(-2 * 60 * 60_000),
      },
    ],
    models: [
      {
        id: 'model_test1_chat',
        providerId: 'provider_test1',
        displayName: 'Prompt Studio Chat',
        modelId: 'prompt-studio-v2',
        type: 'chat',
        enabled: true,
        connectionStatus: 'healthy',
        capabilities: { promptOptimization: true, visionInput: true },
        lastTestAt: iso(-28 * 60_000),
        createdAt: iso(-58 * DAY),
        updatedAt: iso(-28 * 60_000),
      },
      {
        id: 'model_test1_image',
        providerId: 'provider_test1',
        displayName: 'Photon Studio Image',
        modelId: 'photon-image-v3',
        type: 'image',
        enabled: true,
        connectionStatus: 'healthy',
        capabilities: { textToImage: true, imageToImage: true },
        lastTestAt: iso(-25 * 60_000),
        createdAt: iso(-58 * DAY),
        updatedAt: iso(-25 * 60_000),
      },
      {
        id: 'model_test2_image',
        providerId: 'provider_test2',
        displayName: 'Vision Beta',
        modelId: 'vision-beta-1',
        type: 'image',
        enabled: true,
        connectionStatus: 'error',
        capabilities: { textToImage: true, imageToImage: false },
        lastTestAt: iso(-2 * 60 * 60_000),
        createdAt: iso(-25 * DAY),
        updatedAt: iso(-2 * 60 * 60_000),
      },
    ],
    bindings: {
      chatModelId: 'model_test1_chat',
      imageModelId: 'model_test1_image',
    },
    users,
    ledger: [
      {
        id: 'ledger_anna_redeem',
        userId: 'user_anna',
        type: 'redemption',
        amount: 10,
        balanceBefore: 8,
        balanceAfter: 18,
        description: `兑换 ${maskRedemptionCode('YY-U8VW-X9YZ-A2BC')}`,
        createdAt: iso(-8 * DAY),
      },
      {
        id: 'ledger_mia_reserve',
        userId: 'user_mia',
        type: 'reserve',
        amount: -2,
        balanceBefore: 8,
        balanceAfter: 6,
        description: '生成任务预占 2 次',
        createdAt: iso(-DAY),
      },
    ],
    assets,
    tasks,
    tickets,
    audits: [
      {
        id: 'audit_seed_retouch_quote',
        operatorId: 'admin_retouch',
        operatorEmail: 'retouch@yingyan.local',
        operatorRole: 'retouch_operator',
        action: 'retouch.quote',
        resourceType: 'retouch_ticket',
        resourceId: 'ticket_submitted',
        after: { status: 'quote_pending', credits: 4 },
        reason: '完成首轮人工修图报价',
        result: 'success',
        requestId: 'req_seed_retouch_quote',
        ip: '127.0.0.1',
        device: 'Chrome / macOS',
        createdAt: iso(-20 * 60_000),
      },
      {
        id: 'audit_seed_provider',
        operatorId: 'admin_platform',
        operatorEmail: 'admin@yingyan.local',
        operatorRole: 'platform_admin',
        action: 'provider.test',
        resourceType: 'ai_provider',
        resourceId: 'provider_test1',
        after: { connectionStatus: 'healthy' },
        result: 'success',
        requestId: 'req_seed_provider',
        ip: '127.0.0.1',
        device: 'Chrome / macOS',
        createdAt: iso(-30 * 60_000),
      },
      {
        id: 'audit_seed_code',
        operatorId: 'admin_platform',
        operatorEmail: 'admin@yingyan.local',
        operatorRole: 'platform_admin',
        action: 'redemption.disable',
        resourceType: 'redemption_code',
        resourceId: 'code_disabled',
        reason: '咸鱼订单已取消',
        result: 'success',
        requestId: 'req_seed_code',
        ip: '127.0.0.1',
        device: 'Chrome / macOS',
        createdAt: iso(-2 * DAY),
      },
    ],
    idempotency: {},
  }
}
