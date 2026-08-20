import { assetApi } from '@/services/api'

interface DownloadAssetOptions {
  assetId: string
  currentUrl?: string
  filename: string
}

/** Refreshes the short-lived download URL before triggering a browser download. */
export async function downloadAsset({
  assetId,
  currentUrl,
  filename,
}: DownloadAssetOptions): Promise<void> {
  let url = currentUrl
  try {
    const signed = await assetApi.getUrl(assetId, 'download')
    url = signed.url
  } catch {
    // Keep the last known URL as a best-effort fallback.
  }
  if (!url || typeof document === 'undefined') return

  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.rel = 'noreferrer'
  anchor.style.display = 'none'
  document.body.append(anchor)
  anchor.click()
  anchor.remove()
}
