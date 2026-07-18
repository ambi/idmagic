import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminApplicationEditPage } from './AdminApplicationEditPage'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import type { AdminApplication, AdminApplicationDetail } from '../../types'

const t = adminApplicationsDictionary.en

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

describe('AdminApplicationEditPage', () => {
  const originalLocation = window.location
  afterEach(() => vi.unstubAllGlobals())

  it('shows an error and keeps the form when saving fails', async () => {
    vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(400, { message: 'Could not update the name.' }))),
    )
    await renderWithRouter(<AdminApplicationEditPage csrfToken="csrf" detail={detail} />)

    fireEvent.change(screen.getByLabelText(t.nameFieldLabel), { target: { value: 'Renamed App' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not update the name.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})
