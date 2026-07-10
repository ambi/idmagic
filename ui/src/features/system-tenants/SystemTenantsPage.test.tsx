import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { TenantTable } from './SystemTenantsPage'

const tenant = {
  id: 'tenant-1',
  realm: 'acme',
  display_name: 'Acme',
  status: 'active' as const,
  created_at: '2026-01-01T00:00:00Z',
}

describe('TenantTable', () => {
  it('keeps the default tenant protected and forwards a selected tenant', () => {
    const onSelect = vi.fn()
    render(
      <TenantTable
        tenants={[tenant, { ...tenant, id: 'default', realm: 'default' }]}
        busy={false}
        onSelect={onSelect}
        onToggleDisabled={vi.fn()}
      />,
    )
    fireEvent.click(screen.getByText('acme'))
    expect(onSelect).toHaveBeenCalledWith(tenant)
    expect(screen.getByRole('button', { name: '無効化' })).toBeInTheDocument()
  })
})
