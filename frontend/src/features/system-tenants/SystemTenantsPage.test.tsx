import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { SystemTenantsPage } from './SystemTenantsPage'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'

const t = systemTenantsDictionary.en

const tenant = {
  id: 'tenant-1',
  realm: 'acme',
  display_name: 'Acme',
  status: 'active' as const,
  created_at: '2026-01-01T00:00:00Z',
}

const defaultTenant = { ...tenant, id: 'default-id', realm: 'default', display_name: 'Default' }

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('locale', () => {
  it('renders the tenants page in English by default', async () => {
    await renderWithRouter(
      <SystemTenantsPage csrfToken="csrf" actorUsername="admin" tenants={[]} />,
    )
    expect(screen.getByRole('heading', { name: t.pageTitle })).toBeInTheDocument()
    expect(screen.getByText(t.emptyTenantList)).toBeInTheDocument()
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

describe('SystemTenantsPage rows', () => {
  //spec:covers REQ-FRONTEND-003: 一覧の行は詳細と変更画面へ移る入口を示し、新規作成は専用画面へ移る。
  it('links each row to its detail and edit screens and creation to its own screen', async () => {
    await renderWithRouter(<SystemTenantsPage csrfToken="csrf" tenants={[tenant, defaultTenant]} />)

    const row = screen.getByText('acme').closest('tr')!
    expect(within(row).getByRole('button', { name: `${t.viewDetails}: acme` })).toHaveAttribute(
      'href',
      '/system/tenants/acme',
    )
    expect(within(row).getByRole('button', { name: `${t.edit}: acme` })).toHaveAttribute(
      'href',
      '/system/tenants/acme/edit',
    )
    expect(screen.getByRole('button', { name: t.newTenant })).toHaveAttribute(
      'href',
      '/system/new-tenant',
    )
    expect(screen.queryByRole('textbox')).toBeNull()
  })

  it('offers no status action for the default tenant', async () => {
    await renderWithRouter(<SystemTenantsPage csrfToken="csrf" tenants={[defaultTenant]} />)
    expect(screen.queryByRole('button', { name: t.disable })).toBeNull()
  })
})

describe('SystemTenantsPage status actions', () => {
  afterEach(() => restoreGlobals())

  //spec:covers REQ-FRONTEND-004: 行の無効化は確認を承認するまで要求せず、取り消せば要求しない。
  it('sends the disable request only after the operator confirms', async () => {
    const fetchMock = mock((url: string) =>
      Promise.resolve(
        url.endsWith('/api/admin/v1/tenants')
          ? response(200, { tenants: [{ ...tenant, status: 'disabled' }] })
          : response(204),
      ),
    )
    stubGlobal('fetch', fetchMock)
    await renderWithRouter(<SystemTenantsPage csrfToken="csrf" tenants={[tenant]} />)

    fireEvent.click(screen.getByRole('button', { name: t.disable }))
    expect(fetchMock).not.toHaveBeenCalled()
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('acme')
    fireEvent.click(within(dialog).getByRole('button', { name: t.cancel }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(fetchMock).not.toHaveBeenCalled()
    expect(screen.getByText('active')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: t.disable }))
    fireEvent.click(await screen.findByRole('button', { name: t.confirmDisable }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/admin/v1/tenants/acme/disable')
    expect(init.method).toBe('POST')
    expect(new Headers(init.headers).get('X-CSRF-Token')).toBe('csrf')
    expect(await screen.findByText('disabled')).toBeInTheDocument()
  })

  //spec:covers REQ-FRONTEND-001: 再取得に失敗しても直前の一覧を残し、失敗を表示する。
  it('keeps the previous list when reloading fails', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(500))),
    )
    await renderWithRouter(<SystemTenantsPage csrfToken="csrf" tenants={[tenant]} />)

    fireEvent.click(screen.getByRole('button', { name: t.reloadListAriaLabel }))
    expect(await screen.findByText(t.tenantListFailedError)).toBeInTheDocument()
    expect(screen.getByText('acme')).toBeInTheDocument()
  })
})
