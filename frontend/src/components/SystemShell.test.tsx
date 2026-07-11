import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderWithRouter } from '../test/renderWithRouter'
import { SystemShell } from './SystemShell'

describe('SystemShell', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('marks the active nav item and shows a return link back to the admin console', async () => {
    await renderWithRouter(
      <SystemShell active="tenants" title="テナント" description="説明文">
        <p>content</p>
      </SystemShell>,
    )

    expect(screen.getByRole('link', { name: 'テナント' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: '管理コンソール' })).toHaveAttribute('href', '/admin')
    expect(screen.getByText('システムコンソール')).toBeInTheDocument()
    expect(screen.getByText('説明文')).toBeInTheDocument()
  })

  it('falls back to a default label when no actor username is provided', async () => {
    await renderWithRouter(
      <SystemShell active="key-health" title="署名鍵の状態">
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
      <SystemShell active="key-health" actorUsername="Sonoko" title="署名鍵の状態">
        <p>content</p>
      </SystemShell>,
    )

    fireEvent.keyDown(screen.getByRole('button', { name: 'アカウントメニュー' }), { key: 'Enter' })
    const menu = screen.getByRole('menu')
    expect(within(menu).getByText('Sonoko')).toBeInTheDocument()
    fireEvent.click(within(menu).getByRole('menuitem', { name: /ログアウト/ }))

    await waitFor(() =>
      expect(assign).toHaveBeenCalledWith(expect.stringContaining('/end_session')),
    )
  })
})
