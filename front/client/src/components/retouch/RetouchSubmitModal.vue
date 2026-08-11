<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import {
  Check,
  FileImage,
  ImagePlus,
  ShieldCheck,
  Trash2,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import { ASSET_CONFIG, RETOUCH_TICKET_CONFIG } from '@/config'
import type { GenerationTask } from '@/types/domain'

interface SupplementalPreview {
  file: File
  url: string
}

const props = defineProps<{
  open: boolean
  task: GenerationTask | null
  loading: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [
    selectedResultIds: string[],
    requirement: string,
    supplementalFiles: File[],
  ]
}>()

const fileInput = ref<HTMLInputElement | null>(null)
const selectedResultIds = ref<string[]>([])
const requirement = ref('')
const supplementalPreviews = ref<SupplementalPreview[]>([])
const validationError = ref('')

const canSubmit = computed(
  () =>
    selectedResultIds.value.length > 0 &&
    requirement.value.trim().length > 0 &&
    !props.loading,
)

function releasePreviews(): void {
  supplementalPreviews.value.forEach((item) => URL.revokeObjectURL(item.url))
  supplementalPreviews.value = []
}

function resetForm(): void {
  releasePreviews()
  selectedResultIds.value = props.task?.results[0]
    ? [props.task.results[0].id]
    : []
  requirement.value = ''
  validationError.value = ''
}

watch(
  () => [props.open, props.task?.id] as const,
  ([open]) => {
    if (open) resetForm()
  },
)

onBeforeUnmount(releasePreviews)

function toggleResult(resultId: string): void {
  validationError.value = ''
  if (selectedResultIds.value.includes(resultId)) {
    selectedResultIds.value = selectedResultIds.value.filter(
      (item) => item !== resultId,
    )
    return
  }
  if (
    selectedResultIds.value.length >=
    RETOUCH_TICKET_CONFIG.maxSelectedResults
  ) {
    validationError.value = `最多选择 ${RETOUCH_TICKET_CONFIG.maxSelectedResults} 张成片`
    return
  }
  selectedResultIds.value = [...selectedResultIds.value, resultId]
}

function handleFiles(event: Event): void {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  validationError.value = ''

  const remaining =
    RETOUCH_TICKET_CONFIG.maxSupplementalAssets -
    supplementalPreviews.value.length
  const accepted = files
    .filter(
      (file) =>
        ASSET_CONFIG.acceptedMimeTypes.some((type) => type === file.type) &&
        file.size <= ASSET_CONFIG.maxFileSize,
    )
    .slice(0, Math.max(0, remaining))

  if (!accepted.length && files.length) {
    validationError.value = `补充参考图需为 JPG、PNG 或 WebP，单张不超过 ${ASSET_CONFIG.maxFileSizeLabel}`
    return
  }

  supplementalPreviews.value.push(
    ...accepted.map((file) => ({
      file,
      url: URL.createObjectURL(file),
    })),
  )
  if (accepted.length < files.length) {
    validationError.value = `最多补充 ${RETOUCH_TICKET_CONFIG.maxSupplementalAssets} 张合规参考图`
  }
}

function removeSupplemental(index: number): void {
  const [removed] = supplementalPreviews.value.splice(index, 1)
  if (removed) URL.revokeObjectURL(removed.url)
  validationError.value = ''
}

function submit(): void {
  if (!selectedResultIds.value.length) {
    validationError.value = '请至少选择一张需要人工精修的 AI 成片'
    return
  }
  const normalizedRequirement = requirement.value.trim()
  if (!normalizedRequirement) {
    validationError.value = '请填写具体的人工修图要求'
    return
  }
  emit(
    'submit',
    selectedResultIds.value,
    normalizedRequirement,
    supplementalPreviews.value.map((item) => item.file),
  )
}
</script>

<template>
  <BaseModal
    :open="open"
    size="wide"
    title="申请人工精修"
    description="从本次 AI 成片中选择需要继续处理的图片。"
    @close="$emit('close')"
  >
    <form class="retouch-form" @submit.prevent="submit">
      <section class="form-section">
        <div class="section-heading">
          <div>
            <span>01 · 选择成片</span>
            <h3>哪些图片需要人工处理</h3>
          </div>
          <b>
            {{ selectedResultIds.length }}/{{
              RETOUCH_TICKET_CONFIG.maxSelectedResults
            }}
          </b>
        </div>

        <div class="result-grid">
          <button
            v-for="(result, index) in task?.results ?? []"
            :key="result.id"
            class="result-option"
            :class="{ selected: selectedResultIds.includes(result.id) }"
            type="button"
            :aria-pressed="selectedResultIds.includes(result.id)"
            :aria-label="`选择第 ${index + 1} 张成片`"
            @click="toggleResult(result.id)"
          >
            <img :src="result.url" :alt="`AI 成片 ${index + 1}`" />
            <span class="result-index">成片 {{ index + 1 }}</span>
            <span class="selection-mark">
              <Check :size="15" />
            </span>
          </button>
        </div>
      </section>

      <section class="form-section">
        <div class="section-heading">
          <div>
            <span>02 · 修图要求</span>
            <h3>告诉修图师要改什么</h3>
          </div>
          <b>
            {{ requirement.length }}/{{
              RETOUCH_TICKET_CONFIG.requirementMaxLength
            }}
          </b>
        </div>
        <textarea
          v-model="requirement"
          :maxlength="RETOUCH_TICKET_CONFIG.requirementMaxLength"
          rows="5"
          placeholder="例如：保留人物五官和真实皮肤纹理，调整右侧碎发，减轻眼下阴影，并让衣服褶皱更自然。"
          aria-label="人工修图要求"
          @input="validationError = ''"
        />
      </section>

      <section class="form-section">
        <div class="section-heading">
          <div>
            <span>03 · 补充参考</span>
            <h3>可选的效果或细节参考图</h3>
          </div>
          <b>
            {{ supplementalPreviews.length }}/{{
              RETOUCH_TICKET_CONFIG.maxSupplementalAssets
            }}
          </b>
        </div>

        <input
          ref="fileInput"
          class="sr-only"
          type="file"
          multiple
          :accept="ASSET_CONFIG.acceptAttribute"
          @change="handleFiles"
        />
        <div class="supplemental-grid">
          <button
            v-if="
              supplementalPreviews.length <
                RETOUCH_TICKET_CONFIG.maxSupplementalAssets
            "
            class="upload-option"
            type="button"
            @click="fileInput?.click()"
          >
            <ImagePlus :size="22" />
            <strong>添加参考图</strong>
            <span>不超过 {{ ASSET_CONFIG.maxFileSizeLabel }}/张</span>
          </button>

          <article
            v-for="(item, index) in supplementalPreviews"
            :key="`${item.file.name}:${item.file.lastModified}`"
            class="supplemental-item"
          >
            <img :src="item.url" :alt="item.file.name" />
            <div>
              <FileImage :size="14" />
              <span :title="item.file.name">{{ item.file.name }}</span>
              <button
                type="button"
                :aria-label="`移除 ${item.file.name}`"
                title="移除参考图"
                @click="removeSupplemental(index)"
              >
                <Trash2 :size="15" />
              </button>
            </div>
          </article>
        </div>
      </section>

      <div class="association-note">
        <ShieldCheck :size="19" />
        <p>
          原任务的原图、参考图、提示词与生成参数会自动关联到工单，无需重复上传。
        </p>
      </div>

      <p v-if="validationError" class="form-error" role="alert">
        {{ validationError }}
      </p>

      <footer>
        <BaseButton
          type="button"
          variant="secondary"
          :disabled="loading"
          @click="$emit('close')"
        >
          暂不申请
        </BaseButton>
        <BaseButton type="submit" :loading="loading" :disabled="!canSubmit">
          提交人工精修需求
        </BaseButton>
      </footer>
    </form>
  </BaseModal>
</template>

<style scoped src="./RetouchSubmitModal.css"></style>
