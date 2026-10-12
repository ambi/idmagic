import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { LocaleProvider } from '../../lib/i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import { SystemTenantsPage, TenantDetailCard, TenantTable } from './SystemTenantsPage'
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
  afterEach(() => restoreGlobals())

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
    const onSelect = mock()
    renderEn(
      <TenantTable
        tenants={[tenant, { ...tenant, id: 'default', realm: 'default' }]}
        busy={false}
        onSelect={onSelect}
        onToggleDisabled={mock()}
      />,
    )
    fireEvent.click(screen.getByText('acme'))
    expect(onSelect).toHaveBeenCalledWith(tenant)
    expect(screen.getByRole('button', { name: t.disable })).toBeInTheDocument()
  })
})

describe('TenantDetailCard', () => {
  afterEach(() => restoreGlobals())

  //spec:covers REQ-TENANCY-046: クォータの保存要求は内部 ID ではなく realm を送り、方式、本文と CSRF を維持する。
  it('saves the quota by realm while retaining the internal selection ID', async () => {
    const fetchMock = mock().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ users: 321 }),
    })
    stubGlobal('fetch', fetchMock)
    const onSaved = mock()
    renderEn(<TenantDetailCard tenant={tenant} csrfToken="csrf" busy={false} onSaved={onSaved} />)
    fireEvent.change(screen.getByLabelText(t.quotaUsers), { target: { value: '321' } })
    fireEvent.submit(screen.getByLabelText(t.quotaUsers).closest('form')!)
    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(tenant.id))
    expect(fetchMock).toHaveBeenCalledTimes(2)
    const [url, init] = fetchMock.mock.calls[1] as unknown as [string, RequestInit]
    expect(url).toContain('/api/admin/v1/tenants/acme/quota')
    expect(init.method).toBe('PUT')
    expect(JSON.parse(String(init.body))).toEqual({ users: 321 })
    expect(new Headers(init.headers).get('X-CSRF-Token')).toBe('csrf')
  })
})
