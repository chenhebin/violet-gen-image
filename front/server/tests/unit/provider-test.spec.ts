import { describe, expect, it } from 'vitest'
import type { AIModel, AIProvider } from '@/types'
import {
  buildImageModelTestRequests,
  providerEndpoint,
} from '@/utils/providerTest'

const provider = {
  id: 'provider-1',
  name: 'Daidai',
  code: 'daidai',
  protocol: 'openai-compatible',
  baseUrl: 'https://api.daidaiweb.cn/v1',
  maskedApiKey: 'sk-a9***13d',
  enabled: true,
  connectionStatus: 'healthy',
  createdAt: '2026-07-31T00:00:00Z',
  updatedAt: '2026-07-31T00:00:00Z',
} satisfies AIProvider

const model = {
  id: 'model-1',
  providerId: provider.id,
  providerName: provider.name,
  displayName: 'GPT Image 2',
  modelId: 'gpt-image-2',
  type: 'image',
  enabled: true,
  connectionStatus: 'untested',
  capabilities: { textToImage: true, imageToImage: true },
  createdAt: '2026-07-31T00:00:00Z',
  updatedAt: '2026-07-31T00:00:00Z',
  isPlatformModel: false,
} satisfies AIModel

describe('image model test request preview', () => {
  it('does not duplicate the v1 path', () => {
    expect(
      providerEndpoint(provider.baseUrl, '/v1/images/generations'),
    ).toBe('https://api.daidaiweb.cn/v1/images/generations')
  })

  it('matches the backend text and image test requests', () => {
    const requests = buildImageModelTestRequests(provider, model)

    expect(requests).toHaveLength(2)
    expect(requests[0]?.curl).toContain('/v1/images/generations')
    expect(requests[0]?.curl).toContain('"model": "gpt-image-2"')
    expect(requests[0]?.curl).not.toContain('"n"')
    expect(requests[1]?.curl).toContain('/v1/images/edits')
    expect(requests[1]?.curl).toContain('image=@capability-test.png')
    expect(requests[0]?.curl).not.toContain(provider.maskedApiKey)
  })
})
