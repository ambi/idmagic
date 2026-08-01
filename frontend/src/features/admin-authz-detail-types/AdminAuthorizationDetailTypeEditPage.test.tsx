import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AuthorizationDetailType } from '../../types'
import { adminAuthorizationDetailTypesDictionary } from './AdminAuthorizationDetailTypesPage.i18n'
import { AdminAuthorizationDetailTypeEditPage } from './AdminAuthorizationDetailTypeEditPage'

const t = adminAuthorizationDetailTypesDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const detailType: AuthorizationDetailType = {
  tenant_id: 'tenant-1',
  type: 'payment_initiation',
  description: 'Payment initiation',
  schema: { rules: [{ name: 'actions', semantics: 'set', required: true }] },
  display_template: 'Up to {instructedAmount}',
  state: 'Enabled',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('AdminAuthorizationDetailTypeEditPage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('locks the type field and updates the detail type', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { ...detailType, description: 'Updated' }))),
    )
    await renderWithRouter(
      <AdminAuthorizationDetailTypeEditPage csrfToken="csrf" detailType={detailType} />,
    )

    expect(screen.getByLabelText(t.typeIdLabel)).toBeDisabled()

    fireEvent.change(screen.getByLabelText(t.descriptionLabel), {
      target: { value: 'Updated' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.update }))

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/admin/authorization-detail-types/payment_initiation'),
        expect.objectContaining({ method: 'PATCH' }),
      ),
    )
    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/authorization-detail-types'),
    )
  })

  it('shows an error and keeps the form when updating fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(400, { message: 'Failed to save.' }))),
    )
    await renderWithRouter(
      <AdminAuthorizationDetailTypeEditPage csrfToken="csrf" detailType={detailType} />,
    )

    fireEvent.change(screen.getByLabelText(t.descriptionLabel), {
      target: { value: 'Updated' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.update }))

    expect(await screen.findByText('Failed to save.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})
