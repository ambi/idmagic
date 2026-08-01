import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminApplicationEditPage } from './AdminApplicationEditPage'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import type { AdminApplication, AdminApplicationDetail } from '../../types'

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

const detail: AdminApplicationDetail = { application: app }

describe('AdminApplicationEditPage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('shows an error and keeps the form when saving fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(400, { message: 'Could not update the name.' }))),
    )
    await renderWithRouter(<AdminApplicationEditPage csrfToken="csrf" detail={detail} />)

    fireEvent.change(screen.getByLabelText(t.nameFieldLabel), { target: { value: 'Renamed App' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not update the name.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  it('shows the chosen icon file name instead of the native file input text', async () => {
    await renderWithRouter(<AdminApplicationEditPage csrfToken="csrf" detail={detail} />)

    expect(screen.getByText(t.noIconFileChosen)).toBeInTheDocument()

    const pngSignature = new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])
    const file = new File([pngSignature], 'logo.png', { type: 'image/png' })
    const fileInput = screen.getByLabelText(t.iconImageFieldLabel) as HTMLInputElement
    fireEvent.change(fileInput, { target: { files: [file] } })

    expect(await screen.findByText('logo.png')).toBeInTheDocument()
    expect(screen.queryByText(t.noIconFileChosen)).not.toBeInTheDocument()
  })

  it('links to the provisioning settings screen (T019: also reachable from edit, not just detail)', async () => {
    await renderWithRouter(<AdminApplicationEditPage csrfToken="csrf" detail={detail} />)

    expect(screen.getByRole('button', { name: /Provisioning/i })).toHaveAttribute(
      'href',
      '/admin/applications/app-1/provisioning',
    )
  })

  it('places client secret management outside the settings form in its own top-level card', async () => {
    const oidcDetail: AdminApplicationDetail = {
      application: app,
      oidc: {
        client_id: 'client-1',
        client_type: 'confidential',
        redirect_uris: ['https://app.example/callback'],
        grant_types: ['authorization_code'],
        response_types: ['code'],
        token_endpoint_auth_method: 'client_secret_basic',
        scope: 'openid',
        require_pushed_authorization_requests: false,
        dpop_bound_access_tokens: false,
        fapi_profile: 'none',
        client_secret_rotatable: true,
        secret_credentials: [
          {
            credential_id: 'credential-1',
            created_at: '2026-08-01T00:00:00Z',
            status: 'Active',
          },
        ],
        sub_source_attribute: '',
        rules: [],
      },
    }
    await renderWithRouter(<AdminApplicationEditPage csrfToken="csrf" detail={oidcDetail} />)

    const secretRegion = screen.getByRole('region', { name: t.secretManagementHeading })
    const settingsForm = screen.getByRole('button', { name: t.save }).closest('form')
    const clientID = screen.getByText('client-1')

    expect(settingsForm).not.toContainElement(secretRegion)
    expect(clientID.closest('form')).toBe(settingsForm)
    expect(secretRegion).not.toBe(settingsForm?.parentElement)
  })
})
