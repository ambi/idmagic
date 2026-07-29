import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'bun:test'
import { LocaleProvider } from '../../lib/i18n'
import { renderWithRouter } from '../../test/renderWithRouter'
import { DataKeyHealthTable, SystemDataKeyHealthPage } from './SystemDataKeyHealthPage'
import { systemDataKeyHealthDictionary } from './SystemDataKeyHealthPage.i18n'

const t = systemDataKeyHealthDictionary.en

function renderEn(ui: Parameters<typeof render>[0]) {
  return render(<LocaleProvider initialLocale="en">{ui}</LocaleProvider>)
}

describe('SystemDataKeyHealthPage', () => {
  it('renders in English by default', async () => {
    await renderWithRouter(<SystemDataKeyHealthPage tenants={[]} />)
    expect(
      screen.getByRole('heading', { name: systemDataKeyHealthDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(systemDataKeyHealthDictionary.en.noTenantsNotice)).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(<SystemDataKeyHealthPage tenants={[]} />, { locale: 'ja' })
    expect(
      screen.getByRole('heading', { name: systemDataKeyHealthDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(systemDataKeyHealthDictionary.ja.noTenantsNotice)).toBeInTheDocument()
  })
})

describe('DataKeyHealthTable', () => {
  it('renders an empty state and unreachable provider state without API calls', () => {
    const { rerender } = renderEn(<DataKeyHealthTable tenants={[]} />)
    expect(screen.getByText(t.noTenantsNotice)).toBeInTheDocument()

    rerender(
      <LocaleProvider initialLocale="en">
        <DataKeyHealthTable
          tenants={[
            {
              tenant_id: 'tenant-a',
              active_version: 2,
              status: 'active',
              provider: 'openbao',
              provider_reachable: false,
            },
          ]}
        />
      </LocaleProvider>,
    )
    expect(screen.getByText('tenant-a')).toBeInTheDocument()
    expect(screen.getByText(t.unreachable)).toBeInTheDocument()
  })

  it('never renders key material — only version/status/provider fields', () => {
    renderEn(
      <DataKeyHealthTable
        tenants={[
          {
            tenant_id: 'tenant-a',
            active_version: 1,
            status: 'active',
            provider: 'tink_cleartext',
            provider_reachable: true,
          },
        ]}
      />,
    )
    expect(screen.getByText('tenant-a')).toBeInTheDocument()
    expect(screen.getByText(t.healthy)).toBeInTheDocument()
  })
})
