import { describe, expect, it } from 'vitest'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { displayNameError, passwordPolicyOverride } from './AdminSettingsShared'

const t = adminSettingsDictionary.en

describe('admin settings presentation helpers', () => {
  it('rejects a blank display name and trims policy input presence', () => {
    expect(displayNameError('  ', t)).toBe(t.displayNameRequiredError)
    expect(displayNameError('Acme', t)).toBeNull()
    expect(passwordPolicyOverride(' 12 ', '', '3')).toEqual({ min_length: 12, history_depth: 3 })
  })
})
