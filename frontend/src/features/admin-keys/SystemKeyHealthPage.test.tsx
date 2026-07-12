import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LocaleProvider } from '../../lib/i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import { KeyHealthTable, SystemKeyHealthPage } from './SystemKeyHealthPage'
import { systemKeyHealthDictionary } from './SystemKeyHealthPage.i18n'

const t = systemKeyHealthDictionary.en

function renderEn(ui: Parameters<typeof render>[0]) {
  return render(<LocaleProvider initialLocale="en">{ui}</LocaleProvider>)
}

describe('SystemKeyHealthPage', () => {
  it('renders in English by default', async () => {
    await renderWithRouter(<SystemKeyHealthPage tenants={[]} />)
    expect(
      screen.getByRole('heading', { name: systemKeyHealthDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(systemKeyHealthDictionary.en.noTenantsNotice)).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(<SystemKeyHealthPage tenants={[]} />, { locale: 'ja' })
    expect(
      screen.getByRole('heading', { name: systemKeyHealthDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(systemKeyHealthDictionary.ja.noTenantsNotice)).toBeInTheDocument()
  })
})

describe('KeyHealthTable', () => {
  it('renders an empty state and unhealthy provider state without API calls', () => {
    const { rerender } = renderEn(<KeyHealthTable tenants={[]} />)
    expect(screen.getByText(t.noTenantsNotice)).toBeInTheDocument()

    rerender(
      <LocaleProvider initialLocale="en">
        <KeyHealthTable
          tenants={[
            {
              tenant_id: 'tenant-a',
              provider: 'VaultTransit',
              usage: 'signing',
              active_kid: '',
              jwks_key_count: 2,
              provider_healthy: false,
            },
          ]}
        />
      </LocaleProvider>,
    )
    expect(screen.getByText('tenant-a')).toBeInTheDocument()
    expect(screen.getByText(t.unreachable)).toBeInTheDocument()
  })
})
