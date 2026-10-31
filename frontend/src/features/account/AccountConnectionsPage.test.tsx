import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { TokenVaultMyConnection } from '../../types'
import { AccountConnectionsPage } from './AccountConnectionsPage'
import { accountConnectionsDictionary } from './AccountConnectionsPage.i18n'

const t = accountConnectionsDictionary.en
const base = { csrfToken: 'csrf', username: 'taro', isAdmin: false }
const linked: TokenVaultMyConnection = {
  connector_id: 'c1',
  connector_name: 'Calendar',
  connector_status: 'active',
  status: 'active',
  granted_scopes: ['read'],
}
const originalFetch = globalThis.fetch

afterEach(() => {
  globalThis.fetch = originalFetch
})

function jsonResponse(body: unknown, status = 200) {
  return new Response(body === null ? null : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('AccountConnectionsPage', () => {
  it('offers linking only for a service that is not linked', async () => {
    await renderWithRouter(
      <AccountConnectionsPage
        {...base}
        connections={[
          linked,
          {
            connector_id: 'c2',
            connector_name: 'Mail',
            connector_status: 'active',
            granted_scopes: [],
          },
        ]}
      />,
    )
    expect(screen.getAllByRole('button', { name: t.link })).toHaveLength(1)
    expect(screen.getAllByRole('button', { name: t.unlink })).toHaveLength(1)
  })

  it('offers linking again after the upstream refused the connection', async () => {
    await renderWithRouter(
      <AccountConnectionsPage
        {...base}
        connections={[{ ...linked, status: 'reauthorization_required' }]}
      />,
    )
    expect(screen.getByText(t.statusReauthorization)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.relink })).toBeInTheDocument()
  })

  it('allows unlinking but not linking while the connector is disabled', async () => {
    await renderWithRouter(
      <AccountConnectionsPage
        {...base}
        connections={[
          { ...linked, connector_status: 'disabled', status: 'reauthorization_required' },
        ]}
      />,
    )
    expect(screen.queryByRole('button', { name: t.relink })).toBeNull()
    expect(screen.getByRole('button', { name: t.unlink })).toBeInTheDocument()
  })

  it('shows the callback result without echoing unknown values', async () => {
    await renderWithRouter(<AccountConnectionsPage {...base} connections={[]} result="denied" />)
    expect(screen.getByText(t.resultDenied)).toBeInTheDocument()
  })

  it('navigates to the authorization URL returned by the server', async () => {
    const navigate = mock((_url: string) => {})
    globalThis.fetch = mock(async () =>
      jsonResponse({ authorization_url: 'https://idp.example/authorize?state=s' }),
    ) as unknown as typeof fetch
    await renderWithRouter(
      <AccountConnectionsPage
        {...base}
        navigate={navigate}
        connections={[
          {
            connector_id: 'c2',
            connector_name: 'Mail',
            connector_status: 'active',
            granted_scopes: [],
          },
        ]}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.link }))
    await waitFor(() =>
      expect(navigate).toHaveBeenCalledWith('https://idp.example/authorize?state=s'),
    )
  })

  it('refreshes the list after unlinking', async () => {
    const calls: string[] = []
    globalThis.fetch = mock(async (input: RequestInfo | URL, init?: RequestInit) => {
      calls.push(`${init?.method ?? 'GET'} ${String(input)}`)
      if (init?.method === 'DELETE') return new Response(null, { status: 204 })
      return jsonResponse({ items: [{ ...linked, status: undefined, granted_scopes: [] }] })
    }) as unknown as typeof fetch
    await renderWithRouter(<AccountConnectionsPage {...base} connections={[linked]} />)
    fireEvent.click(screen.getByRole('button', { name: t.unlink }))
    await waitFor(() => expect(screen.getByRole('button', { name: t.link })).toBeInTheDocument())
    expect(
      calls.some(
        (call) =>
          call.startsWith('DELETE') && call.includes('/api/account/v1/token-vault/connections/c1'),
      ),
    ).toBe(true)
  })
})
