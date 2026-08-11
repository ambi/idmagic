import { afterEach, describe, it, expect, mock } from 'bun:test'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { AdminTenantGroupAttributesPage } from './AdminTenantGroupAttributesPage'
import { adminTenantGroupAttributesDictionary } from './AdminTenantGroupAttributesPage.i18n'
import type { GroupAttributeDef, TenantGroupAttributeSchema } from '../../types'

const schema: TenantGroupAttributeSchema = {
  tenant_id: 'tenant-1',
  attributes: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const attribute: GroupAttributeDef = {
  key: 'cost_center',
  label: 'Cost center',
  type: 'string',
  multi_valued: false,
  required: false,
}

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('AdminTenantGroupAttributesPage', () => {
  afterEach(() => restoreGlobals())

  it('renders in English by default', async () => {
    await renderWithRouterBase(<AdminTenantGroupAttributesPage csrfToken="csrf" schema={schema} />)
    expect(
      screen.getByRole('heading', { name: adminTenantGroupAttributesDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(adminTenantGroupAttributesDictionary.en.noCustomAttributesNotice),
    ).toBeInTheDocument()
  })

  it('lists existing attributes with edit/delete controls', async () => {
    await renderWithRouterBase(
      <AdminTenantGroupAttributesPage
        csrfToken="csrf"
        schema={{ ...schema, attributes: [attribute] }}
      />,
    )
    expect(
      screen.getByRole('button', {
        name: adminTenantGroupAttributesDictionary.en.editAttributeAria.replace(
          '{key}',
          'cost_center',
        ),
      }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', {
        name: adminTenantGroupAttributesDictionary.en.deleteAttributeAria.replace(
          '{key}',
          'cost_center',
        ),
      }),
    ).toBeInTheDocument()
  })

  it('adds a new attribute via the dialog and persists it', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.resolve(response(200, { ...schema, attributes: [attribute] }))),
    )
    await renderWithRouterBase(<AdminTenantGroupAttributesPage csrfToken="csrf" schema={schema} />)
    fireEvent.click(
      screen.getByRole('button', {
        name: new RegExp(adminTenantGroupAttributesDictionary.en.addAttribute),
      }),
    )
    fireEvent.change(screen.getByLabelText(adminTenantGroupAttributesDictionary.en.keyFieldLabel), {
      target: { value: 'cost_center' },
    })
    fireEvent.click(
      screen.getByRole('button', { name: adminTenantGroupAttributesDictionary.en.add }),
    )
    await waitFor(() =>
      expect(
        screen.getByRole('button', {
          name: adminTenantGroupAttributesDictionary.en.editAttributeAria.replace(
            '{key}',
            'cost_center',
          ),
        }),
      ).toBeInTheDocument(),
    )
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/admin/v1/tenant/group_attribute_schema'),
      expect.objectContaining({ method: 'PUT' }),
    )
  })
})
