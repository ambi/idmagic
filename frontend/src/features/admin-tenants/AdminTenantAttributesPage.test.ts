import { describe, expect, it } from 'vitest'
import { newAttribute, normalizeAttribute } from './AdminTenantAttributesPage'

describe('attribute presentation helpers', () => {
  it('creates a safe default and normalizes form-only values', () => {
    expect(newAttribute().visibility).toBe('admin_readable')
    expect(
      normalizeAttribute({
        ...newAttribute(),
        key: ' department ',
        label: ' 部署 ',
        type: 'string_array',
        claim_name: ' dept ',
        oidc_scope: ' profile ',
      }),
    ).toMatchObject({
      key: 'department',
      label: '部署',
      multi_valued: true,
      claim_name: 'dept',
      oidc_scope: 'profile',
    })
  })
})
