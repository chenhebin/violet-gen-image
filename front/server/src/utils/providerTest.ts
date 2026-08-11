import type { AIModel, AIProvider } from '@/types'

export type ModelTestRequestKind = 'text-to-image' | 'image-to-image'

export interface ModelTestRequestPreview {
  kind: ModelTestRequestKind
  label: string
  method: 'POST'
  url: string
  curl: string
}

const TEST_PROMPTS = {
  textToImage: 'A plain white studio card.',
  imageToImage: 'Keep this image unchanged.',
} as const

export function providerEndpoint(
  baseUrl: string,
  endpointPath: string,
): string {
  const base = baseUrl.trim().replace(/\/+$/, '')
  const endpoint = `/${endpointPath.replace(/^\/+/, '')}`
  if (base.endsWith('/v1') && endpoint.startsWith('/v1/')) {
    return `${base}${endpoint.slice(3)}`
  }
  return `${base}${endpoint}`
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'"'"'`)}'`
}

function jsonCurl(url: string, body: Record<string, string>): string {
  return [
    `curl --location ${shellQuote(url)} \\`,
    "  --header 'Authorization: Bearer <API_KEY>' \\",
    "  --header 'Content-Type: application/json' \\",
    `  --data ${shellQuote(JSON.stringify(body, null, 2))}`,
  ].join('\n')
}

function multipartCurl(url: string, modelId: string): string {
  return [
    `curl --location ${shellQuote(url)} \\`,
    "  --header 'Authorization: Bearer <API_KEY>' \\",
    `  --form ${shellQuote(`model=${modelId}`)} \\`,
    `  --form ${shellQuote(`prompt=${TEST_PROMPTS.imageToImage}`)} \\`,
    "  --form 'image=@capability-test.png;type=image/png'",
  ].join('\n')
}

export function buildImageModelTestRequests(
  provider: AIProvider,
  model: AIModel,
): ModelTestRequestPreview[] {
  const requests: ModelTestRequestPreview[] = []

  if (model.capabilities.textToImage) {
    const url = providerEndpoint(
      provider.baseUrl,
      '/v1/images/generations',
    )
    requests.push({
      kind: 'text-to-image',
      label: '文生图能力',
      method: 'POST',
      url,
      curl: jsonCurl(url, {
        model: model.modelId,
        prompt: TEST_PROMPTS.textToImage,
      }),
    })
  }

  if (model.capabilities.imageToImage) {
    const url = providerEndpoint(provider.baseUrl, '/v1/images/edits')
    requests.push({
      kind: 'image-to-image',
      label: '图生图 / 编辑能力',
      method: 'POST',
      url,
      curl: multipartCurl(url, model.modelId),
    })
  }

  return requests
}
