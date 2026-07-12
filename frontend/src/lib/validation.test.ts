import { describe, it, expect } from 'vitest'
import { validateReauthMaxAge, parseNetworkCIDRs } from './validation'

describe('validateReauthMaxAge', () => {
  it('should return undefined for empty or whitespace-only input', () => {
    expect(validateReauthMaxAge('')).toEqual({ isValid: true, parsed: undefined })
    expect(validateReauthMaxAge('   ')).toEqual({ isValid: true, parsed: undefined })
  })

  it('should parse valid positive integer strings', () => {
    expect(validateReauthMaxAge('3600')).toEqual({ isValid: true, parsed: 3600 })
    expect(validateReauthMaxAge(' 1800 ')).toEqual({ isValid: true, parsed: 1800 })
    expect(validateReauthMaxAge('1')).toEqual({ isValid: true, parsed: 1 })
  })

  it('should return error for non-integer or decimal numbers', () => {
    expect(validateReauthMaxAge('1.5')).toEqual({ isValid: false })
    expect(validateReauthMaxAge('abc')).toEqual({ isValid: false })
  })

  it('should return error for numbers less than 1', () => {
    expect(validateReauthMaxAge('0')).toEqual({ isValid: false })
    expect(validateReauthMaxAge('-10')).toEqual({ isValid: false })
  })
})

describe('parseNetworkCIDRs', () => {
  it('should return empty array for empty input', () => {
    expect(parseNetworkCIDRs('')).toEqual([])
    expect(parseNetworkCIDRs('\n\n')).toEqual([])
  })

  it('should split by newline and trim values', () => {
    const input = '192.168.1.0/24\n  10.0.0.0/8  \n\n172.16.0.0/12'
    expect(parseNetworkCIDRs(input)).toEqual(['192.168.1.0/24', '10.0.0.0/8', '172.16.0.0/12'])
  })
})
