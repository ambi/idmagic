import { afterEach, describe, expect, it, mock } from 'bun:test'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminSamlIDPProfile } from '../../types'
import { AdminSamlIDPProfileCreatePage } from './AdminSamlIDPProfileCreatePage'
import { AdminSamlIDPProfileDetailPage } from './AdminSamlIDPProfileDetailPage'
import { AdminSamlIDPProfileEditPage } from './AdminSamlIDPProfileEditPage'
import { AdminSamlIDPProfilesListPage } from './AdminSamlIDPProfilesListPage'
import { adminSamlIDPProfilesDictionary } from './AdminSamlIDPProfilesPage.i18n'

const t = adminSamlIDPProfilesDictionary.en
const originalLocation = window.location

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const defaultProfile: AdminSamlIDPProfile = {
  profile: {
    tenant_id: 'tenant-1',
    profile_id: 'default',
    name: 'Default',
    mode: 'shared',
    is_default: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
  entity_id: 'https://idp.example/realms/acme',
  metadata_url: 'https://idp.example/realms/acme/saml/metadata',
  sso_url: 'https://idp.example/realms/acme/saml/sso',
  slo_url: 'https://idp.example/realms/acme/saml/slo',
  signing_certificate_url: 'https://idp.example/realms/acme/saml/signing-certificate.pem',
  signing_certificate_fingerprint_sha256: 'AA:BB:CC',
  service_provider_count: 2,
}

const partnerProfile: AdminSamlIDPProfile = {
  ...defaultProfile,
  profile: {
    ...defaultProfile.profile,
    profile_id: 'partner',
    name: 'Partner trust',
    is_default: false,
  },
  entity_id: 'https://idp.example/realms/acme/saml/idp/partner',
  metadata_url: 'https://idp.example/realms/acme/saml/idp/partner/metadata',
  sso_url: 'https://idp.example/realms/acme/saml/idp/partner/sso',
  slo_url: 'https://idp.example/realms/acme/saml/idp/partner/slo',
  signing_certificate_url:
    'https://idp.example/realms/acme/saml/idp/partner/signing-certificate.pem',
  service_provider_count: 0,
}

describe('SAML IdP profile routed management', () => {
  afterEach(() => restoreGlobals())

  it('shows a read-only list with detail links and a dedicated create route', async () => {
    await renderWithRouter(
      <AdminSamlIDPProfilesListPage profiles={[defaultProfile, partnerProfile]} />,
    )

    expect(screen.getByText('Default')).toBeInTheDocument()
    expect(screen.getByText('Partner trust')).toBeInTheDocument()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: t.createProfile }).getAttribute('href')).toBe(
      '/admin/settings/saml-idp-profiles/new',
    )
    expect(screen.getAllByRole('link', { name: t.viewDetails })[1].getAttribute('href')).toBe(
      '/admin/settings/saml-idp-profiles/partner',
    )
  })

  it('keeps default detail read-only and offers editing only for an additional profile', async () => {
    const { unmount } = await renderWithRouter(
      <AdminSamlIDPProfileDetailPage csrfToken="csrf" entry={defaultProfile} />,
    )
    expect(screen.getByText(t.defaultImmutable)).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: t.editProfile })).not.toBeInTheDocument()
    unmount()

    await renderWithRouter(
      <AdminSamlIDPProfileDetailPage csrfToken="csrf" entry={partnerProfile} />,
    )
    expect(screen.getByRole('link', { name: t.editProfile }).getAttribute('href')).toBe(
      '/admin/settings/saml-idp-profiles/partner/edit',
    )
    expect(screen.getByRole('button', { name: t.deleteProfile })).toBeEnabled()
  })

  it('creates a profile and redirects to its read-only detail route', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(201, partnerProfile))),
    )
    await renderWithRouter(<AdminSamlIDPProfileCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.profileName), {
      target: { value: partnerProfile.profile.name },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith(
        '/admin/settings/saml-idp-profiles/partner',
      ),
    )
  })

  it('updates a profile only on its dedicated edit route', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() =>
        Promise.resolve(
          response(200, {
            ...partnerProfile,
            profile: { ...partnerProfile.profile, name: 'Partner renamed' },
          }),
        ),
      ),
    )
    await renderWithRouter(<AdminSamlIDPProfileEditPage csrfToken="csrf" entry={partnerProfile} />)

    fireEvent.change(screen.getByLabelText(t.profileName), {
      target: { value: 'Partner renamed' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith(
        '/admin/settings/saml-idp-profiles/partner',
      ),
    )
  })
})
