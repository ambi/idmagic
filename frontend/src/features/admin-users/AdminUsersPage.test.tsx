import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminUsersPage, AdminUserCreatePage, AdminUserEditPage } from './AdminUsersPage'
import type { AdminUser, TenantUserAttributeSchema } from '../../types'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

const emptySchema: TenantUserAttributeSchema = {
  tenant_id: 'tenant-1',
  builtin: [],
  attributes: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const user: AdminUser = {
  id: 'user-1',
  preferred_username: 'taro',
  name: 'Taro Yamada',
  email: 'taro@example.com',
  email_verified: true,
  mfa_enrolled: false,
  roles: ['support'],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('AdminUsersPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('deletes a user and refreshes the list on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/users') && init?.method === 'DELETE') {
          return Promise.resolve(response(204))
        }
        if (url.includes('/api/admin/users')) {
          return Promise.resolve(response(200, { users: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminUsersPage csrfToken="csrf" users={[user]} />)

    fireEvent.click(screen.getByRole('button', { name: 'アカウントを削除' }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: '削除を確定' }))

    expect(
      await screen.findByText('ユーザーの削除を予約しました。30 日以内なら復元できます。'),
    ).toBeInTheDocument()
    expect(screen.getByText('ユーザーを選択すると詳細が表示されます。')).toBeInTheDocument()
  })

  it('shows an error and keeps the dialog open when deletion fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/users') && init?.method === 'DELETE') {
          return Promise.resolve(response(409, { message: 'ユーザーを削除できませんでした。' }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminUsersPage csrfToken="csrf" users={[user]} />)

    fireEvent.click(screen.getByRole('button', { name: 'アカウントを削除' }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: '削除を確定' }))

    expect(await screen.findByText('ユーザーを削除できませんでした。')).toBeInTheDocument()
  })

  it('shows an error when disabling a user fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) => {
        if (url.includes('/disable')) {
          return Promise.resolve(response(403, { message: '無効化する権限がありません。' }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminUsersPage csrfToken="csrf" users={[user]} />)

    fireEvent.click(screen.getByRole('button', { name: 'アカウントを無効化' }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: '無効化を確定' }))

    expect(await screen.findByText('無効化する権限がありません。')).toBeInTheDocument()
  })

  it('shows an error when reloading the list fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(500, { message: '一覧を取得できませんでした。' }))),
    )
    await renderWithRouter(<AdminUsersPage csrfToken="csrf" users={[user]} />)

    fireEvent.click(screen.getByRole('button', { name: '一覧を再読み込み' }))

    expect(await screen.findByText('一覧を取得できませんでした。')).toBeInTheDocument()
  })
})

describe('AdminUserCreatePage', () => {
  const originalLocation = window.location
  afterEach(() => vi.unstubAllGlobals())

  it('creates a user and redirects to the detail page', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(201, { ...user, id: 'user-2' }))),
    )
    await renderWithRouter(<AdminUserCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText('ユーザー名'), { target: { value: 'jiro' } })
    fireEvent.change(screen.getByLabelText('初期パスワード'), {
      target: { value: 'correct horse battery staple' },
    })
    fireEvent.click(screen.getByRole('button', { name: '作成' }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/admin/users/user-2'))
  })

  it('shows an error and keeps the form when creation fails', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(response(409, { message: 'このユーザー名は既に使われています。' })),
      ),
    )
    await renderWithRouter(<AdminUserCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText('ユーザー名'), { target: { value: 'jiro' } })
    fireEvent.change(screen.getByLabelText('初期パスワード'), {
      target: { value: 'correct horse battery staple' },
    })
    fireEvent.click(screen.getByRole('button', { name: '作成' }))

    expect(await screen.findByText('このユーザー名は既に使われています。')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})

describe('AdminUserEditPage', () => {
  const originalLocation = window.location
  afterEach(() => vi.unstubAllGlobals())

  it('shows an error and keeps the form when updating profile fields fails', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(400, { message: '表示名を更新できませんでした。' }))),
    )
    await renderWithRouter(<AdminUserEditPage csrfToken="csrf" user={user} schema={emptySchema} />)

    fireEvent.change(screen.getByLabelText('表示名'), { target: { value: 'Jiro Yamada' } })
    fireEvent.click(screen.getByRole('button', { name: '保存' }))

    expect(await screen.findByText('表示名を更新できませんでした。')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  it('requires a confirmation step before submitting a role change', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(200, user))),
    )
    await renderWithRouter(<AdminUserEditPage csrfToken="csrf" user={user} schema={emptySchema} />)

    fireEvent.change(screen.getByLabelText('ロール'), { target: { value: 'admin' } })
    fireEvent.click(screen.getByRole('button', { name: '変更内容を確認' }))

    expect(await screen.findByText('ロール変更を含む更新です')).toBeInTheDocument()
    expect(fetch).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: '変更を確定' }))

    await waitFor(() => expect(window.location.assign).toHaveBeenCalledWith('/admin/users/user-1'))
  })
})
