import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { EntraFederationList } from './AdminEntraFederationPage'

describe('EntraFederationList', () => {
  it('renders an empty state and delegates a deletion action', () => {
    const { rerender } = render(<EntraFederationList items={[]} onDelete={vi.fn()} />)
    expect(screen.getByText(/まだフェデレーション済みのドメインがありません/)).toBeInTheDocument()

    const onDelete = vi.fn()
    rerender(
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
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'urn:contoso を削除' }))
    expect(onDelete).toHaveBeenCalledWith(expect.objectContaining({ wtrealm: 'urn:contoso' }))
  })
})
