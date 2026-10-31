import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { TokenVaultConnector } from '../../types'
import { AdminTokenVaultPage, parseRules } from './AdminTokenVaultPage'
import { adminTokenVaultDictionary } from './AdminTokenVaultPage.i18n'

const t = adminTokenVaultDictionary.en
const connector: TokenVaultConnector = {
  id: 'c1',
  tenant_id: 'tenant',
  name: 'Calendar',
  status: 'active',
  authorization_endpoint: 'https://idp.example/authorize',
  token_endpoint: 'https://idp.example/token',
  client_id: 'client',
  token_endpoint_auth_method: 'client_secret_basic',
  client_secret_configured: true,
  allowed_scopes: ['read'],
  rules: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}
const originalFetch = globalThis.fetch

afterEach(() => {
  globalThis.fetch = originalFetch
})

describe('parseRules', () => {
  it('accepts only a JSON array', () => {
    expect(parseRules('[]')).toEqual([])
    expect(parseRules('{}')).toBeNull()
    expect(parseRules('not json')).toBeNull()
  })
})

describe('AdminTokenVaultPage', () => {
  it('stops a connector through the status update', async () => {
    const calls: { method?: string; body?: unknown }[] = []
    globalThis.fetch = mock(async (_input: RequestInfo | URL, init?: RequestInit) => {
      calls.push({ method: init?.method, body: init?.body })
      const body =
        init?.method === 'PATCH' ? connector : { items: [{ ...connector, status: 'disabled' }] }
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    }) as unknown as typeof fetch
    await renderWithRouter(<AdminTokenVaultPage csrfToken="csrf" connectors={[connector]} />)
    fireEvent.click(screen.getByRole('button', { name: t.disable }))
    await waitFor(() => expect(screen.getByRole('button', { name: t.enable })).toBeInTheDocument())
    expect(calls[0]).toEqual({ method: 'PATCH', body: JSON.stringify({ status: 'disabled' }) })
  })

  it('does not delete without confirmation', async () => {
    const fetcher = mock(async () => new Response(null, { status: 204 }))
    globalThis.fetch = fetcher as unknown as typeof fetch
    await renderWithRouter(
      <AdminTokenVaultPage csrfToken="csrf" connectors={[connector]} confirmDelete={() => false} />,
    )
    fireEvent.click(screen.getByRole('button', { name: t.delete }))
    expect(fetcher).not.toHaveBeenCalled()
  })

  it('refuses rules that are not an array before calling the server', async () => {
    const fetcher = mock(async () => new Response(null, { status: 500 }))
    globalThis.fetch = fetcher as unknown as typeof fetch
    await renderWithRouter(<AdminTokenVaultPage csrfToken="csrf" connectors={[]} />)
    fireEvent.change(screen.getByLabelText(t.rules), { target: { value: '{}' } })
    fireEvent.submit(screen.getByRole('button', { name: t.add }).closest('form') as HTMLFormElement)
    expect(screen.getByText(t.rulesInvalid)).toBeInTheDocument()
    expect(fetcher).not.toHaveBeenCalled()
  })
})
