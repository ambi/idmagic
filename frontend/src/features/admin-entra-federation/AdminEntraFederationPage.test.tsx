import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '../../lib/i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminEntraFederationPage, EntraFederationList } from './AdminEntraFederationPage'
import { adminEntraFederationDictionary } from './AdminEntraFederationPage.i18n'

const t = adminEntraFederationDictionary.en

function renderEn(ui: Parameters<typeof render>[0]) {
  return render(<LocaleProvider initialLocale="en">{ui}</LocaleProvider>)
}

describe('locale', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('renders the entra federation page in English by default', async () => {
    await renderWithRouter(
      <AdminEntraFederationPage csrfToken="csrf" actorUsername="admin" relyingParties={[]} />,
    )
    expect(screen.getByRole('heading', { name: t.pageTitle })).toBeInTheDocument()
    expect(screen.getByText(t.noFederationsNotice)).toBeInTheDocument()
  })

  it('renders the entra federation page in Japanese when explicitly selected', async () => {
    await renderWithRouter(
      <AdminEntraFederationPage csrfToken="csrf" actorUsername="admin" relyingParties={[]} />,
      { locale: 'ja' },
    )
    expect(
      screen.getByRole('heading', { name: adminEntraFederationDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('EntraFederationList', () => {
  it('renders an empty state and delegates a deletion action', () => {
    const { rerender } = renderEn(<EntraFederationList items={[]} onDelete={vi.fn()} />)
    expect(screen.getByText(t.noFederationsNotice)).toBeInTheDocument()

    const onDelete = vi.fn()
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
