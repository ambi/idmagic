import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { KeyHealthTable } from './SystemKeyHealthPage'

describe('KeyHealthTable', () => {
  it('renders an empty state and unhealthy provider state without API calls', () => {
    const { rerender } = render(<KeyHealthTable tenants={[]} />)
    expect(screen.getByText('テナントがありません。')).toBeInTheDocument()

    rerender(
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
      />,
    )
    expect(screen.getByText('tenant-a')).toBeInTheDocument()
    expect(screen.getByText('接続不可')).toBeInTheDocument()
  })
})
