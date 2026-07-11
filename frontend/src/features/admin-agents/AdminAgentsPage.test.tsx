import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, within } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminAgentsPage } from './AdminAgentsPage'
import type { AdminAgent } from '../../types'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

const agent: AdminAgent = {
  id: 'agent-1',
  tenant_id: 'tenant-1',
  name: 'invoice-bot',
  kind: 'autonomous',
  owner_user_id: 'user-1',
  status: 'active',
  roles: ['invoice:read'],
  client_ids: [],
  created_at: '2026-01-01T00:00:00Z',
}

describe('AdminAgentsPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('registers an agent and refreshes the list on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/agents') && init?.method === 'POST') {
          return Promise.resolve(response(201, { ...agent, id: 'agent-2', name: 'billing-bot' }))
        }
        if (url.includes('/api/admin/agents')) {
          return Promise.resolve(response(200, { agents: [agent] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} />)

    fireEvent.click(screen.getByRole('button', { name: '登録' }))
    const nameInput = await screen.findByLabelText('エージェント名')
    fireEvent.change(nameInput, { target: { value: 'billing-bot' } })
    const form = nameInput.closest('form') as HTMLFormElement
    fireEvent.click(within(form).getByRole('button', { name: '登録' }))

    expect(await screen.findByText('エージェントを登録しました。')).toBeInTheDocument()
  })

  it('shows an error and keeps the dialog open when registration fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(response(409, { message: 'このエージェント名は既に使われています。' })),
      ),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} />)

    fireEvent.click(screen.getByRole('button', { name: '登録' }))
    const nameInput = await screen.findByLabelText('エージェント名')
    fireEvent.change(nameInput, { target: { value: 'billing-bot' } })
    const form = nameInput.closest('form') as HTMLFormElement
    fireEvent.click(within(form).getByRole('button', { name: '登録' }))

    expect(await screen.findByText('このエージェント名は既に使われています。')).toBeInTheDocument()
  })

  it('deletes an agent and refreshes the list on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/agents/agent-1') && init?.method === 'DELETE') {
          return Promise.resolve(response(204))
        }
        if (url.includes('/api/admin/agents')) {
          return Promise.resolve(response(200, { agents: [] }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} />)

    fireEvent.click(screen.getByRole('button', { name: 'エージェントを削除' }))
    fireEvent.click(screen.getByRole('button', { name: '削除を確定' }))

    expect(await screen.findByText('エージェントを削除しました。')).toBeInTheDocument()
    expect(screen.getByText('エージェントを選択してください。')).toBeInTheDocument()
  })

  it('shows an error when binding a credential fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(response(409, { message: 'この client_id は既に使用されています。' })),
      ),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} />)

    fireEvent.change(screen.getByLabelText('バインドする client_id'), {
      target: { value: 'client-x' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'バインド' }))

    expect(await screen.findByText('この client_id は既に使用されています。')).toBeInTheDocument()
  })

  it('shows an error and keeps the dialog open when editing an agent fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(response(400, { message: 'エージェントを更新できませんでした。' })),
      ),
    )
    await renderWithRouter(<AdminAgentsPage csrfToken="csrf" agents={[agent]} />)

    fireEvent.click(screen.getByRole('button', { name: '編集' }))
    fireEvent.change(await screen.findByLabelText('エージェント名'), {
      target: { value: 'renamed-bot' },
    })
    fireEvent.click(screen.getByRole('button', { name: '保存' }))

    expect(await screen.findByText('エージェントを更新できませんでした。')).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})
