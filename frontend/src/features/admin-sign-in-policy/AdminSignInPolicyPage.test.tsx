import { describe, it, expect } from 'bun:test'
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

  //spec:covers EX-APPLICATION-010-01: 画面は MFA 未登録の有効なユーザー数と、強制開始までに
  // 何を用意する必要があるかを示す。未登録が 0 人ならその案内は出さない。
  it('shows how many active users have not enrolled in MFA and what enforcement needs', async () => {
    await renderWithRouterBase(
      <AdminSignInPolicyPage csrfToken="csrf" policy={policy} apps={[]} unenrolledUserCount={2} />,
    )
    expect(
      screen.getByText(adminSignInPolicyDictionary.en.unenrolledWarning.replace('{count}', '2')),
    ).toBeInTheDocument()
  })

  it('omits the enrollment impact notice when every active user has enrolled', async () => {
    await renderWithRouterBase(
      <AdminSignInPolicyPage csrfToken="csrf" policy={policy} apps={[]} unenrolledUserCount={0} />,
    )
    expect(
      screen.queryByText(adminSignInPolicyDictionary.en.unenrolledWarning.replace('{count}', '0')),
    ).not.toBeInTheDocument()
  })
})
