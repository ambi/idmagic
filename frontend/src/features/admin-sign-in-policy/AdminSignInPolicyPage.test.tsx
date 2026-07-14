import { describe, it, expect } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import { AdminSignInPolicyPage } from './AdminSignInPolicyPage'
import { adminSignInPolicyDictionary } from './AdminSignInPolicyPage.i18n'
import type { TenantDefaultSignInPolicy } from '../../types'

const policy: TenantDefaultSignInPolicy = {
  tenant_id: 'tenant-1',
  rules: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui, { locale: 'ja' })

describe('AdminSignInPolicyPage', () => {
  it('renders in English by default', async () => {
    await renderWithRouterBase(
      <AdminSignInPolicyPage csrfToken="csrf" policy={policy} apps={[]} unenrolledUserCount={0} />,
    )
    expect(
      screen.getByRole('heading', { name: adminSignInPolicyDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminSignInPolicyDictionary.en.noAppsNotice)).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminSignInPolicyPage csrfToken="csrf" policy={policy} apps={[]} unenrolledUserCount={2} />,
    )
    expect(
      screen.getByRole('heading', { name: adminSignInPolicyDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminSignInPolicyDictionary.ja.noAppsNotice)).toBeInTheDocument()
  })
})
