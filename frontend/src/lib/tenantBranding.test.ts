import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  isHTTPSURL,
  isTenantBrandingAssetFile,
  isValidHexColor,
  MAX_TENANT_BRANDING_ASSET_BYTES,
  safeTenantBrandingAssetURL,
  validateTenantBrandingAssetFile,
} from './tenantBranding'

describe('tenant branding guards', () => {
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

  it('accepts same-origin branding asset delivery URLs', () => {
    expect(safeTenantBrandingAssetURL('/realms/acme/tenant-branding-assets/logo/key-1')).toBe(
      '/realms/acme/tenant-branding-assets/logo/key-1',
    )
    expect(
      safeTenantBrandingAssetURL(
        'https://id.example/realms/acme/tenant-branding-assets/logo/key-1',
      ),
    ).toBe('/realms/acme/tenant-branding-assets/logo/key-1')
  })

  it('rejects URLs that are not internal branding asset delivery URLs', () => {
    expect(safeTenantBrandingAssetURL('javascript:alert(1)')).toBe('')
    expect(
      safeTenantBrandingAssetURL('https://evil.example/tenant-branding-assets/logo/key-1'),
    ).toBe('')
    expect(safeTenantBrandingAssetURL('/api/admin/tenant/branding')).toBe('')
  })

  it('accepts only supported raster image file types', () => {
    expect(isTenantBrandingAssetFile(new File(['x'], 'logo.png', { type: 'image/png' }))).toBe(true)
    expect(isTenantBrandingAssetFile(new File(['x'], 'logo.svg', { type: 'image/svg+xml' }))).toBe(
      false,
    )
  })

  it('rejects oversized and spoofed asset files', async () => {
    await expect(
      validateTenantBrandingAssetFile(
        new File([new Uint8Array(MAX_TENANT_BRANDING_ASSET_BYTES + 1)], 'logo.png', {
          type: 'image/png',
        }),
      ),
    ).resolves.toBe('too-large')
    await expect(
      validateTenantBrandingAssetFile(
        new File(['<script>alert(1)</script>'], 'logo.png', { type: 'image/png' }),
      ),
    ).resolves.toBe('invalid-signature')
    await expect(
      validateTenantBrandingAssetFile(
        new File(['<svg onload=alert(1)></svg>'], 'logo.svg', { type: 'image/svg+xml' }),
      ),
    ).resolves.toBe('unsupported-type')
  })

  it('validates hex color tokens', () => {
    expect(isValidHexColor('#0f172a')).toBe(true)
    expect(isValidHexColor('#0F172A')).toBe(true)
    expect(isValidHexColor('0f172a')).toBe(false)
    expect(isValidHexColor('red')).toBe(false)
    expect(isValidHexColor('#fff')).toBe(false)
  })

  it('validates https-only links', () => {
    expect(isHTTPSURL('https://support.example.com')).toBe(true)
    expect(isHTTPSURL('http://support.example.com')).toBe(false)
    expect(isHTTPSURL('javascript:alert(1)')).toBe(false)
  })
})
