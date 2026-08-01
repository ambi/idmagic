import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminApplicationCreatePage } from './AdminApplicationCreatePage'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'

const t = adminApplicationsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('AdminApplicationCreatePage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('creates an OIDC application and redirects to its detail page after confirming the secret', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock((url: string, init?: RequestInit) => {
        if (url.includes('/api/admin/applications') && init?.method === 'POST') {
          return Promise.resolve(
            response(201, {
              application: { application_id: 'app-2', name: 'New App' },
              client_id: 'client-2',
              client_secret: 'secret-2',
            }),
          )
        }
        throw new Error(`unexpected fetch ${url}`)
      }),
    )
    await renderWithRouter(<AdminApplicationCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.nameFieldLabel), {
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

  it('shows an error and keeps the form when creation fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(409, { message: 'Could not create the application.' }))),
    )
    await renderWithRouter(<AdminApplicationCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.nameFieldLabel), {
      target: { value: 'New App' },
    })
    fireEvent.change(screen.getByLabelText(t.redirectUriFieldLabel), {
      target: { value: 'https://app.example.com/callback' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.create }))

    expect(await screen.findByText('Could not create the application.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})
