import { act, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { LocaleProvider } from '../../lib/i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminKeysPage, SigningKeyTable } from './AdminKeysPage'
import { adminKeysDictionary } from './AdminKeysPage.i18n'

const t = adminKeysDictionary.en

afterEach(() => restoreGlobals())

function renderEn(ui: Parameters<typeof render>[0]) {
  return render(<LocaleProvider initialLocale="en">{ui}</LocaleProvider>)
}

const key = {
  kid: 'kid-1',
  provider: 'local',
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
    const deletion = Promise.withResolvers<Response>()
    const fetchMock = mock((input: RequestInfo | URL, init?: RequestInit) => {
      const path = new URL(input instanceof Request ? input.url : input, window.location.href)
        .pathname
      const method = init?.method ?? 'GET'
      if (path.endsWith('/api/branding') && method === 'GET') {
        return Promise.resolve(Response.json({}))
      }
      if (path.endsWith(`/api/admin/v1/keys/${inactiveKey.kid}/disable`) && method === 'POST') {
        return deletion.promise
      }
      if (path.endsWith('/api/admin/v1/keys') && method === 'GET') {
        return Promise.resolve(Response.json({ message: 'Unavailable' }, { status: 503 }))
      }
      throw new Error(`Unexpected request: ${method} ${path}`)
    })
    stubGlobal('fetch', fetchMock)
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
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: t.executeDisable }))
    })

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining(`/api/admin/v1/keys/${inactiveKey.kid}/disable`),
      expect.objectContaining({ method: 'POST' }),
    )
    const table = within(screen.getByRole('table'))
    expect(table.getByText(inactiveKey.kid)).toBeInTheDocument()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()

    // 応答の到着と React の更新を確定させてから観測し、消える通知を実時間でポーリングしない。
    await act(async () => {
      deletion.resolve(Response.json(inactiveKey))
    })
    expect(table.queryByText(inactiveKey.kid)).not.toBeInTheDocument()
    expect(table.getByText(key.kid)).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent(
      t.disabledNotice.replace('{kid}', inactiveKey.kid),
    )
  })
})

describe('SigningKeyTable', () => {
  it('notifies selection without exposing destructive actions to non-managers', () => {
    const onSelect = mock()
    renderEn(
      <SigningKeyTable
        keys={[key]}
        canManage={false}
        busy={false}
        onSelect={onSelect}
        onDisable={mock()}
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
