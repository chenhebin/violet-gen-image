import { del, get, set } from 'idb-keyval'

const objectUrls = new Map<string, string>()

export async function cacheAssetFile(assetId: string, file: File): Promise<string> {
  await set(`asset:${assetId}`, file)
  return createAssetUrl(assetId, file)
}

export async function loadAssetUrl(assetId: string): Promise<string | undefined> {
  if (objectUrls.has(assetId)) {
    return objectUrls.get(assetId)
  }

  const file = await get<File>(`asset:${assetId}`)
  return file ? createAssetUrl(assetId, file) : undefined
}

export async function removeCachedAsset(assetId: string): Promise<void> {
  const url = objectUrls.get(assetId)
  if (url) {
    URL.revokeObjectURL(url)
    objectUrls.delete(assetId)
  }
  await del(`asset:${assetId}`)
}

function createAssetUrl(assetId: string, file: File): string {
  const previous = objectUrls.get(assetId)
  if (previous) {
    URL.revokeObjectURL(previous)
  }

  const url = URL.createObjectURL(file)
  objectUrls.set(assetId, url)
  return url
}
