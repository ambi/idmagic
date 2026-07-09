const APPLICATION_ICON_PATH_SEGMENT = '/application-icons/'

export const APPLICATION_ICON_CONTENT_TYPES = new Set([
  'image/png',
  'image/jpeg',
  'image/webp',
  'image/gif',
])

export function isApplicationIconFile(file: File): boolean {
  return APPLICATION_ICON_CONTENT_TYPES.has(file.type)
}

export function safeApplicationIconURL(value?: string): string {
  if (!value) return ''
  const base = window.location.origin
  let url: URL
  try {
    url = new URL(value, base)
  } catch {
    return ''
  }
  if (url.origin !== base) return ''
  if (!url.pathname.includes(APPLICATION_ICON_PATH_SEGMENT)) return ''
  return `${url.pathname}${url.search}`
}
