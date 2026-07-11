import { describe, expect, it } from 'vitest'
import { brandingColorError, brandingSupportURLError } from './BrandingTab'

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

  it('rejects malformed hex colors', () => {
    expect(brandingColorError('#0f172a')).toBeNull()
    expect(brandingColorError('#fff')).not.toBeNull()
    expect(brandingColorError('red')).not.toBeNull()
  })
})
