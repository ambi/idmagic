import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { LocaleProvider } from '../../lib/i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminEntraFederationPage, EntraFederationList } from './AdminEntraFederationPage'
import { adminEntraFederationDictionary } from './AdminEntraFederationPage.i18n'

const t = adminEntraFederationDictionary.en

const integrationEndpoints = {
  issuer: 'https://login.idmagic.example/realms/acme',
  oauth: {},
  saml: {},
  ws_federation: {
    realm: 'https://login.idmagic.example/realms/acme',
    metadata_url:
      'https://login.idmagic.example/realms/acme/federationmetadata/2007-06/federationmetadata.xml',
    passive_logon_url: 'https://login.idmagic.example/realms/acme/wsfed',
    active_logon_url: 'https://login.idmagic.example/realms/acme/trust/usernamemixed',
    metadata_exchange_url: 'https://login.idmagic.example/realms/acme/trust/mex',
  },
  apis: {},
} as const

function renderEn(ui: Parameters<typeof render>[0]) {
  return render(<LocaleProvider initialLocale="en">{ui}</LocaleProvider>)
}

describe('locale', () => {
  afterEach(() => restoreGlobals())

  it('renders the entra federation page in English by default', async () => {
    await renderWithRouter(
      <AdminEntraFederationPage
        csrfToken="csrf"
        actorUsername="admin"
        relyingParties={[]}
        integrationEndpoints={integrationEndpoints as any}
      />,
    )
    expect(screen.getByRole('heading', { name: t.pageTitle })).toBeInTheDocument()
    expect(screen.getByText(t.noFederationsNotice)).toBeInTheDocument()
  })

  it('renders the entra federation page in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminEntraFederationPage
        csrfToken="csrf"
        actorUsername="admin"
        relyingParties={[]}
        integrationEndpoints={integrationEndpoints as any}
      />,
      { locale: 'ja' },
    )
    expect(
      screen.getByRole('heading', { name: adminEntraFederationDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })

  it('uses catalog URLs for Entra federation endpoints', async () => {
    await renderWithRouter(
      <AdminEntraFederationPage
        csrfToken="csrf"
        actorUsername="admin"
        relyingParties={[]}
        integrationEndpoints={integrationEndpoints as any}
      />,
    )

    expect(
      screen.getByText(integrationEndpoints.ws_federation.metadata_url).closest('a'),
    ).toHaveAttribute('href', integrationEndpoints.ws_federation.metadata_url)
    expect(
      screen.getByText(integrationEndpoints.ws_federation.metadata_exchange_url),
    ).toBeInTheDocument()
  })
})

describe('EntraFederationList', () => {
  it('renders an empty state and delegates a deletion action', () => {
    const { rerender } = renderEn(<EntraFederationList items={[]} onDelete={mock()} />)
    expect(screen.getByText(t.noFederationsNotice)).toBeInTheDocument()

    const onDelete = mock()
    rerender(
      <LocaleProvider initialLocale="en">
        <EntraFederationList
          items={[
            {
              tenant_id: 'tenant-a',
              wtrealm: 'urn:contoso',
              reply_urls: ['https://contoso.example.test/reply'],
              claim_policy: { name_id: { format: 'persistent', source_attribute: 'sub' } },
              created_at: '2026-01-01T00:00:00Z',
              entra_profile: {
                domain: 'contoso.com',
                source_anchor_attribute: 'object_guid',
                issuer_uri: 'https://idp.example.test',
                immutable_id_attribute: 'object_guid',
              },
            },
          ]}
          onDelete={onDelete}
        />
      </LocaleProvider>,
    )
    fireEvent.click(
      screen.getByRole('button', { name: t.deleteAriaLabel.replace('{wtrealm}', 'urn:contoso') }),
    )
    expect(onDelete).toHaveBeenCalledWith(expect.objectContaining({ wtrealm: 'urn:contoso' }))
  })
})
