import { screen } from '@testing-library/react'
import { describe, expect, it } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminDirectoryConnectorsPage } from './AdminDirectoryConnectorsPage'
import { adminDirectoryConnectorsDictionary } from './AdminDirectoryConnectorsPage.i18n'

const t = adminDirectoryConnectorsDictionary.en

describe('AdminDirectoryConnectorsPage', () => {
  it('Connector の状態と詳細へのリンクを表示する', async () => {
    await renderWithRouter(
      <AdminDirectoryConnectorsPage
        connectors={[
          {
            id: 'connector-1',
            display_name: 'Corp AD',
            status: 'disabled',
            created_at: '2026-10-31T00:00:00Z',
            accidental_deletion_count_threshold: 500,
            accidental_deletion_percent_threshold: 20,
          },
        ]}
      />,
    )
    expect(screen.getByText('Corp AD')).toBeTruthy()
    expect(screen.getByText(t.statusDisabled)).toBeTruthy()
    expect(screen.getByText(t.detail).closest('a')?.getAttribute('href')).toContain(
      '/admin/directory-connectors/connector-1',
    )
  })

  it('Connector がなければ案内を表示する', async () => {
    await renderWithRouter(<AdminDirectoryConnectorsPage connectors={[]} />)
    expect(screen.getByText(t.emptyNotice)).toBeTruthy()
  })
})
