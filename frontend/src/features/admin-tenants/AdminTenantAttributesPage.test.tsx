import { describe, it, expect } from 'bun:test'
import { screen } from '@testing-library/react'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import { AdminTenantAttributesPage } from './AdminTenantAttributesPage'
import { adminTenantAttributesDictionary } from './AdminTenantAttributesPage.i18n'
import type { TenantUserAttributeSchema, UserAttributeDef } from '../../types'

const schema: TenantUserAttributeSchema = {
  tenant_id: 'tenant-1',
  builtin: [],
  attributes: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const attribute: UserAttributeDef = {
  key: 'region',
  label: 'Region',
  type: 'string',
  multi_valued: false,
  required: false,
  editable_by_user: false,
  visibility: 'admin_readable',
  pii: false,
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

  it('links "add attribute" to the dedicated create route instead of a modal', async () => {
    await renderWithRouterBase(<AdminTenantAttributesPage csrfToken="csrf" schema={schema} />)
    expect(
      screen.getByRole('button', {
        name: new RegExp(adminTenantAttributesDictionary.en.addAttribute),
      }),
    ).toHaveAttribute('href', '/admin/tenant/attributes/new')
  })

  it('shows text-labeled edit/delete buttons instead of icon-only buttons', async () => {
    await renderWithRouterBase(
      <AdminTenantAttributesPage
        csrfToken="csrf"
        schema={{ ...schema, attributes: [attribute] }}
      />,
    )
    expect(
      screen.getByRole('button', {
        name: adminTenantAttributesDictionary.en.editAttributeAria.replace('{key}', 'region'),
      }),
    ).toHaveTextContent(adminTenantAttributesDictionary.en.edit)
    expect(
      screen.getByRole('button', {
        name: adminTenantAttributesDictionary.en.deleteAttributeAria.replace('{key}', 'region'),
      }),
    ).toHaveTextContent(adminTenantAttributesDictionary.en.delete)
  })
})
