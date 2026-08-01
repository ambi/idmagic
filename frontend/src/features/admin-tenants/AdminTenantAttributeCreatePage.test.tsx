import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminTenantAttributeCreatePage } from './AdminTenantAttributeCreatePage'
import { adminTenantAttributesDictionary } from './AdminTenantAttributesPage.i18n'

const t = adminTenantAttributesDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('AdminTenantAttributeCreatePage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('adds the new attribute to the existing schema and redirects to the list', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { attributes: [{ key: 'region' }], builtin: [] }))),
    )
    await renderWithRouter(
      <AdminTenantAttributeCreatePage csrfToken="csrf" existingAttributes={[]} />,
    )

    fireEvent.change(screen.getByLabelText(t.keyFieldLabel), { target: { value: 'region' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/tenant/attributes'),
    )
  })

  it('shows an error and keeps the form when saving fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(400, { message: 'Could not save the schema.' }))),
    )
    await renderWithRouter(
      <AdminTenantAttributeCreatePage csrfToken="csrf" existingAttributes={[]} />,
    )

    fireEvent.change(screen.getByLabelText(t.keyFieldLabel), { target: { value: 'region' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText('Could not save the schema.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})
