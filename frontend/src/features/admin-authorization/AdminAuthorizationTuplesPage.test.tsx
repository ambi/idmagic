import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { RELATION_TUPLE_LIST_LIMIT } from '../../api/admin'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { RelationTuple } from '../../types'
import { AdminAuthorizationTuplesPage } from './AdminAuthorizationTuplesPage'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'

const t = adminAuthorizationDictionary.en
const originalFetch = globalThis.fetch

const tuple: RelationTuple = {
  resource_type: 'doc',
  resource_id: 'd1',
  relation: 'viewer',
  subject_type: 'group',
  subject_id: 'eng',
  subject_relation: 'member',
}

function tuples(count: number): RelationTuple[] {
  return Array.from({ length: count }, (_, index) => ({ ...tuple, resource_id: `d${index}` }))
}

function stubAPI(remaining: RelationTuple[] = []) {
  const posts: unknown[] = []
  globalThis.fetch = mock(async (_input: RequestInfo | URL, init?: RequestInit) => {
    if (init?.method === 'POST') posts.push(JSON.parse(String(init.body)))
    const body =
      init?.method === 'POST'
        ? { written_count: 0, deleted_count: 1, consistency: 'after-write' }
        : { tuples: remaining, consistency: 'after-write' }
    return new Response(JSON.stringify(body), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }) as unknown as typeof fetch
  return posts
}

async function renderPage(list: RelationTuple[], onFiltersChange = (_: object) => {}) {
  await renderWithRouter(
    <AdminAuthorizationTuplesPage
      csrfToken="csrf"
      tuples={list}
      consistency="initial"
      filters={{}}
      onFiltersChange={onFiltersChange}
    />,
  )
}

afterEach(() => {
  globalThis.fetch = originalFetch
})

describe('AdminAuthorizationTuplesPage', () => {
  //spec:covers REQ-AUTHORIZATION-012: 上限の直前では警告せず、上限に達すると打ち切りの可能性を示す。
  it('warns only when the list reaches the limit', async () => {
    await renderPage(tuples(RELATION_TUPLE_LIST_LIMIT - 1))
    expect(screen.queryByText(t.tuplesLimitReached)).not.toBeInTheDocument()
    await renderPage(tuples(RELATION_TUPLE_LIST_LIMIT))
    expect(screen.getByText(t.tuplesLimitReached)).toBeInTheDocument()
  })

  it('shows the next step when the list is empty', async () => {
    await renderPage([])
    expect(screen.getByText(t.tuplesEmpty)).toBeInTheDocument()
  })

  it('deletes one tuple only after confirmation and keeps the subject relation', async () => {
    const posts = stubAPI()
    await renderPage([tuple])
    expect(screen.getByText('group:eng#member')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: t.delete }))
    expect(posts).toEqual([])
    expect(screen.getByRole('dialog')).toHaveTextContent('doc:d1#viewer')

    fireEvent.click(screen.getByRole('button', { name: t.deleteConfirm }))
    await screen.findByText(t.tuplesEmpty)
    expect(posts).toEqual([{ deletes: [tuple] }])
  })

  it('does not delete when the dialog is cancelled', async () => {
    const posts = stubAPI()
    await renderPage([tuple])
    fireEvent.click(screen.getByRole('button', { name: t.delete }))
    fireEvent.click(screen.getByRole('button', { name: t.cancel }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(posts).toEqual([])
  })

  it('warns about both sides before deleting an object', async () => {
    const posts = stubAPI()
    await renderPage([tuple])
    fireEvent.change(screen.getByLabelText(t.objectType), { target: { value: 'doc' } })
    fireEvent.change(screen.getByLabelText(t.objectId), { target: { value: 'd1' } })
    fireEvent.click(screen.getByRole('button', { name: t.deleteObject }))
    expect(screen.getByRole('dialog')).toHaveTextContent(t.deleteObjectResult)
    expect(posts).toEqual([])
    fireEvent.click(screen.getByRole('button', { name: t.deleteConfirm }))
    await waitFor(() => expect(posts).toEqual([{ delete_objects: [{ type: 'doc', id: 'd1' }] }]))
  })

  it('carries the consistency of the latest write to the access check link', async () => {
    stubAPI()
    await renderPage([tuple])
    expect(screen.getByRole('link', { name: t.sectionCheck }).getAttribute('href')).toContain(
      'consistency=initial',
    )
    fireEvent.click(screen.getByRole('button', { name: t.delete }))
    fireEvent.click(screen.getByRole('button', { name: t.deleteConfirm }))
    await waitFor(() =>
      expect(screen.getByRole('link', { name: t.sectionCheck }).getAttribute('href')).toContain(
        'consistency=after-write',
      ),
    )
  })

  it('passes only the filled filters to the URL state', async () => {
    const onFiltersChange = mock((_: object) => {})
    await renderPage([], onFiltersChange)
    fireEvent.change(screen.getByLabelText(t.resourceType), { target: { value: 'doc' } })
    fireEvent.click(screen.getByRole('button', { name: t.search }))
    expect(onFiltersChange).toHaveBeenCalledWith({ resource_type: 'doc' })
  })
})
