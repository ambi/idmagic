import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '../../lib/i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import { SystemTenantsPage, TenantTable } from './SystemTenantsPage'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'

const t = systemTenantsDictionary.en

const tenant = {
  id: 'tenant-1',
  realm: 'acme',
  display_name: 'Acme',
  status: 'active' as const,
  created_at: '2026-01-01T00:00:00Z',
}

function renderEn(ui: Parameters<typeof render>[0]) {
  return render(<LocaleProvider initialLocale="en">{ui}</LocaleProvider>)
}

describe('locale', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('renders the tenants page in English by default', async () => {
    await renderWithRouter(
      <SystemTenantsPage csrfToken="csrf" actorUsername="admin" tenants={[]} />,
    )
    expect(screen.getByRole('heading', { name: t.pageTitle })).toBeInTheDocument()
    expect(screen.getByText(t.selectTenantPrompt)).toBeInTheDocument()
  })

  it('renders the tenants page in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <SystemTenantsPage csrfToken="csrf" actorUsername="admin" tenants={[]} />,
      { locale: 'ja' },
    )
    expect(
      screen.getByRole('heading', { name: systemTenantsDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('TenantTable', () => {
  it('keeps the default tenant protected and forwards a selected tenant', () => {
    const onSelect = vi.fn()
    renderEn(
      <TenantTable
        tenants={[tenant, { ...tenant, id: 'default', realm: 'default' }]}
        busy={false}
        onSelect={onSelect}
        onToggleDisabled={vi.fn()}
      />,
    )
    fireEvent.click(screen.getByText('acme'))
    expect(onSelect).toHaveBeenCalledWith(tenant)
    expect(screen.getByRole('button', { name: t.disable })).toBeInTheDocument()
  })
})
