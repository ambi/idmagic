import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import {
  AdminApplicationsPage,
  AdminApplicationDetailPage,
  AdminApplicationEditPage,
} from './AdminApplicationsPage'
import type { AdminApplication, AdminApplicationDetail } from '../../types'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

const app: AdminApplication = {
  application_id: 'app-1',
  name: 'Payroll',
  kind: 'federated',
  status: 'active',
  bindings: [{ type: 'oidc', client_id: 'client-1' }],
  category_ids: [],
  category_names: [],
  binding_summaries: [],
  assigned_subject_count: 0,
  sign_in_policy_summary: '',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const detail: AdminApplicationDetail = { application: app }

describe('AdminApplicationsPage', () => {
  const originalLocation = window.location
  afterEach(() => vi.unstubAllGlobals())

  it('deletes an application and refreshes the list on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/applications') && init?.method === 'DELETE') {
          return Promise.resolve(response(204))
        }
        if (url.includes('/api/admin/applications')) {
          return Promise.resolve(response(200, { applications: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminApplicationsPage csrfToken="csrf" applications={[app]} />)

    fireEvent.click(screen.getByRole('button', { name: 'アプリケーションを削除' }))
    fireEvent.click(screen.getByRole('button', { name: '削除を確定' }))

    expect(await screen.findByText('アプリケーションを削除しました。')).toBeInTheDocument()
    expect(screen.getByText('アプリケーションを選択してください。')).toBeInTheDocument()
  })

  it('shows an error when deleting an application fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/applications') && init?.method === 'DELETE') {
          return Promise.resolve(
            response(409, { message: 'アプリケーションを削除できませんでした。' }),
          )
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminApplicationsPage csrfToken="csrf" applications={[app]} />)

    fireEvent.click(screen.getByRole('button', { name: 'アプリケーションを削除' }))
    fireEvent.click(screen.getByRole('button', { name: '削除を確定' }))

    expect(await screen.findByText('アプリケーションを削除できませんでした。')).toBeInTheDocument()
  })

  it('creates an OIDC application and redirects to its detail page', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/applications') && init?.method === 'POST') {
          return Promise.resolve(
            response(201, {
              application: { ...app, application_id: 'app-2', name: 'New App' },
              client_id: 'client-2',
              client_secret: 'secret-2',
            }),
          )
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminApplicationsPage csrfToken="csrf" applications={[app]} />)

    fireEvent.click(screen.getByRole('button', { name: 'アプリケーションを追加' }))
    fireEvent.change(await screen.findByLabelText('名前'), { target: { value: 'New App' } })
    fireEvent.change(screen.getByLabelText('リダイレクト URI'), {
      target: { value: 'https://app.example.com/callback' },
    })
    fireEvent.click(screen.getByRole('button', { name: '作成' }))

    fireEvent.click(await screen.findByRole('button', { name: '保管しました' }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/applications/app-2'),
    )
  })

  it('shows an error and keeps the dialog open when creation fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(response(409, { message: 'アプリケーションを作成できませんでした。' })),
      ),
    )
    await renderWithRouter(<AdminApplicationsPage csrfToken="csrf" applications={[app]} />)

    fireEvent.click(screen.getByRole('button', { name: 'アプリケーションを追加' }))
    fireEvent.change(await screen.findByLabelText('名前'), { target: { value: 'New App' } })
    fireEvent.change(screen.getByLabelText('リダイレクト URI'), {
      target: { value: 'https://app.example.com/callback' },
    })
    fireEvent.click(screen.getByRole('button', { name: '作成' }))

    expect(await screen.findByText('アプリケーションを作成できませんでした。')).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})

describe('AdminApplicationDetailPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('shows an error and keeps the confirmation open when deletion fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(response(409, { message: 'アプリケーションを削除できませんでした。' })),
      ),
    )
    await renderWithRouter(<AdminApplicationDetailPage csrfToken="csrf" detail={detail} />)

    fireEvent.click(screen.getByRole('button', { name: '削除' }))
    fireEvent.click(screen.getByRole('button', { name: '削除を確定' }))

    expect(await screen.findByText('アプリケーションを削除できませんでした。')).toBeInTheDocument()
  })
})

describe('AdminApplicationEditPage', () => {
  const originalLocation = window.location
  afterEach(() => vi.unstubAllGlobals())

  it('shows an error and keeps the form when saving fails', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(400, { message: '名前を更新できませんでした。' }))),
    )
    await renderWithRouter(<AdminApplicationEditPage csrfToken="csrf" detail={detail} />)

    fireEvent.change(screen.getByLabelText('名前'), { target: { value: 'Renamed App' } })
    fireEvent.click(screen.getByRole('button', { name: '保存' }))

    expect(await screen.findByText('名前を更新できませんでした。')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})
