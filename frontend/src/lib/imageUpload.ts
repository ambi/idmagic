// magic byte による raster 画像アップロードの共通ガード (wi-89, ADR-096)。
// Application icon と Tenant branding asset は同じ受理形式・検証方針を持つため、
// クライアント側の事前検証ロジックをここに集約する (実際の検証は server 側の
// backend/shared/mediavalidation が正本)。

export const MAX_IMAGE_UPLOAD_BYTES = 256 * 1024

export const IMAGE_UPLOAD_CONTENT_TYPES = new Set([
  'image/png',
  'image/jpeg',
  'image/webp',
  'image/gif',
])

export type ImageUploadValidationError = 'too-large' | 'unsupported-type' | 'invalid-signature'

export function isSupportedImageType(file: File): boolean {
  return IMAGE_UPLOAD_CONTENT_TYPES.has(file.type)
}

export async function validateImageUploadFile(
  file: File,
  maxBytes: number = MAX_IMAGE_UPLOAD_BYTES,
): Promise<ImageUploadValidationError | null> {
  if (file.size > maxBytes) return 'too-large'
  if (!isSupportedImageType(file)) return 'unsupported-type'
  try {
    if (!(await hasImageSignature(file))) return 'invalid-signature'
  } catch {
    return 'invalid-signature'
  }
  return null
}

// safeInternalAssetURL は同一オリジンかつ指定 path segment を含む内部配信 URL のみを
// 通す。それ以外 (外部オリジン、javascript: 等) は空文字列を返す。
export function safeInternalAssetURL(value: string | undefined, pathSegment: string): string {
  if (!value) return ''
  const base = window.location.origin
  let url: URL
  try {
    url = new URL(value, base)
  } catch {
    return ''
  }
  if (url.origin !== base) return ''
  if (!url.pathname.includes(pathSegment)) return ''
  return `${url.pathname}${url.search}`
}

async function hasImageSignature(file: File): Promise<boolean> {
  const bytes = new Uint8Array(await readBlobPrefix(file, 16))
  switch (file.type) {
    case 'image/png':
      return bytesStartsWith(bytes, [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])
    case 'image/jpeg':
      return bytesStartsWith(bytes, [0xff, 0xd8, 0xff])
    case 'image/gif':
      return (
        bytesStartsWith(bytes, [0x47, 0x49, 0x46, 0x38, 0x37, 0x61]) ||
        bytesStartsWith(bytes, [0x47, 0x49, 0x46, 0x38, 0x39, 0x61])
      )
    case 'image/webp':
      return (
        bytesStartsWith(bytes, [0x52, 0x49, 0x46, 0x46]) &&
        bytesStartsWith(bytes.slice(8), [0x57, 0x45, 0x42, 0x50])
      )
    default:
      return false
  }
}

async function readBlobPrefix(file: File, length: number): Promise<ArrayBuffer> {
  const blob = file.slice(0, length)
  if (typeof blob.arrayBuffer === 'function') {
    return blob.arrayBuffer()
  }
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(reader.error)
    reader.onload = () => {
      if (reader.result instanceof ArrayBuffer) {
        resolve(reader.result)
        return
      }
      reject(new Error('画像を読み込めませんでした。'))
    }
    reader.readAsArrayBuffer(blob)
  })
}

function bytesStartsWith(bytes: Uint8Array, signature: number[]): boolean {
  if (bytes.length < signature.length) return false
  return signature.every((value, index) => bytes[index] === value)
}
