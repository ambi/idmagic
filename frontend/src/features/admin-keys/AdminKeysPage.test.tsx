import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { SigningKeyTable } from './AdminKeysPage'

const key = {
  kid: 'kid-1',
  provider: 'Local',
  alg: 'RS256',
  active: true,
  created_at: '2026-01-01T00:00:00Z',
  public_jwk: { kty: 'RSA' },
}

describe('SigningKeyTable', () => {
  it('notifies selection without exposing destructive actions to non-managers', () => {
    const onSelect = vi.fn()
    render(
      <SigningKeyTable
        keys={[key]}
        canManage={false}
        busy={false}
        onSelect={onSelect}
        onDisable={vi.fn()}
      />,
    )
    fireEvent.click(screen.getByText('kid-1'))
    expect(onSelect).toHaveBeenCalledWith(key)
    expect(screen.queryByRole('button', { name: '鍵 kid-1 を無効化' })).not.toBeInTheDocument()
  })
})
