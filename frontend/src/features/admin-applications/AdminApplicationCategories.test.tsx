import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { CategoryManager } from './AdminApplicationCategories'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import type { AdminApplication, ApplicationCategory } from '../../types'

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

const category: ApplicationCategory = {
  category_id: 'cat-1',
  name: 'Finance',
  position: 0,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

function stubFetch(
  handler: (url: string, init?: RequestInit) => ReturnType<typeof response> | undefined,
) {
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string, init?: RequestInit) => {
      const result = handler(url, init)
      if (result) return Promise.resolve(result)
      throw new Error(`unexpected fetch ${url}`)
    }),
  )
}

describe('CategoryManager', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('shows the empty state when the tenant has no categories yet', async () => {
    stubFetch((url) => {
      if (url.includes('/application-categories')) return response(200, { categories: [] })
      return undefined
    })
    await renderWithRouter(<CategoryManager app={app} csrfToken="csrf" onError={() => {}} />)
    expect(await screen.findByText(t.noCategoriesNotice)).toBeInTheDocument()
  })

  it('toggles a category on and persists the new assignment set', async () => {
    let putBody: unknown
    stubFetch((url, init) => {
      if (url.includes('/application-categories')) return response(200, { categories: [category] })
      if (url.includes('/categories') && init?.method === 'PUT') {
        putBody = JSON.parse(String(init.body))
        return response(200, { ...app, category_ids: ['cat-1'] })
      }
      return undefined
    })
    await renderWithRouter(<CategoryManager app={app} csrfToken="csrf" onError={() => {}} />)

    const checkbox = await screen.findByRole('checkbox', { name: 'Finance' })
    expect(checkbox).not.toBeChecked()
    fireEvent.click(checkbox)

    await waitFor(() => expect(checkbox).toBeChecked())
    expect(putBody).toEqual({ category_ids: ['cat-1'] })
  })

  it('adds a new category from the input field', async () => {
    stubFetch((url, init) => {
      if (url.includes('/application-categories') && init?.method === 'POST') {
        return response(201, { category: { ...category, category_id: 'cat-2', name: 'Sales' } })
      }
      if (url.includes('/application-categories')) return response(200, { categories: [] })
      return undefined
    })
    await renderWithRouter(<CategoryManager app={app} csrfToken="csrf" onError={() => {}} />)

    await screen.findByText(t.noCategoriesNotice)
    fireEvent.change(screen.getByPlaceholderText(t.newCategoryPlaceholder), {
      target: { value: 'Sales' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.add }))

    expect(await screen.findByText('Sales')).toBeInTheDocument()
  })

  it('removes a category', async () => {
    stubFetch((url, init) => {
      if (url.includes('/application-categories') && init?.method === 'DELETE') {
        return response(204)
      }
      if (url.includes('/application-categories')) return response(200, { categories: [category] })
      return undefined
    })
    await renderWithRouter(<CategoryManager app={app} csrfToken="csrf" onError={() => {}} />)

    await screen.findByText('Finance')
    fireEvent.click(
      screen.getByRole('button', { name: t.deleteCategoryAria.replace('{name}', 'Finance') }),
    )

    await waitFor(() => expect(screen.queryByText('Finance')).not.toBeInTheDocument())
  })

  it('reports a fetch failure through onError', async () => {
    stubFetch((url) => {
      if (url.includes('/application-categories')) {
        return response(500, { message: 'Could not load categories.' })
      }
      return undefined
    })
    const onError = vi.fn()
    await renderWithRouter(<CategoryManager app={app} csrfToken="csrf" onError={onError} />)
    await waitFor(() => expect(onError).toHaveBeenCalledWith('Could not load categories.'))
  })
})
