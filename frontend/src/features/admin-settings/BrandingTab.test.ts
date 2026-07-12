import { describe, expect, it } from 'vitest'
import { brandingTabDictionary } from './BrandingTab.i18n'
import { brandingColorError, brandingFooterLinkError, brandingSupportURLError } from './BrandingTab'

const t = brandingTabDictionary.en

describe('branding tab presentation helpers', () => {
  it('allows blank links and colors (means unset)', () => {
    expect(brandingSupportURLError('', t)).toBeNull()
    expect(brandingSupportURLError('   ', t)).toBeNull()
    expect(brandingColorError('', t)).toBeNull()
  })

  it('rejects non-https links', () => {
    expect(brandingSupportURLError('https://support.example.com', t)).toBeNull()
    expect(brandingSupportURLError('http://support.example.com', t)).not.toBeNull()
    expect(brandingSupportURLError('javascript:alert(1)', t)).not.toBeNull()
  })

  it('requires footer link label and URL as a pair', () => {
    expect(brandingFooterLinkError('', '', t)).toBeNull()
    expect(brandingFooterLinkError('Help', 'https://help.example.com', t)).toBeNull()
    expect(brandingFooterLinkError('', 'https://help.example.com', t)).not.toBeNull()
    expect(brandingFooterLinkError('Help', '', t)).not.toBeNull()
    expect(brandingFooterLinkError('<img>', 'https://help.example.com', t)).toBeNull()
  })

  it('accepts any #rrggbb color and rejects malformed values', () => {
    expect(brandingColorError('#0f172a', t)).toBeNull()
    expect(brandingColorError('#eeeeee', t)).toBeNull()
    expect(brandingColorError('#fff', t)).not.toBeNull()
    expect(brandingColorError('red', t)).not.toBeNull()
  })
})
