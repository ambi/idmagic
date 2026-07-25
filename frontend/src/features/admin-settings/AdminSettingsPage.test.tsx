import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminSettingsPage } from './AdminSettingsPage'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { notificationTemplatesTabDictionary } from './NotificationTemplatesTab.i18n'
import type { AdminSettings } from '../../types'

const t = adminSettingsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const settings: AdminSettings = {
  tenant_id: 'tenant-1',
  realm: 'acme',
  display_name: 'Acme',
  password_policy_defaults: { min_length: 8, max_length: 64, history_depth: 5 },
  supported_locales: ['ja', 'en'],
}

describe('locale', () => {
  afterEach(() => restoreGlobals())

  it('renders the settings page in English by default', async () => {
    await renderWithRouter(
      <AdminSettingsPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="acme"
        settings={settings}
      />,
    )
    expect(screen.getByRole('heading', { name: t.pageTitle })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.tabGeneralLabel })).toBeInTheDocument()
  })

  it('renders the settings page in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminSettingsPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="acme"
        settings={settings}
      />,
      { locale: 'ja' },
    )
    expect(
      screen.getByRole('heading', { name: adminSettingsDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('AdminSettingsPage', () => {
  afterEach(() => restoreGlobals())

  it('updates the display name and shows a success notice', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { ...settings, display_name: 'Acme Renamed' }))),
    )
    await renderWithRouter(
      <AdminSettingsPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="acme"
        settings={settings}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    fireEvent.change(screen.getByLabelText(t.displayNameLabel), {
      target: { value: 'Acme Renamed' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText(t.displayNameUpdatedNotice)).toBeInTheDocument()
  })

  it('shows an error when updating the display name fails', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(400, { message: 'Could not update the name.' }))),
    )
    await renderWithRouter(
      <AdminSettingsPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="acme"
        settings={settings}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    fireEvent.change(screen.getByLabelText(t.displayNameLabel), {
      target: { value: 'Acme Renamed' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not update the name.')).toBeInTheDocument()
  })

  it('switches to the password policy tab and shows the effective values', async () => {
    await renderWithRouter(
      <AdminSettingsPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="acme"
        settings={settings}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.tabPasswordPolicyLabel }))

    expect(screen.getByRole('heading', { name: t.passwordPolicyHeading })).toBeInTheDocument()
    expect(screen.getAllByText('8 chars').length).toBeGreaterThan(0)
  })

  // 通知メールタブは wi-288 で実装済み。「近日公開」の無効タブのままにしない。
  it('opens the notification template catalog from the email tab', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(200, { templates: [], supported_locales: ['ja', 'en'] })),
    )
    await renderWithRouter(
      <AdminSettingsPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="acme"
        settings={settings}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.tabEmailLabel }))

    expect(
      await screen.findByRole('heading', { name: notificationTemplatesTabDictionary.en.heading }),
    ).toBeInTheDocument()
  })

  // テナント既定 locale は通知の locale 解決の第 2 段 (ADR-142 決定 7)。
  it('updates the tenant default locale from the general tab', async () => {
    const fetch = mock().mockResolvedValue(response(200, { ...settings, default_locale: 'ja' }))
    stubGlobal('fetch', fetch)
    await renderWithRouter(
      <AdminSettingsPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="acme"
        settings={settings}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    fireEvent.change(screen.getByLabelText(t.defaultLocaleLabel), { target: { value: 'ja' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(1))
    const [url, init] = (fetch as any).mock.calls[0]
    expect(url).toBe('/api/admin/settings')
    expect(JSON.parse(init.body)).toEqual({ default_locale: 'ja' })
  })

  it('keeps a contextual heading and distinguishes the issued token list', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(200, { tokens: [] })))
    await renderWithRouter(
      <AdminSettingsPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="acme"
        settings={settings}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.tabApiTokensLabel }))

    await screen.findByText(t.noTokensNotice)
    expect(screen.getByRole('heading', { level: 2, name: t.tabApiTokensLabel })).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { level: 3, name: t.apiTokensListHeading }),
    ).toBeInTheDocument()
  })
})
