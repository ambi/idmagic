import { screen } from '@testing-library/react'
import { describe, expect, it } from 'bun:test'
import type { AdminAuditEvent, WorkloadTrustBundle } from '../../types'
import { renderWithRouter } from '../../test/renderWithRouter'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'
import { AdminWorkloadTrustBundlesPage } from './AdminWorkloadTrustBundlesPage'

const t = adminWorkloadIdentityDictionary.en

const bundle: WorkloadTrustBundle = {
  id: 'bundle-1',
  tenant_id: 'tenant-a',
  name: 'prod-cluster',
  trust_domain: 'example.org',
  issuer: 'https://issuer.example',
  jwks_uri: 'https://issuer.example/keys',
  has_inline_jwks: false,
  accepted_audiences: ['idmagic'],
  max_subject_token_ttl_seconds: 3600,
  status: 'enabled',
  created_at: '2026-08-16T00:00:00Z',
  jwks_cached_at: '2026-08-16T02:00:00Z',
}

const rejection: AdminAuditEvent = {
  id: 'event-1',
  tenant_id: 'tenant-a',
  type: 'WorkloadAttestationRejected',
  occurred_at: '2026-08-16T03:00:00Z',
  payload: { tenantId: 'tenant-a', reason: 'invalid_signature', trustBundleId: 'bundle-1' },
}

describe('AdminWorkloadTrustBundlesPage', () => {
  it('shows the trust domain, issuer, status, and JWKS freshness the operator needs to triage', async () => {
    await renderWithRouter(
      <AdminWorkloadTrustBundlesPage trustBundles={[bundle]} rejectionEvents={[]} />,
    )

    expect(screen.getByRole('heading', { name: t.pageTitle })).toBeInTheDocument()
    expect(screen.getByText('example.org')).toBeInTheDocument()
    expect(screen.getByText('https://issuer.example')).toBeInTheDocument()
    expect(screen.getByText(t.statusEnabled)).toBeInTheDocument()
    // jwks_cached_at はロケール依存の表記なので、未設定を表す em ダッシュでないことだけを見る。
    expect(screen.queryByText('—')).not.toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminWorkloadTrustBundlesPage trustBundles={[bundle]} rejectionEvents={[]} />,
      { locale: 'ja' },
    )
    expect(
      screen.getByRole('heading', { name: adminWorkloadIdentityDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })

  it('links registration and details to their dedicated routes', async () => {
    await renderWithRouter(
      <AdminWorkloadTrustBundlesPage trustBundles={[bundle]} rejectionEvents={[]} />,
    )

    expect(screen.getByRole('button', { name: new RegExp(t.registerTrustBundle) })).toHaveAttribute(
      'href',
      '/admin/workload-identity/new',
    )
    expect(screen.getByRole('button', { name: t.detail })).toHaveAttribute(
      'href',
      '/admin/workload-identity/bundle-1',
    )
  })

  it('shows an empty state when nothing is registered', async () => {
    await renderWithRouter(<AdminWorkloadTrustBundlesPage trustBundles={[]} rejectionEvents={[]} />)
    expect(screen.getByText(t.emptyNotice)).toBeInTheDocument()
  })

  it('presents a rejection with a translated reason and the bundle it was attributed to', async () => {
    await renderWithRouter(
      <AdminWorkloadTrustBundlesPage trustBundles={[bundle]} rejectionEvents={[rejection]} />,
    )

    expect(screen.getByText(t.reasonInvalidSignature)).toBeInTheDocument()
    expect(screen.getAllByText('prod-cluster').length).toBeGreaterThan(0)
  })

  it('still lists a rejection from an issuer no bundle covers', async () => {
    await renderWithRouter(
      <AdminWorkloadTrustBundlesPage
        trustBundles={[bundle]}
        rejectionEvents={[
          { ...rejection, payload: { tenantId: 'tenant-a', reason: 'unregistered_issuer' } },
        ]}
      />,
    )

    expect(screen.getByText(t.rejectionsUnknownBundle)).toBeInTheDocument()
  })

  it('says the rejection history is unreadable rather than pretending there were none', async () => {
    await renderWithRouter(
      <AdminWorkloadTrustBundlesPage
        trustBundles={[bundle]}
        rejectionEvents={[]}
        rejectionsUnavailable
      />,
    )

    expect(screen.getByText(t.rejectionsLoadFailedError)).toBeInTheDocument()
    expect(screen.queryByText(t.rejectionsEmptyNotice)).not.toBeInTheDocument()
  })
})
