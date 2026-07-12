import { describe, it, expect } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import { AdminAuthorizationDetailTypesPage } from './AdminAuthorizationDetailTypesPage'
import { adminAuthorizationDetailTypesDictionary } from './AdminAuthorizationDetailTypesPage.i18n'
import type { AuthorizationDetailType } from '../../types'

const detailType: AuthorizationDetailType = {
  tenant_id: 'tenant-1',
  type: 'payment_initiation',
  description: 'Payment initiation',
  schema: { rules: [{ name: 'actions', semantics: 'set', required: true }] },
  display_template: 'Up to {instructedAmount}',
  state: 'Enabled',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui, { locale: 'ja' })

describe('AdminAuthorizationDetailTypesPage', () => {
  it('renders in English by default', async () => {
    await renderWithRouterBase(
      <AdminAuthorizationDetailTypesPage csrfToken="csrf" types={[detailType]} />,
    )
    expect(
      screen.getByRole('heading', { name: adminAuthorizationDetailTypesDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: adminAuthorizationDetailTypesDictionary.en.registerType }),
    ).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminAuthorizationDetailTypesPage csrfToken="csrf" types={[detailType]} />,
    )
    expect(
      screen.getByRole('heading', { name: adminAuthorizationDetailTypesDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: adminAuthorizationDetailTypesDictionary.ja.registerType }),
    ).toBeInTheDocument()
  })

  it('shows an empty state when no types are registered', async () => {
    await renderWithRouterBase(<AdminAuthorizationDetailTypesPage csrfToken="csrf" types={[]} />)
    expect(
      screen.getByText(adminAuthorizationDetailTypesDictionary.en.emptyNotice),
    ).toBeInTheDocument()
  })
})
