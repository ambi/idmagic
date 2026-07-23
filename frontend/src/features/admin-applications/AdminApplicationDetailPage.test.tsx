import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminApplicationDetailPage } from './AdminApplicationDetailPage'
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
  protocol: { type: 'oidc', client_id: 'client-1' },
  category_ids: [],
  category_names: [],
  assigned_subject_count: 0,
  sign_in_policy_summary: '',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const detail: AdminApplicationDetail = { application: app }

describe('AdminApplicationDetailPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('shows an error and keeps the confirmation open when deletion fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(response(409, { message: 'Could not delete the application.' }))),
    )
    await renderWithRouter(<AdminApplicationDetailPage csrfToken="csrf" detail={detail} />)

    fireEvent.click(screen.getByRole('button', { name: t.delete }))
    fireEvent.click(screen.getByRole('button', { name: t.confirmDelete }))

    expect(await screen.findByText('Could not delete the application.')).toBeInTheDocument()
  })
})
