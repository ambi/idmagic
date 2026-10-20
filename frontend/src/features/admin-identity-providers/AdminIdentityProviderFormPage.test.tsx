import { afterEach, expect, it, mock } from 'bun:test'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import type { IdentityProviderConnection } from '../../api'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminIdentityProviderEditPage } from './AdminIdentityProviderFormPage'
import { identityProvidersDictionary } from './AdminIdentityProvidersPage.i18n'

const t = identityProvidersDictionary.en
const connection: IdentityProviderConnection = {
  id: 'preview-provider',
  tenant_id: 'default',
  display_name: 'Preview provider',
  protocol: 'oidc',
  status: 'disabled',
  issuer: 'https://preview.example',
  client_id: 'preview-client',
  client_secret_configured: false,
  authorization_endpoint: 'https://preview.example/original-auth',
  token_endpoint: 'https://preview.example/original-token',
  jwks_uri: 'https://preview.example/original-jwks',
  claim_mapping: { subject: 'sub', username: 'email' },
  linking_policy: 'none',
  jit_provisioning: false,
  created_at: '2026-10-21T00:00:00Z',
  updated_at: '2026-10-21T00:00:00Z',
}
const candidates = {
  issuer: connection.issuer,
  authorization_endpoint: 'https://preview.example/new-auth',
  token_endpoint: 'https://preview.example/new-token',
  jwks_uri: 'https://preview.example/new-jwks',
}
const response = (body: unknown) => ({ ok: true, status: 200, json: async () => body })
afterEach(restoreGlobals)

//spec:covers REQ-AUTHENTICATION-041: 候補の取得、フォームへの適用と保存は別の操作とする。
it('取得した候補は適用と保存を明示するまで接続を書き換えない', async () => {
  const fetchMock = mock((url: string, init?: RequestInit) => {
    if (url.endsWith('/metadata-preview')) return Promise.resolve(response(candidates))
    if (init?.method === 'PUT') return new Promise<Response>(() => {})
    throw new Error(`unexpected request ${url}`)
  })
  stubGlobal('fetch', fetchMock)
  await renderWithRouter(<AdminIdentityProviderEditPage csrfToken="csrf" connection={connection} />)
  fireEvent.click(screen.getByRole('button', { name: 'Fetch metadata' }))
  await screen.findByRole('button', { name: 'Apply candidates' })
  expect(screen.getByLabelText(t.authorizationEndpoint)).toHaveValue(
    connection.authorization_endpoint,
  )
  expect(fetchMock).toHaveBeenCalledTimes(1)
  expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).toEqual({
    issuer: connection.issuer,
  })
  fireEvent.click(screen.getByRole('button', { name: 'Apply candidates' }))
  expect(screen.getByLabelText(t.authorizationEndpoint)).toHaveValue(
    candidates.authorization_endpoint,
  )
  expect(screen.getByLabelText(t.tokenEndpoint)).toHaveValue(candidates.token_endpoint)
  expect(screen.getByLabelText(t.jwksUri)).toHaveValue(candidates.jwks_uri)
  expect(fetchMock).toHaveBeenCalledTimes(1)
  fireEvent.submit(screen.getByRole('button', { name: t.update }).closest('form')!)
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  const [url, init] = fetchMock.mock.calls[1]
  expect(url).toContain(`/identity-providers/${connection.id}`)
  expect(init?.method).toBe('PUT')
  expect(JSON.parse(String(init?.body))).toMatchObject(candidates)
})

//spec:covers REQ-AUTHENTICATION-041: 取得中に issuer が変わったとき、古い応答をフォームへ適用できない。
it('取得中に issuer が変わったら古い候補を破棄する', async () => {
  let resolvePreview!: (value: ReturnType<typeof response>) => void
  stubGlobal(
    'fetch',
    mock(
      () =>
        new Promise<ReturnType<typeof response>>((resolve) => {
          resolvePreview = resolve
        }),
    ),
  )
  await renderWithRouter(<AdminIdentityProviderEditPage csrfToken="csrf" connection={connection} />)
  fireEvent.click(screen.getByRole('button', { name: 'Fetch metadata' }))
  fireEvent.change(screen.getByLabelText(t.issuer), {
    target: { value: 'https://replacement.example' },
  })
  resolvePreview(response(candidates))
  await waitFor(() =>
    expect(screen.getByRole('button', { name: 'Fetch metadata' })).not.toBeDisabled(),
  )
  expect(screen.queryByRole('button', { name: 'Apply candidates' })).not.toBeInTheDocument()
  expect(screen.getByLabelText(t.authorizationEndpoint)).toHaveValue(
    connection.authorization_endpoint,
  )
})

//spec:covers REQ-AUTHENTICATION-041: 取得後のプロトコルの切り替えは候補を破棄し、OIDC に戻しても再利用しない。
it('取得後にプロトコルを変えたら候補を破棄する', async () => {
  stubGlobal(
    'fetch',
    mock(() => Promise.resolve(response(candidates))),
  )
  await renderWithRouter(<AdminIdentityProviderEditPage csrfToken="csrf" connection={connection} />)
  fireEvent.click(screen.getByRole('button', { name: 'Fetch metadata' }))
  await screen.findByRole('button', { name: 'Apply candidates' })
  fireEvent.change(screen.getByLabelText(t.protocol), { target: { value: 'saml' } })
  expect(screen.queryByRole('button', { name: 'Apply candidates' })).not.toBeInTheDocument()
  fireEvent.change(screen.getByLabelText(t.protocol), { target: { value: 'oidc' } })
  expect(screen.queryByRole('button', { name: 'Apply candidates' })).not.toBeInTheDocument()
  expect(screen.getByLabelText(t.authorizationEndpoint)).toHaveValue(
    connection.authorization_endpoint,
  )
})
