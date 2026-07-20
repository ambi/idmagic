import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '../../lib/i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminKeysPage, SigningKeyTable } from './AdminKeysPage'
import { adminKeysDictionary } from './AdminKeysPage.i18n'

const t = adminKeysDictionary.en

afterEach(() => vi.unstubAllGlobals())

function renderEn(ui: Parameters<typeof render>[0]) {
  return render(<LocaleProvider initialLocale="en">{ui}</LocaleProvider>)
}

const key = {
  kid: 'kid-1',
  provider: 'Local',
  alg: 'RS256',
  active: true,
  created_at: '2026-01-01T00:00:00Z',
  public_jwk: { kty: 'RSA' },
}

describe('AdminKeysPage', () => {
  it('renders in English by default', async () => {
    await renderWithRouter(
      <AdminKeysPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="tenant-1"
        keys={[key]}
      />,
    )
    expect(
      screen.getByRole('heading', { name: adminKeysDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: adminKeysDictionary.en.rotate })).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminKeysPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="tenant-1"
        keys={[key]}
      />,
      { locale: 'ja' },
    )
    expect(
      screen.getByRole('heading', { name: adminKeysDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: adminKeysDictionary.ja.rotate })).toBeInTheDocument()
  })

  it('keeps a successfully disabled key removed even when the subsequent list request is unavailable', async () => {
    const inactiveKey = { ...key, kid: 'kid-previous', active: false }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: vi.fn().mockResolvedValue(inactiveKey),
      }),
    )
    await renderWithRouter(
      <AdminKeysPage
        csrfToken="csrf"
        actorUsername="admin"
        actorRoles={['admin']}
        actorRealm="tenant-1"
        keys={[key, inactiveKey]}
      />,
    )

    fireEvent.click(
      screen.getByRole('button', { name: t.disableKeyAria.replace('{kid}', inactiveKey.kid) }),
    )
    fireEvent.click(screen.getByRole('button', { name: t.executeDisable }))

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining(`/api/admin/keys/${inactiveKey.kid}/disable`),
        expect.objectContaining({ method: 'POST' }),
      )
    })
    expect(within(screen.getByRole('table')).queryByText(inactiveKey.kid)).not.toBeInTheDocument()
    expect(screen.getByText(t.disabledNotice.replace('{kid}', inactiveKey.kid))).toBeInTheDocument()
  })
})

describe('SigningKeyTable', () => {
  it('notifies selection without exposing destructive actions to non-managers', () => {
    const onSelect = vi.fn()
    renderEn(
      <SigningKeyTable
        keys={[key]}
        canManage={false}
        busy={false}
        onSelect={onSelect}
        onDisable={vi.fn()}
      />,
    )
    fireEvent.click(screen.getByText('kid-1'))
    expect(onSelect).toHaveBeenCalledWith(key)
    expect(
      screen.queryByRole('button', {
        name: t.disableKeyAria.replace('{kid}', 'kid-1'),
      }),
    ).not.toBeInTheDocument()
  })
})
