import { describe, it, expect } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import { AdminTenantAttributesPage } from './AdminTenantAttributesPage'
import { adminTenantAttributesDictionary } from './AdminTenantAttributesPage.i18n'
import type { TenantUserAttributeSchema } from '../../types'

const schema: TenantUserAttributeSchema = {
  tenant_id: 'tenant-1',
  builtin: [],
  attributes: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui, { locale: 'ja' })

describe('AdminTenantAttributesPage', () => {
  it('renders in English by default', async () => {
    await renderWithRouterBase(<AdminTenantAttributesPage csrfToken="csrf" schema={schema} />)
    expect(
      screen.getByRole('heading', { name: adminTenantAttributesDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(adminTenantAttributesDictionary.en.noCustomAttributesNotice),
    ).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(<AdminTenantAttributesPage csrfToken="csrf" schema={schema} />)
    expect(
      screen.getByRole('heading', { name: adminTenantAttributesDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(adminTenantAttributesDictionary.ja.noCustomAttributesNotice),
    ).toBeInTheDocument()
  })
})
