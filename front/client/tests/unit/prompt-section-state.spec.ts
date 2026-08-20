import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PROMPT_CONFIG, PROMPT_SECTION_OPTIONS } from '@/config'
import { generationApi, promptApi } from '@/services/api'
import { useWorkspaceStore } from '@/stores/workspace'
import type { GenerationTask, PromptVersion } from '@/types/domain'

const optimizedPrompt: PromptVersion = {
  id: 'optimized-prompt',
  source: '只修改服装颜色',
  sections: {
    subject: '保留人物五官，服装改成白色',
    scene: '保持原场景的室内环境',
    style: '真实摄影风格',
    composition: '沿用原图构图',
    details: '保留皮肤纹理',
    negative: '不要改变人物身份',
    output: '输出高清竖图',
  },
}

describe('prompt section unchanged state', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('restores the latest user value after the switch is disabled', () => {
    const workspace = useWorkspaceStore()
    workspace.draft.promptVersion = {
      id: 'prompt-1',
      source: 'source',
      sections: {
        subject: '原来的主体描述',
        scene: '场景',
        style: '风格',
        composition: '构图',
        details: '细节',
        negative: '禁止内容',
        output: '输出规格',
      },
    }

    workspace.setPromptSectionUnchanged('subject', true)
    expect(workspace.isPromptSectionUnchanged('subject')).toBe(true)
    expect(workspace.draft.promptVersion.sections.subject).toBe(
      PROMPT_CONFIG.unchangedText,
    )

    workspace.updatePromptSection('subject', '禁用时不应写入')
    workspace.setPromptSectionUnchanged('subject', false)
    expect(workspace.draft.promptVersion.sections.subject).toBe(
      '原来的主体描述',
    )

    workspace.updatePromptSection('subject', '用户修改后的主体')
    workspace.setPromptSectionUnchanged('subject', true)
    workspace.setPromptSectionUnchanged('subject', false)
    expect(workspace.draft.promptVersion.sections.subject).toBe(
      '用户修改后的主体',
    )
  })

  it('keeps each section independent and invalidates confirmation', () => {
    const workspace = useWorkspaceStore()
    workspace.draft.promptVersion = {
      id: 'prompt-2',
      source: 'source',
      sections: {
        subject: '主体',
        scene: '场景原值',
        style: '风格原值',
        composition: '构图',
        details: '细节',
        negative: '禁止内容',
        output: '输出规格',
      },
      confirmedAt: '2026-08-01T00:00:00Z',
    }

    workspace.setPromptSectionUnchanged('scene', true)
    workspace.setPromptSectionUnchanged('style', true)

    expect(workspace.isConfirmed).toBe(false)
    expect(workspace.isPromptSectionUnchanged('scene')).toBe(true)
    expect(workspace.isPromptSectionUnchanged('style')).toBe(true)
    expect(workspace.isPromptSectionUnchanged('subject')).toBe(false)

    workspace.setPromptSectionUnchanged('scene', false)
    expect(workspace.draft.promptVersion.sections.scene).toBe('场景原值')
    expect(workspace.draft.promptVersion.sections.style).toBe(
      PROMPT_CONFIG.unchangedText,
    )
  })

  it('uses image editing when an original image is present', async () => {
    const optimize = vi
      .spyOn(promptApi, 'optimize')
      .mockResolvedValue(structuredClone(optimizedPrompt))
    const workspace = useWorkspaceStore()
    workspace.draft.assets.push({
      id: 'source-1',
      name: 'source.png',
      kind: 'source',
      mimeType: 'image/png',
      size: 128,
      uploadProgress: 100,
    })
    workspace.setSourcePrompt('只修改服装颜色')

    await workspace.optimizePrompt()

    expect(optimize).toHaveBeenCalledWith(
      expect.objectContaining({
        mode: 'image-to-image',
        sourceAssetIds: ['source-1'],
      }),
      expect.anything(),
    )

    for (const { key } of PROMPT_SECTION_OPTIONS) {
      expect(workspace.isPromptSectionUnchanged(key)).toBe(true)
      expect(workspace.draft.promptVersion?.sections[key]).toBe(
        PROMPT_CONFIG.unchangedText,
      )
      expect(workspace.draft.promptSectionBackups[key]).toBe(
        optimizedPrompt.sections[key],
      )
    }

    workspace.setPromptSectionUnchanged('style', false)
    expect(workspace.draft.promptVersion?.sections.style).toBe(
      optimizedPrompt.sections.style,
    )
  })

  it('uses text generation when no image is present', async () => {
    const optimize = vi
      .spyOn(promptApi, 'optimize')
      .mockResolvedValue(structuredClone(optimizedPrompt))
    const workspace = useWorkspaceStore()
    workspace.setSourcePrompt('生成一张室内人像')

    await workspace.optimizePrompt()

    expect(optimize).toHaveBeenCalledWith(
      expect.objectContaining({
        mode: 'text-to-image',
        sourceAssetIds: [],
        referenceAssets: [],
      }),
      expect.anything(),
    )

    for (const { key } of PROMPT_SECTION_OPTIONS) {
      expect(workspace.isPromptSectionUnchanged(key)).toBe(true)
      expect(workspace.draft.promptVersion?.sections[key]).toBe(
        PROMPT_CONFIG.unchangedText,
      )
      expect(workspace.draft.promptSectionBackups[key]).toBe(
        optimizedPrompt.sections[key],
      )
    }
  })

  it('uses image editing when only a reference image is present', async () => {
    const optimize = vi
      .spyOn(promptApi, 'optimize')
      .mockResolvedValue(structuredClone(optimizedPrompt))
    const workspace = useWorkspaceStore()
    workspace.draft.assets.push({
      id: 'reference-1',
      name: 'reference.png',
      kind: 'reference',
      role: 'style',
      mimeType: 'image/png',
      size: 128,
      uploadProgress: 100,
    })
    workspace.setSourcePrompt('参考这张图片生成一张室内人像')

    await workspace.optimizePrompt()

    expect(optimize).toHaveBeenCalledWith(
      expect.objectContaining({
        mode: 'image-to-image',
        sourceAssetIds: [],
        referenceAssets: [
          { assetId: 'reference-1', role: 'style' },
        ],
      }),
      expect.anything(),
    )
  })

  it('returns to text generation after the final image is removed', () => {
    const workspace = useWorkspaceStore()
    expect(workspace.requestMode).toBe('text-to-image')

    workspace.draft.assets.push({
      id: 'source-1',
      name: 'source.png',
      kind: 'source',
      mimeType: 'image/png',
      size: 128,
      uploadProgress: 100,
    })
    expect(workspace.requestMode).toBe('image-to-image')

    workspace.draft.assets = []
    expect(workspace.requestMode).toBe('text-to-image')
  })

  it('submits the raw requirement directly when no prompt was confirmed', async () => {
    const create = vi
      .spyOn(generationApi, 'create')
      .mockResolvedValue({ id: 'direct-task' } as GenerationTask)
    const workspace = useWorkspaceStore()
    workspace.setSourcePrompt('生成一张自然光室内人像')

    expect(workspace.canSubmit).toBe(true)
    await workspace.submit()

    expect(create).toHaveBeenCalledWith(
      {
        source: '生成一张自然光室内人像',
        referenceAssets: [],
        assetIds: [],
        settings: workspace.draft.settings,
      },
      expect.any(String),
    )
  })

  it('requires a source image before preparing a reference image prompt', async () => {
    const workspace = useWorkspaceStore()
    workspace.setSourcePrompt('参考这张图片生成一张自然光室内人像')
    workspace.draft.assets.push({
      id: 'reference-only',
      name: 'reference.png',
      kind: 'reference',
      role: 'composition',
      mimeType: 'image/png',
      size: 128,
      uploadProgress: 100,
    })

    await expect(workspace.submit()).rejects.toThrow('请先上传待修改原图')
  })

  it('prepares a reference prompt before allowing image generation', async () => {
    const describeReferences = vi
      .spyOn(promptApi, 'describeReferences')
      .mockResolvedValue({
        prompt: '清透杂志氛围，50mm 镜头，柔和侧逆光，真实材质细节',
        referenceAssets: [{ assetId: 'reference-1', role: 'style' }],
      })
    const optimize = vi
      .spyOn(promptApi, 'optimize')
      .mockResolvedValue(structuredClone(optimizedPrompt))
    const create = vi.spyOn(generationApi, 'create')
    const workspace = useWorkspaceStore()
    workspace.setSourcePrompt('只修改服装颜色')
    workspace.draft.assets.push(
      {
        id: 'source-1',
        name: 'source.png',
        kind: 'source',
        mimeType: 'image/png',
        size: 128,
        uploadProgress: 100,
      },
      {
        id: 'reference-1',
        name: 'reference.png',
        kind: 'reference',
        role: 'style',
        mimeType: 'image/png',
        size: 128,
        uploadProgress: 100,
      },
    )

    const result = await workspace.submit()

    expect(result).toBeNull()
    expect(describeReferences).toHaveBeenCalledWith(
      [{ assetId: 'reference-1', role: 'style' }],
      expect.anything(),
    )
    expect(optimize).toHaveBeenCalledWith(
      expect.objectContaining({
        sourceAssetIds: ['source-1'],
        referencePrompt: '清透杂志氛围，50mm 镜头，柔和侧逆光，真实材质细节',
      }),
      expect.anything(),
    )
    expect(workspace.draft.promptVersion?.sections.referencePrompt).toContain(
      '清透杂志氛围',
    )
    for (const { key } of PROMPT_SECTION_OPTIONS) {
      expect(workspace.isPromptSectionUnchanged(key)).toBe(false)
      expect(workspace.draft.promptVersion?.sections[key]).toBe(
        optimizedPrompt.sections[key],
      )
      expect(workspace.draft.promptSectionBackups[key]).toBeUndefined()
    }
    expect(create).not.toHaveBeenCalled()
  })

  it('migrates an existing optimized draft to the default protected state', () => {
    localStorage.setItem(
      'yingyan:workspace-drafts:v1',
      JSON.stringify({
        'image-to-image': {
          mode: 'image-to-image',
          sourcePrompt: optimizedPrompt.source,
          assets: [],
          promptVersion: structuredClone(optimizedPrompt),
          promptSectionBackups: {},
          settings: {
            aspectRatio: '3:4',
            outputCount: 1,
            referenceStrength: 68,
          },
        },
      }),
    )

    const workspace = useWorkspaceStore()

    for (const { key } of PROMPT_SECTION_OPTIONS) {
      expect(workspace.isPromptSectionUnchanged(key)).toBe(true)
      expect(workspace.draft.promptVersion?.sections[key]).toBe(
        PROMPT_CONFIG.unchangedText,
      )
      expect(workspace.draft.promptSectionBackups[key]).toBe(
        optimizedPrompt.sections[key],
      )
    }
  })
})
