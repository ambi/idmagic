import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminTenant } from '../../types'
import { SystemTenantCreatePage } from './SystemTenantCreatePage'
import { SystemTenantDetailPage } from './SystemTenantDetailPage'
import { SystemTenantEditPage } from './SystemTenantEditPage'
import { SystemTenantEndpointStylePage } from './SystemTenantEndpointStylePage'
import { SystemTenantQuotaPage } from './SystemTenantQuotaPage'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'

const t = systemTenantsDictionary.en

// realm と内部 ID を変えておき、画面が realm で要求することを区別できるようにする。
const tenant: AdminTenant = {
  id: 'tenant-1',
  realm: 'acme',
  display_name: 'Acme',
  status: 'active',
  endpoint_style: 'path',
  created_at: '2026-01-01T00:00:00Z',
  quota: { users: 100 },
  usage: {
    users: 7,
    groups: 0,
    agents: 0,
    applications: 0,
    oauth2_clients: 0,
    active_sessions: 0,
    consents: 0,
    active_jobs: 0,
    ssf_streams: 0,
    audit_events_retained: 0,
    export_artifacts_bytes: 0,
  },
}

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

function stubNavigation() {
  stubGlobal('location', { ...window.location, assign: mock() })
}

function stubFetch(status: number, body: unknown = {}) {
  const fetchMock = mock(() => Promise.resolve(response(status, body)))
  stubGlobal('fetch', fetchMock)
  return fetchMock
}

function requestAt(fetchMock: ReturnType<typeof stubFetch>, index: number) {
  return fetchMock.mock.calls[index] as unknown as [string, RequestInit]
}

afterEach(() => restoreGlobals())

