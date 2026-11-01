import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import { renderWithRouter } from '../test/renderWithRouter'
import { AdminShell } from './AdminShell'
import { shellDictionary } from './shell.i18n'
import { commonDictionary } from '../lib/i18n/common.i18n'

describe('AdminShell', () => {
  afterEach(() => restoreGlobals())

  //spec:covers REQ-FRONTEND-001: 総数と表示件数を区別し、一覧を保持して更新する。
  it('shows list totals and refreshes without dropping the current list', async () => {
    const refresh = mock(async () => {})
    await renderWithRouter(
      <AdminShell
        active="groups"
        title="Groups"
        list={{ total: 1234, displayed: 25, onRefresh: refresh }}
      >
        <p>current list</p>
      </AdminShell>,
    )
    expect(
      screen.getByText(commonDictionary.en.totalCount.replace('{count}', '1,234')),
    ).toBeInTheDocument()
    expect(
      screen.getByText(commonDictionary.en.displayedCount.replace('{count}', '25')),
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: commonDictionary.en.refresh }))
    await waitFor(() => expect(refresh).toHaveBeenCalledTimes(1))
    expect(screen.getByText('current list')).toBeInTheDocument()
  })

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

  // 管理コンソールの言語切り替えは、切り替えた側 (ボタン) ではなく切り替えられた側
  // (シェルの文言) で観測する。押した結果が aria-pressed にしか出ない実装でも前者は通る。
  //
  //spec:covers EX-SYSTEM-009-01: Administrator が管理画面で表示言語を選ぶと、シェルとパンくずの文言がその辞書へ替わること。
  it('renders the admin chrome in the language the administrator selects', async () => {
    await renderWithRouter(
      <AdminShell active="dashboard" title="Dashboard">
        <p>content</p>
      </AdminShell>,
      { locale: 'ja' },
    )

    expect(
      screen.getByRole('navigation', { name: shellDictionary.ja.breadcrumb }),
    ).toHaveTextContent(shellDictionary.ja.adminConsole)

    fireEvent.click(screen.getByRole('button', { name: 'English' }))

    expect(
      screen.getByRole('navigation', { name: shellDictionary.en.breadcrumb }),
    ).toHaveTextContent(shellDictionary.en.adminConsole)
    expect(document.documentElement.lang).toBe('en')
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
    const assign = mock()
    stubGlobal('location', { ...window.location, assign })
    await renderWithRouter(
      <AdminShell active="dashboard" actorUsername="Alice" title="Dashboard">
        <p>content</p>
      </AdminShell>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Account menu' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Sign out' }))

    await waitFor(() =>
      expect(assign).toHaveBeenCalledWith(expect.stringContaining('/end_session')),
    )
  })
})
