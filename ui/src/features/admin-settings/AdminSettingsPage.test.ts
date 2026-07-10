import { describe, expect, it } from 'vitest'
import { displayNameError, passwordPolicyOverride } from './AdminSettingsPage'

describe('admin settings presentation helpers', () => {
  it('rejects a blank display name and trims policy input presence', () => {
    expect(displayNameError('  ')).toBe('表示名を入力してください。')
    expect(displayNameError('Acme')).toBeNull()
    expect(passwordPolicyOverride(' 12 ', '', '3')).toEqual({ min_length: 12, history_depth: 3 })
  })
})
