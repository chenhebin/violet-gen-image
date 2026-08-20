import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import {
  ASSET_CONFIG,
  DEFAULT_GENERATION_SETTINGS,
  PROMPT_CONFIG,
  PROMPT_SECTION_OPTIONS,
} from '@/config'
import { assetApi, generationApi, promptApi } from '@/services/api'
import { createIdempotencyKey } from '@/services/http'
import { AppError, ErrorCode } from '@/types/api'
import type {
  Asset,
  AssetKind,
  GenerationCreateResult,
  GenerationSettings,
  GenerationTask,
  PromptSectionBackups,
  PromptSections,
  ReferenceRole,
  WorkspaceDraft,
  WorkspaceMode,
} from '@/types/domain'

const DRAFT_KEY = 'yingyan:workspace-drafts:v1'
const PROMPT_PROTECTION_MIGRATION_KEY =
  'yingyan:workspace-prompt-protection:v1'
const SIMPLE_WORKFLOW_MIGRATION_KEY =
  'yingyan:workspace-simple-workflow:v1'

function defaultSettings(): GenerationSettings {
  return { ...DEFAULT_GENERATION_SETTINGS }
}

function defaultDraft(mode: WorkspaceMode): WorkspaceDraft {
  return {
    mode,
    sourcePrompt: '',
    assets: [],
    referencePrompt: '',
    promptVersion: null,
    promptSectionBackups: {},
    settings: defaultSettings(),
  }
}

function applyDefaultPromptProtection(target: WorkspaceDraft): void {
  target.promptSectionBackups = {}
  if (!target.promptVersion) return

  const backups: PromptSectionBackups = {}
  for (const { key } of PROMPT_SECTION_OPTIONS) {
    backups[key] = target.promptVersion.sections[key]
    target.promptVersion.sections[key] = PROMPT_CONFIG.unchangedText
  }
  target.promptSectionBackups = backups
}

function loadDrafts(): Record<WorkspaceMode, WorkspaceDraft> {
  const defaults = {
    'text-to-image': defaultDraft('text-to-image'),
    'image-to-image': defaultDraft('image-to-image'),
  }
  try {
    const raw = localStorage.getItem(DRAFT_KEY)
    if (!raw) {
      localStorage.setItem(PROMPT_PROTECTION_MIGRATION_KEY, '1')
      localStorage.setItem(SIMPLE_WORKFLOW_MIGRATION_KEY, '1')
      return defaults
    }
    const saved = JSON.parse(raw) as Partial<typeof defaults>
    const loaded = {
      'text-to-image': {
        ...defaults['text-to-image'],
        ...saved['text-to-image'],
        referencePrompt: saved['text-to-image']?.referencePrompt ?? '',
        settings: {
          ...defaults['text-to-image'].settings,
          ...saved['text-to-image']?.settings,
        },
        promptSectionBackups: {
          ...saved['text-to-image']?.promptSectionBackups,
        },
      },
      'image-to-image': {
        ...defaults['image-to-image'],
        ...saved['image-to-image'],
        referencePrompt: saved['image-to-image']?.referencePrompt ?? '',
        settings: {
          ...defaults['image-to-image'].settings,
          ...saved['image-to-image']?.settings,
        },
        promptSectionBackups: {
          ...saved['image-to-image']?.promptSectionBackups,
        },
      },
    }
    if (!localStorage.getItem(PROMPT_PROTECTION_MIGRATION_KEY)) {
      applyDefaultPromptProtection(loaded['text-to-image'])
      applyDefaultPromptProtection(loaded['image-to-image'])
      localStorage.setItem(PROMPT_PROTECTION_MIGRATION_KEY, '1')
    }
    if (!localStorage.getItem(SIMPLE_WORKFLOW_MIGRATION_KEY)) {
      loaded['text-to-image'].settings.outputCount = 1
      loaded['image-to-image'].settings.outputCount = 1
      localStorage.setItem(SIMPLE_WORKFLOW_MIGRATION_KEY, '1')
    }
    return loaded
  } catch {
    return defaults
  }
}

