import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent, waitFor } from '@testing-library/react'
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
  application_id: 'app-1',
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
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

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

  it('creates an OIDC application and redirects to its detail page', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
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

    fireEvent.click(screen.getByRole('button', { name: t.addApplication }))
    fireEvent.change(await screen.findByLabelText(t.nameFieldLabel), {
      target: { value: 'New App' },
    })
    fireEvent.change(screen.getByLabelText(t.redirectUriFieldLabel), {
      target: { value: 'https://app.example.com/callback' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    fireEvent.click(await screen.findByRole('button', { name: t.storedConfirm }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/applications/app-2'),
    )
  })

  it('shows an error and keeps the dialog open when creation fails', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(409, { message: 'Could not create the application.' }))),
    )
    await renderWithRouter(<AdminApplicationsPage csrfToken="csrf" applications={[app]} />)

    fireEvent.click(screen.getByRole('button', { name: t.addApplication }))
    fireEvent.change(await screen.findByLabelText(t.nameFieldLabel), {
      target: { value: 'New App' },
    })
    fireEvent.change(screen.getByLabelText(t.redirectUriFieldLabel), {
      target: { value: 'https://app.example.com/callback' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    expect(await screen.findByText('Could not create the application.')).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})
