import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderWithRouter } from '../test/renderWithRouter'
import { SystemShell } from './SystemShell'

describe('SystemShell', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('marks the active nav item and shows a return link back to the admin console', async () => {
    await renderWithRouter(
      <SystemShell active="tenants" title="Tenants" description="Description">
        <p>content</p>
      </SystemShell>,
    )

    expect(screen.getByRole('link', { name: 'Tenants' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Admin console' })).toHaveAttribute('href', '/admin')
    expect(screen.getByText('System console')).toBeInTheDocument()
    expect(screen.getByText('Description')).toBeInTheDocument()
  })

  it('falls back to a default label when no actor username is provided', async () => {
    await renderWithRouter(
      <SystemShell active="key-health" title="Signing key health">
        <p>content</p>
      </SystemShell>,
    )

    expect(screen.getByText('system administrator')).toBeInTheDocument()
    expect(screen.getByText('S')).toBeInTheDocument()
  })

  it('opens the account menu and signs the system administrator out', async () => {
    const assign = vi.fn()
    vi.stubGlobal('location', { ...window.location, assign })
    await renderWithRouter(
      <SystemShell active="key-health" actorUsername="Sonoko" title="Signing key health">
        <p>content</p>
      </SystemShell>,
    )

    fireEvent.keyDown(screen.getByRole('button', { name: 'Account menu' }), { key: 'Enter' })
    const menu = screen.getByRole('menu')
    expect(within(menu).getByText('Sonoko')).toBeInTheDocument()
    fireEvent.click(within(menu).getByRole('menuitem', { name: /Sign out/ }))

    await waitFor(() =>
      expect(assign).toHaveBeenCalledWith(expect.stringContaining('/end_session')),
    )
  })
})
