import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, mock } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminAgent } from '../../types'
import { AdminAgentCredentialsPage } from './AdminAgentCredentialsPage'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'

const originalFetch = globalThis.fetch
afterEach(() => {
  globalThis.fetch = originalFetch
})
const t = adminAgentsDictionary.en
const agent: AdminAgent = {
  id: 'a1',
  tenant_id: 't',
  name: 'Assistant',
  owner_user_id: 'u1',
  kind: 'autonomous',
  status: 'active',
  client_ids: [],
  created_at: '',
}

//spec:covers REQ-IDMANAGEMENT-104: 束縛の失敗時に作成済み ID を保持し、作成を繰り返さず束縛だけを再試行する。
it('retries binding without creating another client', async () => {
  let created = 0
  let bound = 0
  globalThis.fetch = mock(async (input: RequestInfo | URL) => {
    const url = String(input)
    if (url.endsWith('/clients')) {
      created++
      return Response.json({ client: { client_id: 'created-client' } }, { status: 201 })
    }
    if (url.includes('/credentials')) {
      bound++
      return bound === 1
        ? Response.json(
            { error: 'binding_failed', error_description: 'Try again' },
            { status: 409 },
          )
        : new Response(null, { status: 204 })
    }
    return Response.json({ ...agent, client_ids: ['created-client'] })
  }) as unknown as typeof fetch
  await renderWithRouter(<AdminAgentCredentialsPage agent={agent} csrfToken="csrf" />)
  expect(screen.queryByLabelText(t.bindClientIdAria) === null).toBe(true)
  fireEvent.change(screen.getByLabelText(t.publicKeyLabel), {
    target: { value: JSON.stringify({ kty: 'RSA', n: 'test', e: 'AQAB' }) },
  })
  fireEvent.click(screen.getByRole('button', { name: t.createKeyCredential }))
  await waitFor(() =>
    expect(screen.getByLabelText(t.bindClientIdAria)).toHaveValue('created-client'),
  )
  fireEvent.click(screen.getByRole('button', { name: t.bind }))
  await waitFor(() =>
    expect(
      screen.getByText('Added to this agent. It is also shown in the list below.'),
    ).toBeInTheDocument(),
  )
  expect(created).toBe(1)
  expect(bound).toBe(2)
})

//spec:covers REQ-OAUTH2-089: SPIFFE の JWT-SVID を選ぶと、鍵を持たない spiffe_jwt のクライアントを作成してエージェントに紐付け、信頼設定への案内を示す。
it('creates and binds a spiffe_jwt client without a key', async () => {
  const requests: { url: string; body: unknown }[] = []
  globalThis.fetch = mock(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input)
    requests.push({ url, body: init?.body ? JSON.parse(String(init.body)) : undefined })
    if (url.endsWith('/clients')) {
      return Response.json({ client: { client_id: 'spiffe-client' } }, { status: 201 })
    }
    if (url.includes('/credentials')) {
      return new Response(null, { status: 204 })
    }
    return Response.json({ ...agent, client_ids: ['spiffe-client'] })
  }) as unknown as typeof fetch
  await renderWithRouter(<AdminAgentCredentialsPage agent={agent} csrfToken="csrf" />)
  fireEvent.click(screen.getByLabelText(t.credentialKindSpiffe))
  expect(screen.queryByLabelText(t.publicKeyLabel) === null).toBe(true)
  expect(screen.getByRole('link', { name: t.spiffeTrustLink })).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: t.createKeyCredential }))
  await waitFor(() =>
    expect(
      screen.getByText('Added to this agent. It is also shown in the list below.'),
    ).toBeInTheDocument(),
  )
  const created = requests.find((request) => request.url.endsWith('/clients'))?.body as Record<
    string,
    unknown
  >
  expect(created.token_endpoint_auth_method).toBe('spiffe_jwt')
  expect('jwks' in created || 'jwks_uri' in created).toBe(false)
  expect(requests.some((request) => request.url.endsWith('/agents/a1/credentials'))).toBe(true)
})
