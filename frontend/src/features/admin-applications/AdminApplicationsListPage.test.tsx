import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminApplicationsPage } from './AdminApplicationsListPage'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import type { AdminApplication } from '../../types'

const t = adminApplicationsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const app: AdminApplication = {
  id: 'app-1',
  name: 'Payroll',
  kind: 'federated',
  status: 'active',
  protocol: { type: 'oidc', client_id: 'client-1' },
  category_ids: [],
  category_names: [],
  assigned_subject_count: 0,
  sign_in_policy_summary: '',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('locale', () => {
  afterEach(() => restoreGlobals())

  it('renders the application list in English by default', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { applications: [] }))),
    )
    await renderWithRouter(<AdminApplicationsPage csrfToken="csrf" applications={[]} />)
    expect(
      screen.getByRole('heading', { name: adminApplicationsDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(adminApplicationsDictionary.en.selectApplicationPrompt),
    ).toBeInTheDocument()
  })

  it('renders the application list in Japanese when explicitly selected', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { applications: [] }))),
    )
    await renderWithRouter(<AdminApplicationsPage csrfToken="csrf" applications={[]} />, {
      locale: 'ja',
    })
    expect(
      screen.getByRole('heading', { name: adminApplicationsDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
  })
})

describe('AdminApplicationsPage', () => {
  afterEach(() => restoreGlobals())

  it('links "add application" to the dedicated create route instead of a modal', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { applications: [app] }))),
    )
    await renderWithRouter(<AdminApplicationsPage csrfToken="csrf" applications={[app]} />)

    expect(screen.getByRole('button', { name: new RegExp(t.addApplication) })).toHaveAttribute(
      'href',
      '/admin/applications/new',
    )
  })

  it('deletes an application and refreshes the list on success', async () => {
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
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

    fireEvent.click(screen.getByRole('button', { name: t.deleteApplication }))
    fireEvent.click(screen.getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText(t.applicationDeletedNotice)).toBeInTheDocument()
    expect(screen.getByText(t.selectApplicationPrompt)).toBeInTheDocument()
  })

  it('shows an error when deleting an application fails', async () => {
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/applications') && init?.method === 'DELETE') {
          return Promise.resolve(response(409, { message: 'Could not delete the application.' }))
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminApplicationsPage csrfToken="csrf" applications={[app]} />)

    fireEvent.click(screen.getByRole('button', { name: t.deleteApplication }))
    fireEvent.click(screen.getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText('Could not delete the application.')).toBeInTheDocument()
  })
})
