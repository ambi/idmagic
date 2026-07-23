import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminSettingsPage } from './AdminSettingsPage'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import type { AdminSettings } from '../../types'

const t = adminSettingsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

const settings: AdminSettings = {
  tenant_id: 'tenant-1',
  realm: 'acme',
  display_name: 'Acme',
  password_policy_defaults: { min_length: 8, max_length: 64, history_depth: 5 },
}

describe('locale', () => {
  afterEach(() => vi.unstubAllGlobals())

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
  afterEach(() => vi.unstubAllGlobals())

  it('updates the display name and shows a success notice', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(200, { ...settings, display_name: 'Acme Renamed' }))),
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
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(400, { message: 'Could not update the name.' }))),
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

  it('keeps a contextual heading and distinguishes the issued token list', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, { tokens: [] })))
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
