import { http } from 'msw'
import {
  publicBatch,
  publicModel,
  publicTicketSummary,
  readDb,
} from '@/mocks/db'
import { requireAdmin, respond } from '@/mocks/helpers'
import type {
  DashboardAlert,
  DashboardData,
  DashboardMetric,
} from '@/types/domain'
import {
  deriveRedemptionStatus,
  isExpiringSoon,
} from '@/utils/redemption'

export const dashboardHandlers = [
  http.get('/api/manage/dashboard', () =>
    respond(() => {
      const db = readDb()
      const admin = requireAdmin(db, 'retouch:manage')
      const pendingTickets = db.tickets
        .filter(
          (item) =>
            !['delivered', 'rejected', 'cancelled'].includes(item.status),
        )
        .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
        .map((item) => publicTicketSummary(item, db))
      const overdueTickets = pendingTickets.filter((item) => item.sla.overdue)
      const dueSoonTickets = pendingTickets.filter(
        (item) => !item.sla.overdue && item.sla.remainingSeconds !== null && item.sla.remainingSeconds <= 24 * 60 * 60,
      )

      if (admin.role === 'retouch_operator') {
        return {
          metrics: [
            {
              key: 'pendingTickets',
              label: '待处理人工工单',
              value: pendingTickets.length,
              tone: pendingTickets.length ? 'warning' : 'positive',
            },
            {
              key: 'overdueTickets',
              label: '已逾期工单',
              value: overdueTickets.length,
              tone: overdueTickets.length ? 'danger' : 'positive',
            },
          ],
          currentModels: {},
          alerts: [],
          pendingTickets: pendingTickets.slice(0, 6),
          recentBatches: [],
        } satisfies DashboardData
      }

      const now = new Date()
      const todayStart = new Date(
        now.getFullYear(),
        now.getMonth(),
        now.getDate(),
      ).getTime()
      const redeemedToday = db.codes.filter(
        (item) =>
          item.redeemedAt &&
          new Date(item.redeemedAt).getTime() >= todayStart,
      )
      const unusedCodes = db.codes.filter(
        (item) => deriveRedemptionStatus(item) === 'unused',
      )
      const failedTasks = db.tasks.filter((item) => item.status === 'failed')
      const metrics: DashboardMetric[] = [
        {
          key: 'unusedCodes',
          label: '可发放兑换码',
          value: unusedCodes.length,
          tone: unusedCodes.length < 5 ? 'warning' : 'neutral',
        },
        {
          key: 'expiringCodes',
          label: '7 天内过期',
          value: db.codes.filter((item) => isExpiringSoon(item)).length,
          tone: 'warning',
        },
        {
          key: 'redeemedToday',
          label: '今日兑换',
          value: redeemedToday.length,
          tone: 'positive',
        },
        {
          key: 'creditsGrantedToday',
          label: '今日发放次数',
          value: redeemedToday.reduce((sum, item) => sum + item.credits, 0),
          tone: 'neutral',
        },
        {
          key: 'failedTasks',
          label: '失败任务',
          value: failedTasks.length,
          tone: failedTasks.length ? 'danger' : 'positive',
        },
          {
            key: 'pendingTickets',
          label: '待处理工单',
          value: pendingTickets.length,
            tone: pendingTickets.length ? 'warning' : 'positive',
          },
          {
            key: 'overdueTickets',
            label: '已逾期工单',
            value: overdueTickets.length,
            tone: overdueTickets.length ? 'danger' : 'positive',
          },
          {
            key: 'dueSoonTickets',
            label: '即将逾期工单',
            value: dueSoonTickets.length,
            tone: dueSoonTickets.length ? 'warning' : 'positive',
          },
      ]

      const alerts: DashboardAlert[] = db.providers
        .filter(
          (item) => item.enabled && item.connectionStatus === 'error',
        )
        .map((item) => ({
          id: `alert_provider_${item.id}`,
          type: 'provider',
          title: `${item.name} 连接异常`,
          description: item.lastTest?.message ?? '请重新测试服务连接',
          tone: 'danger',
          href: '/manage/ai-providers',
        }))
      if (failedTasks.length) {
        alerts.push({
          id: 'alert_failed_tasks',
          type: 'task',
          title: `${failedTasks.length} 个生成任务失败`,
          description: '已按任务结算规则退款，请检查服务商状态',
          tone: 'warning',
          href: '/manage/generation-tasks?status=failed',
        })
      }

      const chatModel = db.models.find(
        (item) => item.id === db.bindings.chatModelId,
      )
      const imageModel = db.models.find(
        (item) => item.id === db.bindings.imageModelId,
      )
      const publicChatModel = chatModel
        ? publicModel(chatModel, db)
        : undefined
      const publicImageModel = imageModel
        ? publicModel(imageModel, db)
        : undefined

      return {
        metrics,
        currentModels: {
          chat: publicChatModel
            ? {
                id: publicChatModel.id,
                displayName: publicChatModel.displayName,
                modelId: publicChatModel.modelId,
                providerName: publicChatModel.providerName,
                type: publicChatModel.type,
              }
            : undefined,
          image: publicImageModel
            ? {
                id: publicImageModel.id,
                displayName: publicImageModel.displayName,
                modelId: publicImageModel.modelId,
                providerName: publicImageModel.providerName,
                type: publicImageModel.type,
              }
            : undefined,
        },
        alerts,
        pendingTickets: pendingTickets.slice(0, 6),
        recentBatches: db.batches
          .slice()
          .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
          .slice(0, 4)
          .map((item) => publicBatch(item, db)),
      } satisfies DashboardData
    }),
  ),
]
