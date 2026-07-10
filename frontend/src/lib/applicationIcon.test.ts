import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  isApplicationIconFile,
  MAX_APPLICATION_ICON_BYTES,
  safeApplicationIconURL,
  validateApplicationIconFile,
} from './applicationIcon'

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

  it('accepts supported raster image signatures', async () => {
    await expect(
      validateApplicationIconFile(
        new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])], 'icon.png', {
          type: 'image/png',
        }),
      ),
    ).resolves.toBeNull()
    await expect(
      validateApplicationIconFile(
        new File([new Uint8Array([0xff, 0xd8, 0xff, 0xe0])], 'icon.jpg', {
          type: 'image/jpeg',
        }),
      ),
    ).resolves.toBeNull()
    await expect(
      validateApplicationIconFile(
        new File([new Uint8Array([0x47, 0x49, 0x46, 0x38, 0x39, 0x61])], 'icon.gif', {
          type: 'image/gif',
        }),
      ),
    ).resolves.toBeNull()
    await expect(
      validateApplicationIconFile(
        new File(
          [
            new Uint8Array([
              0x52, 0x49, 0x46, 0x46, 0x04, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50,
            ]),
          ],
          'icon.webp',
          { type: 'image/webp' },
        ),
      ),
    ).resolves.toBeNull()
  })

  it('rejects oversized, unsupported, and spoofed icon files', async () => {
    await expect(
      validateApplicationIconFile(
        new File([new Uint8Array(MAX_APPLICATION_ICON_BYTES + 1)], 'icon.png', {
          type: 'image/png',
        }),
      ),
    ).resolves.toBe('too-large')
    await expect(
      validateApplicationIconFile(new File(['<svg></svg>'], 'icon.svg', { type: 'image/svg+xml' })),
    ).resolves.toBe('unsupported-type')
    await expect(
      validateApplicationIconFile(
        new File(['<script>alert(1)</script>'], 'icon.png', { type: 'image/png' }),
      ),
    ).resolves.toBe('invalid-signature')
  })
})
