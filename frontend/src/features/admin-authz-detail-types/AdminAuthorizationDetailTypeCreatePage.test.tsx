import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { adminAuthorizationDetailTypesDictionary } from './AdminAuthorizationDetailTypesPage.i18n'
import { AdminAuthorizationDetailTypeCreatePage } from './AdminAuthorizationDetailTypeCreatePage'

const t = adminAuthorizationDetailTypesDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('AdminAuthorizationDetailTypeCreatePage', () => {
  const originalLocation = window.location
  afterEach(() => restoreGlobals())

  it('registers a detail type and redirects to the list', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() =>
        Promise.resolve(
          response(201, {
            type: 'payment_initiation',
            display_template: 'Up to {instructedAmount}',
            state: 'Enabled',
            schema: { rules: [] },
          }),
        ),
      ),
    )
    await renderWithRouter(<AdminAuthorizationDetailTypeCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.typeIdLabel), {
      target: { value: 'payment_initiation' },
    })
    fireEvent.change(screen.getByLabelText(t.displayTemplateLabel), {
      target: { value: 'Up to {instructedAmount}' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith('/admin/authorization-detail-types'),
    )
  })

  it('shows a schema error without submitting when the schema JSON is invalid', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, {}))),
    )
    await renderWithRouter(<AdminAuthorizationDetailTypeCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.typeIdLabel), {
      target: { value: 'payment_initiation' },
    })
    fireEvent.change(screen.getByLabelText(t.displayTemplateLabel), {
      target: { value: 'Up to {instructedAmount}' },
    })
    fireEvent.change(screen.getByLabelText(t.schemaLabel), { target: { value: 'not json' } })
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    expect(await screen.findByText(t.schemaInvalidError)).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  it('shows an error and keeps the form when registration fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(409, { message: 'Failed to save.' }))),
    )
    await renderWithRouter(<AdminAuthorizationDetailTypeCreatePage csrfToken="csrf" />)

    fireEvent.change(screen.getByLabelText(t.typeIdLabel), {
      target: { value: 'payment_initiation' },
    })
    fireEvent.change(screen.getByLabelText(t.displayTemplateLabel), {
      target: { value: 'Up to {instructedAmount}' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.register }))

    expect(await screen.findByText('Failed to save.')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})
