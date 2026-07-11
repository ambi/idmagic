import { describe, expect, it } from 'vitest'
import { brandingColorError, brandingFooterLinkError, brandingSupportURLError } from './BrandingTab'

describe('branding tab presentation helpers', () => {
  it('allows blank links and colors (means unset)', () => {
    expect(brandingSupportURLError('')).toBeNull()
    expect(brandingSupportURLError('   ')).toBeNull()
    expect(brandingColorError('')).toBeNull()
  })

  it('rejects non-https links', () => {
    expect(brandingSupportURLError('https://support.example.com')).toBeNull()
    expect(brandingSupportURLError('http://support.example.com')).not.toBeNull()
    expect(brandingSupportURLError('javascript:alert(1)')).not.toBeNull()
  })

  it('requires footer link label and URL as a pair', () => {
    expect(brandingFooterLinkError('', '')).toBeNull()
    expect(brandingFooterLinkError('ヘルプ', 'https://help.example.com')).toBeNull()
    expect(brandingFooterLinkError('', 'https://help.example.com')).not.toBeNull()
    expect(brandingFooterLinkError('ヘルプ', '')).not.toBeNull()
    expect(brandingFooterLinkError('<img>', 'https://help.example.com')).toBeNull()
  })

  it('accepts any #rrggbb color and rejects malformed values', () => {
    expect(brandingColorError('#0f172a')).toBeNull()
    expect(brandingColorError('#eeeeee')).toBeNull()
    expect(brandingColorError('#fff')).not.toBeNull()
    expect(brandingColorError('red')).not.toBeNull()
  })
})
