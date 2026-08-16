import { fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import type { WorkloadTrustBundle } from '../../types'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'
import { AdminWorkloadTrustBundleCreatePage } from './AdminWorkloadTrustBundleCreatePage'
import { AdminWorkloadTrustBundleEditPage } from './AdminWorkloadTrustBundleEditPage'

const t = adminWorkloadIdentityDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
  headers: new Headers(),
})

const bundle: WorkloadTrustBundle = {
  id: 'bundle-1',
  tenant_id: 'tenant-a',
  name: 'prod-cluster',
  trust_domain: 'example.org',
  issuer: 'https://issuer.example',
  jwks_uri: 'https://issuer.example/keys',
  has_inline_jwks: false,
  accepted_audiences: ['idmagic', 'https://idmagic.example'],
  max_subject_token_ttl_seconds: 900,
  status: 'enabled',
  created_at: '2026-08-16T00:00:00Z',
}

function fill(label: string, value: string) {
  fireEvent.change(screen.getByLabelText(label), { target: { value } })
}

describe('AdminWorkloadTrustBundleCreatePage', () => {
  afterEach(() => restoreGlobals())

  it('registers a bundle with the audiences split out of one field', async () => {
    const fetchMock = mock((_url: string, _init: RequestInit) =>
      Promise.resolve(response(201, bundle)),
    )
    stubGlobal('fetch', fetchMock)
    stubGlobal('location', { assign: mock(), pathname: '/', search: '' })
    await renderWithRouter(<AdminWorkloadTrustBundleCreatePage csrfToken="csrf" />)

    fill(t.nameLabel, 'prod-cluster')
    fill(t.trustDomainLabel, 'example.org')
    fill(t.issuerLabel, 'https://issuer.example')
    fill(t.jwksUriLabel, 'https://issuer.example/keys')
    fill(t.acceptedAudiencesLabel, 'idmagic, https://idmagic.example')
    fireEvent.submit(screen.getByRole('button', { name: t.register }).closest('form')!)

    await screen.findByRole('button', { name: t.register })
    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(String(init.body))).toEqual({
      name: 'prod-cluster',
      trust_domain: 'example.org',
      issuer: 'https://issuer.example',
      jwks_uri: 'https://issuer.example/keys',
      accepted_audiences: ['idmagic', 'https://idmagic.example'],
      max_subject_token_ttl_seconds: 3600,
    })
  })

  it('refuses to send an inline JWKS that is not readable as JSON', async () => {
    const fetchMock = mock((_url: string, _init: RequestInit) =>
      Promise.resolve(response(201, bundle)),
    )
    stubGlobal('fetch', fetchMock)
    await renderWithRouter(<AdminWorkloadTrustBundleCreatePage csrfToken="csrf" />)

    fill(t.nameLabel, 'prod-cluster')
    fill(t.jwksLabel, '{not json')
    fireEvent.submit(screen.getByRole('button', { name: t.register }).closest('form')!)

    expect(await screen.findByText(t.jwksInvalidError)).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })
})

describe('AdminWorkloadTrustBundleEditPage', () => {
  afterEach(() => restoreGlobals())

  it('locks the issuer and trust domain, which the update API cannot change', async () => {
    await renderWithRouter(
      <AdminWorkloadTrustBundleEditPage csrfToken="csrf" trustBundle={bundle} />,
    )

    expect(screen.getByLabelText(t.issuerLabel)).toBeDisabled()
    expect(screen.getByLabelText(t.trustDomainLabel)).toBeDisabled()
    expect(screen.getByLabelText(t.nameLabel)).not.toBeDisabled()
  })

  it('starts from the stored values, with the inline JWKS left blank because it is never returned', async () => {
    await renderWithRouter(
      <AdminWorkloadTrustBundleEditPage csrfToken="csrf" trustBundle={bundle} />,
    )

    expect(screen.getByLabelText(t.acceptedAudiencesLabel)).toHaveValue(
      'idmagic https://idmagic.example',
    )
    expect(screen.getByLabelText(t.maxTtlLabel)).toHaveValue(900)
    expect(screen.getByLabelText(t.jwksLabel)).toHaveValue('')
  })

  it('leaves an untouched JWKS URI out of the update, so an inline-only bundle keeps its keys', async () => {
    // Go 側は jwks_uri が非 nil なら空文字でも上書きする。名前だけ直したつもりの更新で
    // インライン JWKS のバンドルから取得元設定が消えないよう、変更が無ければ送らない。
    const fetchMock = mock((_url: string, _init: RequestInit) =>
      Promise.resolve(response(200, bundle)),
    )
    stubGlobal('fetch', fetchMock)
    stubGlobal('location', { assign: mock(), pathname: '/', search: '' })
    await renderWithRouter(
      <AdminWorkloadTrustBundleEditPage
        csrfToken="csrf"
        trustBundle={{ ...bundle, jwks_uri: undefined, has_inline_jwks: true }}
      />,
    )

    fill(t.nameLabel, 'prod-cluster-2')
    fireEvent.submit(screen.getByRole('button', { name: t.update }).closest('form')!)

    await screen.findByRole('button', { name: t.update })
    const [, init] = fetchMock.mock.calls[0]
    expect('jwks_uri' in JSON.parse(String(init.body))).toBe(false)
  })

  it('sends an emptied JWKS URI so the operator can actually clear it', async () => {
    const fetchMock = mock((_url: string, _init: RequestInit) =>
      Promise.resolve(response(200, bundle)),
    )
    stubGlobal('fetch', fetchMock)
    stubGlobal('location', { assign: mock(), pathname: '/', search: '' })
    await renderWithRouter(
      <AdminWorkloadTrustBundleEditPage csrfToken="csrf" trustBundle={bundle} />,
    )

    fill(t.jwksUriLabel, '')
    fireEvent.submit(screen.getByRole('button', { name: t.update }).closest('form')!)

    await screen.findByRole('button', { name: t.update })
    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(String(init.body)).jwks_uri).toBe('')
  })

  it('omits the inline JWKS from the update when the field was left blank', async () => {
    const fetchMock = mock((_url: string, _init: RequestInit) =>
      Promise.resolve(response(200, bundle)),
    )
    stubGlobal('fetch', fetchMock)
    stubGlobal('location', { assign: mock(), pathname: '/', search: '' })
    await renderWithRouter(
      <AdminWorkloadTrustBundleEditPage csrfToken="csrf" trustBundle={bundle} />,
    )

    fill(t.nameLabel, 'prod-cluster-2')
    fireEvent.submit(screen.getByRole('button', { name: t.update }).closest('form')!)

    await screen.findByRole('button', { name: t.update })
    const [, init] = fetchMock.mock.calls[0]
    const body = JSON.parse(String(init.body))
    expect(body.name).toBe('prod-cluster-2')
    expect('jwks' in body).toBe(false)
  })
})
