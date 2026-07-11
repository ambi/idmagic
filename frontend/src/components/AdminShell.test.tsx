import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderWithRouter } from '../test/renderWithRouter'
import { AdminShell } from './AdminShell'

describe('AdminShell', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('marks the active nav item and shows a two-level breadcrumb', async () => {
    await renderWithRouter(
      <AdminShell active="users" title="ユーザー" description="説明文">
        <p>content</p>
      </AdminShell>,
    )

    expect(screen.getByRole('link', { name: 'ユーザー' })).toHaveAttribute('aria-current', 'page')
    const breadcrumb = screen.getByRole('navigation', { name: 'breadcrumb' })
    expect(within(breadcrumb).getByRole('link', { name: '管理コンソール' })).toBeInTheDocument()
    expect(screen.getByText('説明文')).toBeInTheDocument()
  })

  it('collapses the breadcrumb to a single entry on the dashboard', async () => {
    await renderWithRouter(
      <AdminShell active="dashboard" title="ダッシュボード">
        <p>content</p>
      </AdminShell>,
    )

    const breadcrumb = screen.getByRole('navigation', { name: 'breadcrumb' })
    expect(breadcrumb).toHaveTextContent('管理コンソール')
    expect(within(breadcrumb).queryByRole('link')).not.toBeInTheDocument()
  })

  it('falls back to a default label when no actor username is provided', async () => {
    await renderWithRouter(
      <AdminShell active="dashboard" title="ダッシュボード">
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
      <AdminShell active="dashboard" actorUsername="Alice" title="ダッシュボード">
        <p>content</p>
      </AdminShell>,
    )

    fireEvent.keyDown(screen.getByRole('button', { name: 'アカウントメニュー' }), { key: 'Enter' })
    fireEvent.click(await screen.findByRole('menuitem', { name: 'ログアウト' }))

    await waitFor(() =>
      expect(assign).toHaveBeenCalledWith(expect.stringContaining('/end_session')),
    )
  })
})
