import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { isApplicationIconFile, safeApplicationIconURL } from './applicationIcon'

describe('application icon guards', () => {
  const originalLocation = window.location

  beforeEach(() => {
    vi.stubGlobal('location', {
      ...originalLocation,
      origin: 'https://id.example',
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('accepts same-origin application icon delivery URLs', () => {
    expect(safeApplicationIconURL('/realms/acme/application-icons/app-1/key-1')).toBe(
      '/realms/acme/application-icons/app-1/key-1',
    )
    expect(
      safeApplicationIconURL('https://id.example/realms/acme/application-icons/app-1/key-1'),
    ).toBe('/realms/acme/application-icons/app-1/key-1')
  })

  it('rejects URLs that are not internal icon delivery URLs', () => {
    expect(safeApplicationIconURL('javascript:alert(1)')).toBe('')
    expect(safeApplicationIconURL('https://evil.example/application-icons/app-1/key-1')).toBe('')
    expect(safeApplicationIconURL('/api/admin/applications/app-1/icon')).toBe('')
  })

  it('accepts only supported raster image file types', () => {
    expect(isApplicationIconFile(new File(['x'], 'icon.png', { type: 'image/png' }))).toBe(true)
    expect(isApplicationIconFile(new File(['x'], 'icon.svg', { type: 'image/svg+xml' }))).toBe(
      false,
    )
    expect(isApplicationIconFile(new File(['x'], 'icon.html', { type: 'text/html' }))).toBe(false)
  })
})
