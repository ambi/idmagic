import { describe, expect, it } from 'vitest'
import { AuthenticationAPIError } from '../../api'
import type { SignInRule } from '../../types'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import {
  appRuleFromInputs,
  initials,
  messageOf,
  parseList,
  signInRuleWeakerThanDefault,
  summarizeSignInRule,
} from './AdminApplicationsShared'

const t = adminApplicationsDictionary.en

function rule(overrides: Partial<SignInRule> = {}): SignInRule {
  return {
    rule_id: 'r1',
    name: 'r1',
    enabled: true,
    required_authn: { strength: 'Password' },
    condition: {},
    ...overrides,
  }
}

describe('messageOf', () => {
  it('uses the API error message when the cause is an AuthenticationAPIError', () => {
    expect(messageOf(new AuthenticationAPIError('nope'), 'fallback')).toBe('nope')
  })

  it('falls back for any other cause', () => {
    expect(messageOf(new Error('boom'), 'fallback')).toBe('fallback')
    expect(messageOf('boom', 'fallback')).toBe('fallback')
  })
})

describe('parseList', () => {
  it('splits on whitespace and commas and de-duplicates', () => {
    expect(parseList('a, b\nb   c,a')).toEqual(['a', 'b', 'c'])
  })

  it('returns an empty array for blank input', () => {
    expect(parseList('   ')).toEqual([])
  })
})

describe('initials', () => {
  it('takes the first two characters uppercased', () => {
    expect(initials('acme corp')).toBe('AC')
  })

  it('falls back to ?? for blank input', () => {
    expect(initials('   ')).toBe('??')
  })
})

describe('summarizeSignInRule', () => {
  it('summarizes strength only when no condition is set', () => {
    expect(summarizeSignInRule(rule(), t)).toBe(t.strengthPasswordLabel)
  })

  it('appends reauth and network conditions when present', () => {
    const summary = summarizeSignInRule(
      rule({
        required_authn: { strength: 'Mfa' },
        condition: { reauth_max_age_seconds: 300, network_allow_cidrs: ['10.0.0.0/8'] },
      }),
      t,
    )
    expect(summary).toBe(
      [
        t.strengthMfaLabel,
        t.reauthSuffix.replace('{seconds}', '300'),
        t.allowedNetworkPrefix.replace('{cidrs}', '10.0.0.0/8'),
      ].join(' / '),
    )
  })
})

describe('appRuleFromInputs', () => {
  it('parses reauth seconds and CIDR lines into a SignInRule', () => {
    const result = appRuleFromInputs('Mfa', '600', '10.0.0.0/8\n192.168.1.0/24')
    expect(result.required_authn.strength).toBe('Mfa')
    expect(result.condition.reauth_max_age_seconds).toBe(600)
    expect(result.condition.network_allow_cidrs).toEqual(['10.0.0.0/8', '192.168.1.0/24'])
  })

  it('omits reauth and CIDRs when inputs are blank or non-positive', () => {
    const result = appRuleFromInputs('Password', '  ', '')
    expect(result.condition.reauth_max_age_seconds).toBeUndefined()
    expect(result.condition.network_allow_cidrs).toBeUndefined()
  })

  it('treats a non-positive reauth value as unset', () => {
    const result = appRuleFromInputs('Password', '0', '')
    expect(result.condition.reauth_max_age_seconds).toBeUndefined()
  })
})

describe('signInRuleWeakerThanDefault', () => {
  it('is false when there is no enabled default rule', () => {
    expect(signInRuleWeakerThanDefault(rule(), [])).toBe(false)
    expect(signInRuleWeakerThanDefault(rule(), [rule({ enabled: false })])).toBe(false)
  })

  it('is true when default requires MFA but the app rule does not', () => {
    const defaults = [rule({ required_authn: { strength: 'Mfa' } })]
    expect(
      signInRuleWeakerThanDefault(rule({ required_authn: { strength: 'Password' } }), defaults),
    ).toBe(true)
  })

  it('is true when the app allows a longer reauth window than the default', () => {
    const defaults = [rule({ condition: { reauth_max_age_seconds: 300 } })]
    const appRule = rule({ condition: { reauth_max_age_seconds: 3600 } })
    expect(signInRuleWeakerThanDefault(appRule, defaults)).toBe(true)
  })

  it('is true when the app allows a network the default does not', () => {
    const defaults = [rule({ condition: { network_allow_cidrs: ['10.0.0.0/8'] } })]
    const appRule = rule({ condition: { network_allow_cidrs: ['0.0.0.0/0'] } })
    expect(signInRuleWeakerThanDefault(appRule, defaults)).toBe(true)
  })

  it('is false when the app rule matches or is stricter than the default', () => {
    const defaults = [
      rule({
        required_authn: { strength: 'Mfa' },
        condition: { reauth_max_age_seconds: 3600, network_allow_cidrs: ['10.0.0.0/8'] },
      }),
    ]
    const appRule = rule({
      required_authn: { strength: 'Mfa' },
      condition: { reauth_max_age_seconds: 300, network_allow_cidrs: ['10.0.0.0/8'] },
    })
    expect(signInRuleWeakerThanDefault(appRule, defaults)).toBe(false)
  })
})
