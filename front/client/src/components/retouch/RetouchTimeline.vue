<script setup lang="ts">
import { Check, Clock3 } from '@lucide/vue'
import { RETOUCH_TICKET_STATUS_LABELS } from '@/config'
import type { RetouchTicketTimelineEntry } from '@/types/domain'

defineProps<{
  entries: RetouchTicketTimelineEntry[]
}>()

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
</script>

<template>
  <section class="timeline-section" aria-labelledby="retouch-timeline-heading">
    <header>
      <div>
        <p>处理轨迹</p>
        <h3 id="retouch-timeline-heading">工单进度</h3>
      </div>
      <span>{{ entries.length }} 个节点</span>
    </header>

    <ol class="timeline">
      <li v-for="(entry, index) in entries" :key="`${entry.status}-${entry.createdAt}`">
        <span class="timeline-marker" :class="{ current: index === entries.length - 1 }">
          <Clock3 v-if="index === entries.length - 1" :size="13" />
          <Check v-else :size="13" />
        </span>
        <div>
          <div class="entry-heading">
            <strong>{{ RETOUCH_TICKET_STATUS_LABELS[entry.status] }}</strong>
            <time :datetime="entry.createdAt">{{ formatDate(entry.createdAt) }}</time>
          </div>
          <p v-if="entry.note">{{ entry.note }}</p>
        </div>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.timeline-section {
  padding: 24px 0;
  border-top: 1px solid var(--border);
}

header {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

header p {
  color: var(--primary);
  font-size: 9px;
  font-weight: 800;
}

header h3 {
  margin-top: 2px;
  font-size: 16px;
}

header > span {
  color: var(--ink-faint);
  font-size: 10px;
}

.timeline {
  padding: 0;
  margin: 0;
  list-style: none;
}

.timeline li {
  position: relative;
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 10px;
  padding-bottom: 22px;
}

.timeline li:last-child {
  padding-bottom: 0;
}

.timeline li:not(:last-child)::before {
  position: absolute;
  top: 24px;
  bottom: 2px;
  left: 13px;
  width: 1px;
  background: var(--border);
  content: '';
}

.timeline-marker {
  position: relative;
  z-index: 1;
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: var(--surface);
  color: var(--ink-faint);
}

.timeline-marker.current {
  border-color: var(--primary);
  background: var(--primary-soft);
  color: var(--primary);
}

.entry-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 14px;
  padding-top: 3px;
}

.entry-heading strong {
  font-size: 12px;
}

.entry-heading time {
  color: var(--ink-faint);
  font-size: 9px;
  white-space: nowrap;
}

.timeline li p {
  margin-top: 4px;
  color: var(--ink-muted);
  font-size: 11px;
  line-height: 1.6;
}
</style>