export const useWorkspaceStore = defineStore('workspace', () => {
  const mode = ref<WorkspaceMode>('image-to-image')
  const drafts = ref(loadDrafts())
  const optimizing = ref(false)
  const confirming = ref(false)
  const submitting = ref(false)
  const uploadCount = ref(0)
  const error = ref('')
  const currentTask = ref<GenerationTask | null>(null)
  let optimizeController: AbortController | null = null
  let generationIdempotencyKey: string | null = null
  let draftRevision = 0
  const uploadControllers = new Map<string, AbortController>()

  const draft = computed(() => drafts.value[mode.value])
  const requestMode = computed<WorkspaceMode>(() =>
    draft.value.assets.length > 0 ? 'image-to-image' : 'text-to-image',
  )
  const sourceAssets = computed(() =>
    draft.value.assets.filter((asset) => asset.kind === 'source'),
  )
  const referenceAssets = computed(() =>
    draft.value.assets.filter((asset) => asset.kind === 'reference'),
  )
  const isConfirmed = computed(() =>
    Boolean(draft.value.promptVersion?.confirmedAt),
  )
  const hasReferenceAssets = computed(() => referenceAssets.value.length > 0)
  const referenceOptimizationRequired = computed(
    () =>
      hasReferenceAssets.value &&
      (sourceAssets.value.length === 0 ||
        !draft.value.referencePrompt ||
        !draft.value.promptVersion),
  )
  const isUploading = computed(() => uploadCount.value > 0)
  const canOptimize = computed(
    () =>
      !isUploading.value &&
      draft.value.sourcePrompt.trim().length >= PROMPT_CONFIG.minLength,
  )
  const canSubmit = computed(() => {
    const length = draft.value.sourcePrompt.trim().length
    return (
      !isUploading.value &&
      length >= PROMPT_CONFIG.minLength &&
      length <= PROMPT_CONFIG.maxLength
    )
  })
  const canGenerate = computed(
    () =>
      canSubmit.value &&
      !referenceOptimizationRequired.value &&
      (!hasReferenceAssets.value || isConfirmed.value),
  )
  const promptNeedsConfirmation = computed(
    () => hasReferenceAssets.value && Boolean(draft.value.promptVersion) && !isConfirmed.value,
  )

  watch(
    drafts,
    (value) => {
      const serializable = Object.fromEntries(
        Object.entries(value).map(([key, item]) => [
          key,
          {
            ...item,
            settings: { ...item.settings },
            promptVersion: item.promptVersion
              ? {
                  ...item.promptVersion,
                  sections: { ...item.promptVersion.sections },
                }
              : null,
            assets: item.assets.map((asset) => ({
              ...asset,
              previewUrl: asset.previewUrl?.startsWith('/')
                ? asset.previewUrl
                : undefined,
            })),
          },
        ]),
      )
      localStorage.setItem(DRAFT_KEY, JSON.stringify(serializable))
    },
    { deep: true },
  )

  function invalidateConfirmation(): void {
    generationIdempotencyKey = null
    if (!draft.value.promptVersion?.confirmedAt) return
    draft.value.promptVersion = {
      ...draft.value.promptVersion,
      confirmedAt: undefined,
    }
  }

  function cancelStaleOptimization(): void {
    draftRevision += 1
    optimizeController?.abort()
    optimizeController = null
    optimizing.value = false
  }

  function invalidateAssetSnapshot(): void {
    cancelStaleOptimization()
    generationIdempotencyKey = null
    draft.value.referencePrompt = ''
    draft.value.promptVersion = null
    draft.value.promptSectionBackups = {}
  }

  function setMode(nextMode: WorkspaceMode): void {
    cancelStaleOptimization()
    generationIdempotencyKey = null
    mode.value = nextMode
    currentTask.value = null
    error.value = ''
  }

  function setSourcePrompt(value: string): void {
    if (draft.value.sourcePrompt === value) return
    cancelStaleOptimization()
    generationIdempotencyKey = null
    draft.value.sourcePrompt = value
    if (draft.value.promptVersion) {
      draft.value.promptVersion = null
      draft.value.promptSectionBackups = {}
    }
  }

  function updatePromptSection(
    key: keyof PromptSections,
    value: string,
  ): void {
    if (!draft.value.promptVersion) return
    if (isPromptSectionUnchanged(key)) return
    draft.value.promptVersion.sections[key] = value
    invalidateConfirmation()
  }

  function updateReferencePrompt(value: string): void {
    if (!draft.value.promptVersion) return
    draft.value.referencePrompt = value
    draft.value.promptVersion.sections.referencePrompt = value
    invalidateConfirmation()
  }

  function isPromptSectionUnchanged(key: keyof PromptSections): boolean {
    return Object.prototype.hasOwnProperty.call(
      draft.value.promptSectionBackups,
      key,
    )
  }

  function setPromptSectionUnchanged(
    key: keyof PromptSections,
    unchanged: boolean,
  ): void {
    const version = draft.value.promptVersion
    if (!version || unchanged === isPromptSectionUnchanged(key)) return

    if (unchanged) {
      draft.value.promptSectionBackups[key] = version.sections[key]
      version.sections[key] = PROMPT_CONFIG.unchangedText
    } else {
      version.sections[key] = draft.value.promptSectionBackups[key] ?? ''
      delete draft.value.promptSectionBackups[key]
    }
    invalidateConfirmation()
  }

  function updateSettings(patch: Partial<GenerationSettings>): void {
    draft.value.settings = { ...draft.value.settings, ...patch }
    invalidateConfirmation()
  }

  async function hydrateAssets(): Promise<void> {
    for (const key of ['text-to-image', 'image-to-image'] as const) {
      drafts.value[key].assets = await Promise.all(
        drafts.value[key].assets.map((asset) => assetApi.hydrate(asset)),
      )
    }
  }

  async function uploadFiles(
    files: File[],
    kind: AssetKind,
    role?: ReferenceRole,
  ): Promise<void> {
    const accepted = files
      .filter(
        (file) =>
          ASSET_CONFIG.acceptedMimeTypes.some((type) => type === file.type) &&
          file.size <= ASSET_CONFIG.maxFileSize,
      )
      .slice(
        0,
        Math.max(0, ASSET_CONFIG.maxCount - draft.value.assets.length),
      )
    if (!accepted.length) {
      error.value = `请选择不超过 ${ASSET_CONFIG.maxFileSizeLabel} 的 JPG、PNG 或 WebP 图片`
      return
    }

    error.value = ''
    invalidateAssetSnapshot()
    const targetMode = mode.value
    await Promise.all(
      accepted.map(async (file) => {
        const localId = `${file.name}:${file.lastModified}`
        const controller = new AbortController()
        uploadControllers.set(localId, controller)
        uploadCount.value += 1
        try {
          const asset = await assetApi.upload(
            file,
            kind,
            role,
            undefined,
            controller.signal,
          )
          drafts.value[targetMode].assets.push(asset)
          if (targetMode === mode.value) generationIdempotencyKey = null
        } catch (caught) {
          error.value = caught instanceof Error ? caught.message : '图片上传失败'
        } finally {
          uploadControllers.delete(localId)
          uploadCount.value -= 1
        }
      }),
    )
  }

  async function removeAsset(asset: Asset): Promise<void> {
    draft.value.assets = draft.value.assets.filter((item) => item.id !== asset.id)
    invalidateAssetSnapshot()
    try {
      await assetApi.remove(asset.id)
    } catch (caught) {
      if (
        !(caught instanceof AppError && caught.code === ErrorCode.InvalidPayload)
      ) {
        error.value =
          caught instanceof Error ? caught.message : '素材记录清理失败'
      }
    }
  }

  function setReferenceRole(assetId: string, role: ReferenceRole): void {
    const asset = draft.value.assets.find((item) => item.id === assetId)
    if (!asset) return
    asset.role = role
    invalidateAssetSnapshot()
  }

  async function optimizePrompt(): Promise<void> {
    optimizeController?.abort()
    const controller = new AbortController()
    optimizeController = controller
    const revision = draftRevision
    const targetDraft = draft.value
    optimizing.value = true
    error.value = ''
    try {
      const optimized = await promptApi.optimize(
        {
          source: targetDraft.sourcePrompt,
          mode: requestMode.value,
          sourceAssetIds: sourceAssets.value.map((asset) => asset.id),
          referenceAssets: referenceAssets.value.map((asset) => ({
            assetId: asset.id,
            role: asset.role ?? 'style',
          })),
          referencePrompt: targetDraft.referencePrompt || undefined,
        },
        controller.signal,
      )
      if (revision !== draftRevision || draft.value !== targetDraft) return
      targetDraft.promptVersion = {
        ...optimized,
        sections: {
          ...optimized.sections,
          referencePrompt:
            targetDraft.referencePrompt || optimized.sections.referencePrompt,
        },
      }

      // Reference-guided optimization needs an explicit review of every
      // generated section. The source image is the identity anchor, while
      // the reference image only contributes style and atmosphere.
      const requiresReferenceReview =
        sourceAssets.value.length > 0 &&
        referenceAssets.value.length > 0 &&
        Boolean(targetDraft.referencePrompt?.trim())
      if (requiresReferenceReview) {
        targetDraft.promptSectionBackups = {}
      } else {
        applyDefaultPromptProtection(targetDraft)
      }
    } catch (caught) {
      if (
        typeof caught === 'object' &&
        caught !== null &&
        'code' in caught &&
        caught.code === 'ERR_CANCELED'
      ) {
        return
      }
      error.value = caught instanceof Error ? caught.message : '提示词优化失败'
      throw caught
    } finally {
      if (optimizeController === controller) {
        optimizeController = null
        optimizing.value = false
      }
    }
  }

  async function describeReferencePrompt(): Promise<void> {
    if (!referenceAssets.value.length) return
    if (!sourceAssets.value.length) {
      throw new Error('请先上传待修改原图，再分析参考图')
    }
    optimizeController?.abort()
    const controller = new AbortController()
    optimizeController = controller
    const revision = draftRevision
    const targetDraft = draft.value
    optimizing.value = true
    error.value = ''
    try {
      const result = await promptApi.describeReferences(
        referenceAssets.value.map((asset) => ({
          assetId: asset.id,
          role: asset.role ?? 'style',
        })),
        controller.signal,
      )
      if (revision !== draftRevision || draft.value !== targetDraft) return
      targetDraft.referencePrompt = result.prompt
    } catch (caught) {
      if (
        typeof caught === 'object' &&
        caught !== null &&
        'code' in caught &&
        caught.code === 'ERR_CANCELED'
      ) {
        return
      }
      error.value = caught instanceof Error ? caught.message : '参考图分析失败'
      throw caught
    } finally {
      if (optimizeController === controller) {
        optimizeController = null
        optimizing.value = false
      }
    }
  }

  async function preparePrompt(): Promise<void> {
    if (referenceAssets.value.length > 0) {
      if (!sourceAssets.value.length) {
        throw new Error('请先上传待修改原图')
      }
      if (!draft.value.referencePrompt) {
        await describeReferencePrompt()
      }
    }
    await optimizePrompt()
  }

  async function confirmPrompt(): Promise<void> {
    const version = draft.value.promptVersion
    if (!version) return
    confirming.value = true
    error.value = ''
    try {
      draft.value.promptVersion = await promptApi.confirm(
        version.id,
        draft.value.sourcePrompt,
        version.sections,
      )
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : '提示词确认失败'
      throw caught
    } finally {
      confirming.value = false
    }
  }

  async function submit(): Promise<GenerationCreateResult | null> {
    const prompt = draft.value.promptVersion
    const source = draft.value.sourcePrompt.trim()
    if (!canSubmit.value) throw new Error('请先填写完整的画面需求')
    if (referenceOptimizationRequired.value) {
      await preparePrompt()
      return null
    }
    if (hasReferenceAssets.value && !prompt?.confirmedAt) {
      throw new Error('请先确认参考图优化后的提示词')
    }
    cancelStaleOptimization()

    submitting.value = true
    error.value = ''
    const idempotencyKey =
      generationIdempotencyKey ?? createIdempotencyKey('generation')
    generationIdempotencyKey = idempotencyKey
    try {
      const task = await generationApi.create(
        prompt?.confirmedAt
          ? {
              promptVersionId: prompt.id,
              assetIds: draft.value.assets.map((asset) => asset.id),
              settings: draft.value.settings,
            }
          : {
              source,
              referenceAssets: referenceAssets.value.map((asset) => ({
                assetId: asset.id,
                role: asset.role ?? 'style',
              })),
              assetIds: draft.value.assets.map((asset) => asset.id),
              settings: draft.value.settings,
            },
        idempotencyKey,
      )
      generationIdempotencyKey = null
      currentTask.value = task
      return task
    } catch (caught) {
      if (!(caught instanceof AppError && caught.code === ErrorCode.Unknown)) {
        generationIdempotencyKey = null
      }
      error.value = caught instanceof Error ? caught.message : '任务提交失败'
      throw caught
    } finally {
      submitting.value = false
    }
  }

  function reuseTask(task: GenerationTask): void {
    cancelStaleOptimization()
    generationIdempotencyKey = null
    mode.value = task.mode
    const reusedDraft: WorkspaceDraft = {
      mode: task.mode,
      sourcePrompt: task.prompt.source,
      assets: task.assets,
      referencePrompt: task.prompt.sections.referencePrompt ?? '',
      promptVersion: {
        ...task.prompt,
        confirmedAt: undefined,
      },
      promptSectionBackups: {},
      settings: task.settings,
    }
    applyDefaultPromptProtection(reusedDraft)
    drafts.value[task.mode] = reusedDraft
    currentTask.value = null
  }

  function clearCurrentTask(): void {
    currentTask.value = null
  }

  return {
    mode,
    requestMode,
    drafts,
    draft,
    sourceAssets,
    referenceAssets,
    hasReferenceAssets,
    referenceOptimizationRequired,
    optimizing,
    confirming,
    submitting,
    isUploading,
    isConfirmed,
    canOptimize,
    canSubmit,
    canGenerate,
    promptNeedsConfirmation,
    error,
    currentTask,
    setMode,
    setSourcePrompt,
    updatePromptSection,
    updateReferencePrompt,
    isPromptSectionUnchanged,
    setPromptSectionUnchanged,
    updateSettings,
    hydrateAssets,
    uploadFiles,
    removeAsset,
    setReferenceRole,
    optimizePrompt,
    describeReferencePrompt,
    preparePrompt,
    confirmPrompt,
    submit,
    reuseTask,
    clearCurrentTask,
  }
})
