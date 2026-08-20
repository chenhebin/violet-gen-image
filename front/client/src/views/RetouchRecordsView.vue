<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import RetouchDetailDrawer from '@/components/retouch/RetouchDetailDrawer.vue'
import RetouchTicketList from '@/components/retouch/RetouchTicketList.vue'
import { useToast } from '@/composables/useToast'
import {
  isFinalRetouchTicketStatus,
  RETOUCH_TICKET_TIMING,
} from '@/config'
import { useRetouchStore } from '@/stores/retouch'

const route = useRoute()
const router = useRouter()
const retouch = useRetouchStore()
const toast = useToast()
let listTimer: number | null = null
let detailTimer: number | null = null

const ticketId = computed(() =>
  typeof route.params.ticketId === 'string' ? route.params.ticketId : '',
)
const attentionCount = computed(
  () =>
    retouch.tickets.filter((ticket) =>
      ['quote_pending', 'awaiting_confirmation'].includes(ticket.status),
    ).length,
)

onMounted(async () => {
  await loadRecords()
  listTimer = window.setInterval(() => {
    if (
      retouch.hasActiveTickets &&
      !retouch.loading &&
      !retouch.actionLoading
    ) {
      void loadRecords(false)
    }
  }, RETOUCH_TICKET_TIMING.listPollMs)
  detailTimer = window.setInterval(() => {
    const activeTicket = retouch.activeTicket
    if (
      ticketId.value &&
      activeTicket?.id === ticketId.value &&
      !isFinalRetouchTicketStatus(activeTicket.status) &&
      !retouch.loading &&
      !retouch.actionLoading
    ) {
      void openTicket(ticketId.value, false)
    }
  }, RETOUCH_TICKET_TIMING.detailPollMs)
})

onBeforeUnmount(() => {
  if (listTimer !== null) window.clearInterval(listTimer)
  if (detailTimer !== null) window.clearInterval(detailTimer)
  retouch.close()
})

watch(
  ticketId,
  async (id) => {
    if (!id) {
      retouch.close()
      return
    }
    await openTicket(id)
  },
  { immediate: true },
)

async function loadRecords(reportError = true): Promise<void> {
  try {
    await retouch.load({ silent: !reportError })
  } catch (caught) {
    if (reportError) showError('精修记录加载失败', caught)
  }
}

async function changeTicketPage(nextPage: number): Promise<void> {
  if (nextPage < 1 || (nextPage > retouch.page && !retouch.hasMore)) return
  await retouch.load({ page: nextPage })
}

async function openTicket(id: string, reportError = true): Promise<void> {
  try {
    await retouch.open(id)
  } catch (caught) {
    if (!reportError) return
    showError('工单无法打开', caught)
    await router.replace('/app/retouch')
  }
}

async function closeDetail(): Promise<void> {
  retouch.close()
  await router.push('/app/retouch')
}

async function acceptQuote(id: string, quoteId: string): Promise<void> {
  try {
    await retouch.acceptQuote(id, quoteId)
    toast.success('报价已接受', '额度已预占，精修师将开始处理')
  } catch (caught) {
    showError('无法接受报价', caught)
  }
}

async function cancelTicket(id: string): Promise<void> {
  try {
    const ticket = await retouch.cancel(id)
    toast.success(
      '工单已取消',
      ticket?.refundedCredits
        ? `已退回 ${ticket.refundedCredits} 次额度`
        : '工单已停止处理',
    )
  } catch (caught) {
    showError('无法取消工单', caught)
  }
}

async function confirmDelivery(id: string): Promise<void> {
  try {
    await retouch.confirm(id)
    toast.success('交付已确认', '精修工单已完成')
  } catch (caught) {
    showError('无法确认交付', caught)
  }
}

async function requestRevision(id: string, message: string): Promise<void> {
  try {
    await retouch.requestRevision(id, message)
    toast.success('返修要求已提交', '精修师将按新要求处理')
  } catch (caught) {
    showError('无法提交返修要求', caught)
  }
}

function showError(title: string, caught: unknown): void {
  toast.error(
    title,
    caught instanceof Error ? caught.message : '工单状态可能已经变化，请稍后重试',
  )
}
</script>

<template>
  <div class="retouch-page">
    <header class="page-heading">
      <div>
        <span>人工服务</span>
        <h1>人工修图记录</h1>
        <p>跟进报价、人工处理和成片验收，交付文件可随时下载。</p>
      </div>
      <div v-if="attentionCount" class="attention-note">
        <strong>{{ attentionCount }}</strong>
        <span>个工单等待你处理</span>
      </div>
    </header>

    <RetouchTicketList
      :tickets="retouch.tickets"
      :loading="retouch.loading"
      @open="router.push(`/app/retouch/${$event}`)"
    />

    <RetouchDetailDrawer
      :open="Boolean(ticketId)"
      :ticket="retouch.activeTicket"
      :loading="retouch.loading"
      :submitting="retouch.actionLoading"
      @close="closeDetail"
      @accept-quote="acceptQuote"
      @cancel="cancelTicket"
      @confirm="confirmDelivery"
      @request-revision="requestRevision"
    />

    <nav v-if="retouch.total > retouch.pageSize" class="pagination" aria-label="人工修图记录分页">
      <button type="button" :disabled="retouch.page <= 1 || retouch.loading" @click="changeTicketPage(retouch.page - 1)">上一页</button>
      <span>第 {{ retouch.page }} / {{ Math.max(1, Math.ceil(retouch.total / retouch.pageSize)) }} 页</span>
      <button type="button" :disabled="!retouch.hasMore || retouch.loading" @click="changeTicketPage(retouch.page + 1)">下一页</button>
    </nav>
  </div>
</template>

<style scoped>
.retouch-page {
  width: min(1180px, calc(100% - 40px));
  padding: 36px 0 60px;
  margin: 0 auto;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
  margin: 18px 0;
  color: var(--muted);
  font-size: 13px;
}

.pagination button {
  min-height: 40px;
  padding: 0 14px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--surface);
  color: var(--ink);
}

.pagination button:disabled {
  cursor: not-allowed;
  opacity: .45;
}

.page-heading {
  display: flex;
  min-height: 100px;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 26px;
}

.page-heading > div:first-child > span {
  color: var(--primary);
  font-size: 10px;
  font-weight: 800;
}

.page-heading h1 {
  margin-top: 3px;
  font-family: 'Songti SC', 'STSong', serif;
  font-size: 32px;
  font-weight: 600;
}

.page-heading p {
  margin-top: 5px;
  color: var(--ink-muted);
  font-size: 13px;
}

.attention-note {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 10px 12px;
  border-left: 3px solid var(--warning);
  background: #fffaf0;
}

.attention-note strong {
  color: var(--warning);
  font-family: 'Songti SC', serif;
  font-size: 24px;
  font-weight: 600;
}

.attention-note span {
  color: var(--ink-muted);
  font-size: 11px;
}

@media (max-width: 760px) {
  .retouch-page {
    width: calc(100% - 24px);
    padding: 18px 0 28px;
  }

  .page-heading {
    min-height: 0;
    align-items: stretch;
    flex-direction: column;
    gap: 14px;
    margin-bottom: 18px;
  }

  .page-heading h1 {
    font-size: 28px;
  }

  .page-heading p {
    max-width: 320px;
    line-height: 1.65;
  }

  .attention-note {
    align-self: flex-start;
  }
}
</style>
