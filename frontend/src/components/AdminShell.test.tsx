import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderWithRouter } from '../test/renderWithRouter'
import { AdminShell } from './AdminShell'

describe('AdminShell', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('marks the active nav item and shows a two-level breadcrumb', async () => {
    await renderWithRouter(
      <AdminShell active="users" title="Users" description="Description">
        <p>content</p>
      </AdminShell>,
    )

    expect(screen.getByRole('link', { name: 'Users' })).toHaveAttribute('aria-current', 'page')
    const breadcrumb = screen.getByRole('navigation', { name: 'Breadcrumb' })
    expect(within(breadcrumb).getByRole('link', { name: 'Admin console' })).toBeInTheDocument()
    expect(screen.getByText('Description')).toBeInTheDocument()
  })

  it('collapses the breadcrumb to a single entry on the dashboard', async () => {
    await renderWithRouter(
      <AdminShell active="dashboard" title="Dashboard">
        <p>content</p>
      </AdminShell>,
    )

    const breadcrumb = screen.getByRole('navigation', { name: 'Breadcrumb' })
    expect(breadcrumb).toHaveTextContent('Admin console')
    expect(within(breadcrumb).queryByRole('link')).not.toBeInTheDocument()
  })

  it('falls back to a default label when no actor username is provided', async () => {
    await renderWithRouter(
      <AdminShell active="dashboard" title="Dashboard">
        <p>content</p>
      </AdminShell>,
    )

    expect(screen.getByText('administrator')).toBeInTheDocument()
    expect(screen.getByText('A')).toBeInTheDocument()
  })

  it('opens the account menu and signs the admin out', async () => {
    const assign = vi.fn()
    vi.stubGlobal('location', { ...window.location, assign })
    await renderWithRouter(
      <AdminShell active="dashboard" actorUsername="Alice" title="Dashboard">
        <p>content</p>
      </AdminShell>,
    )

    fireEvent.keyDown(screen.getByRole('button', { name: 'Account menu' }), { key: 'Enter' })
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Sign out' }))

    await waitFor(() =>
      expect(assign).toHaveBeenCalledWith(expect.stringContaining('/end_session')),
    )
  })
})