describe('SystemTenantDetailPage', () => {
  //spec:covers REQ-FRONTEND-003: 詳細は現在の値を読み取り専用で示し、変更ごとの画面への入口を置く。
  it('shows current values without inputs and links to each change screen', async () => {
    await renderWithRouter(<SystemTenantDetailPage csrfToken="csrf" tenant={tenant} />)

    expect(screen.queryByRole('textbox')).toBeNull()
    expect(screen.queryByRole('spinbutton')).toBeNull()
    expect(screen.queryByRole('combobox')).toBeNull()
    expect(document.querySelector('form')).toBeNull()
    expect(screen.getByRole('button', { name: t.editSettings })).toHaveAttribute(
      'href',
      '/system/tenants/acme/edit',
    )
    expect(screen.getByRole('button', { name: t.editQuota })).toHaveAttribute(
      'href',
      '/system/tenants/acme/quota',
    )
    expect(screen.getByRole('button', { name: t.switchEndpointStyle })).toHaveAttribute(
      'href',
      '/system/tenants/acme/endpoint-style',
    )
    const usersRow = screen.getByText(t.quotaUsers).closest('tr')!
    expect(within(usersRow).getByText('7')).toBeInTheDocument()
    expect(within(usersRow).getByText('100')).toBeInTheDocument()
  })

  //spec:covers REQ-FRONTEND-004: 詳細の無効化も確認を経るまで要求しない。
  it('asks for confirmation before disabling from the detail', async () => {
    const fetchMock = stubFetch(204)
    await renderWithRouter(<SystemTenantDetailPage csrfToken="csrf" tenant={tenant} />)

    fireEvent.click(screen.getByRole('button', { name: t.disable }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('acme')
    expect(dialog).toHaveTextContent(t.confirmDisableDescription)
    fireEvent.click(within(dialog).getByRole('button', { name: t.cancel }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(fetchMock).not.toHaveBeenCalled()
  })
})

describe('SystemTenantEditPage', () => {
  //spec:covers REQ-FRONTEND-003: 変更画面の保存は UpdateTenant だけを要求し、通知付きで詳細へ戻る。
  it('saves only the tenant settings and returns to the detail', async () => {
    stubNavigation()
    const fetchMock = stubFetch(200, tenant)
    await renderWithRouter(<SystemTenantEditPage csrfToken="csrf" tenant={tenant} />)

    fireEvent.change(screen.getByLabelText(new RegExp(t.displayNameLabel)), {
      target: { value: 'Acme Corp' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/system/tenants/acme?notice=updated'),
    )
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = requestAt(fetchMock, 0)
    expect(url).toContain('/api/admin/v1/tenants/acme')
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(String(init.body))).toEqual({ display_name: 'Acme Corp' })
  })

  //spec:covers REQ-FRONTEND-003: 保存に失敗したら変更画面と入力を残し、失敗を表示する。
  it('keeps the form and shows the failure when saving fails', async () => {
    stubNavigation()
    stubFetch(500)
    await renderWithRouter(<SystemTenantEditPage csrfToken="csrf" tenant={tenant} />)

    const name = screen.getByLabelText(new RegExp(t.displayNameLabel))
    fireEvent.change(name, { target: { value: 'Acme Corp' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByRole('alert')).toBeInTheDocument()
    expect(name).toHaveValue('Acme Corp')
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})

describe('SystemTenantQuotaPage', () => {
  //spec:covers REQ-TENANCY-046, REQ-FRONTEND-003: 上限の保存は realm を指す UpdateTenantQuota だけを要求する。
  it('saves the quota by realm in one request', async () => {
    stubNavigation()
    const fetchMock = stubFetch(200, { users: 321 })
    await renderWithRouter(<SystemTenantQuotaPage csrfToken="csrf" tenant={tenant} />)

    fireEvent.change(screen.getByLabelText(t.quotaUsers), { target: { value: '321' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/system/tenants/acme?notice=updated'),
    )
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = requestAt(fetchMock, 0)
    expect(url).toContain('/api/admin/v1/tenants/acme/quota')
    expect(init.method).toBe('PUT')
    expect(JSON.parse(String(init.body))).toEqual({ users: 321 })
    expect(new Headers(init.headers).get('X-CSRF-Token')).toBe('csrf')
  })

  //spec:covers WCAG22-LABELS-ERRORS: 整数でない上限を欄のエラー文で示し、保存しないこと
  it('does not save while a quota is not a whole number', () => {
    const fetchMock = stubFetch(200)
    return renderWithRouter(<SystemTenantQuotaPage csrfToken="csrf" tenant={tenant} />).then(() => {
      const users = screen.getByLabelText(t.quotaUsers)
      fireEvent.change(users, { target: { value: '1.5' } })
      expect(users).toHaveAccessibleDescription('Enter a whole number of 0 or more.')
      fireEvent.submit(users.closest('form')!)
      expect(fetchMock).not.toHaveBeenCalled()
    })
  })
})

describe('SystemTenantEndpointStylePage', () => {
  //spec:covers REQ-FRONTEND-004: 正規ロケーションの切替は影響を示す確認を承認するまで要求しない。
  it('switches the endpoint style only after confirmation', async () => {
    stubNavigation()
    const fetchMock = stubFetch(204)
    await renderWithRouter(<SystemTenantEndpointStylePage csrfToken="csrf" tenant={tenant} />)

    fireEvent.change(screen.getByLabelText(t.endpointStyleLabel), {
      target: { value: 'subdomain' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent(t.endpointStyleWarning)
    expect(fetchMock).not.toHaveBeenCalled()

    fireEvent.click(within(dialog).getByRole('button', { name: t.cancel }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(fetchMock).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: t.save }))
    fireEvent.click(await screen.findByRole('button', { name: t.confirmEndpointStyle }))
    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/system/tenants/acme?notice=updated'),
    )
    const [url, init] = requestAt(fetchMock, 0)
    expect(url).toContain('/api/admin/v1/tenants/acme/endpoint-style')
    expect(JSON.parse(String(init.body))).toEqual({ endpoint_style: 'subdomain' })
  })
})

describe('SystemTenantCreatePage', () => {
  //spec:covers WCAG22-LABELS-ERRORS: 契約の文法に合わない realm を欄のエラー文で示し、作成しないこと
  it.each(['acme-', '-acme', 'Acme', `a${'b'.repeat(63)}`])(
    'rejects the realm %s',
    async (value) => {
      await renderWithRouter(<SystemTenantCreatePage csrfToken="csrf" />)
      const realm = screen.getByLabelText(new RegExp(t.realmLabel))
      fireEvent.change(realm, { target: { value } })
      expect(realm).toHaveAccessibleDescription(t.realmInvalidError)
      expect(screen.getByRole('button', { name: t.create })).toBeDisabled()
    },
  )

  //spec:covers REQ-FRONTEND-003: 作成に成功したら作成したテナントの詳細へ通知付きで移る。
  it('opens the created tenant detail', async () => {
    stubNavigation()
    stubFetch(201, { ...tenant, realm: 'beta' })
    await renderWithRouter(<SystemTenantCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(new RegExp(t.realmLabel)), { target: { value: 'beta' } })
    fireEvent.change(screen.getByLabelText(new RegExp(t.displayNameLabel)), {
      target: { value: 'Beta' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/system/tenants/beta?notice=created'),
    )
  })
})
