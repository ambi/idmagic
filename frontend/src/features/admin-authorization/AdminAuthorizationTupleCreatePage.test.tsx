import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { ResourceTypeDefinition } from '../../types'
import { AdminAuthorizationTupleCreatePage } from './AdminAuthorizationTupleCreatePage'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'

const t = adminAuthorizationDictionary.en
const originalFetch = globalThis.fetch

const types: ResourceTypeDefinition[] = [
  { name: 'user', relations: [] },
  {
    name: 'doc',
    relations: [
      { name: 'viewer', rewrites: [{ kind: 'direct', direct_subject_types: ['user', 'user:*'] }] },
      { name: 'owner', rewrites: [{ kind: 'computed_userset', computed_relation: 'viewer' }] },
    ],
  },
]

afterEach(() => {
  globalThis.fetch = originalFetch
})

describe('AdminAuthorizationTupleCreatePage', () => {
  it('points to publication when there is no model', async () => {
    await renderWithRouter(
      <AdminAuthorizationTupleCreatePage
        csrfToken="csrf"
        resourceTypes={null}
        onCreated={() => {}}
      />,
    )
    expect(screen.getByText(t.noModelForTuples)).toBeInTheDocument()
  })

  it('offers only relations with a direct rule and writes the wildcard subject', async () => {
    const posts: unknown[] = []
    globalThis.fetch = mock(async (_input: RequestInfo | URL, init?: RequestInit) => {
      posts.push(JSON.parse(String(init?.body)))
      return new Response(
        JSON.stringify({ written_count: 1, deleted_count: 0, consistency: 'c' }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        },
      )
    }) as unknown as typeof fetch
    const onCreated = mock(() => {})
    await renderWithRouter(
      <AdminAuthorizationTupleCreatePage
        csrfToken="csrf"
        resourceTypes={types}
        onCreated={onCreated}
      />,
    )
    fireEvent.change(screen.getByLabelText(t.resourceType), { target: { value: 'doc' } })
    const relation = screen.getByLabelText(t.relation) as HTMLSelectElement
    expect([...relation.options].map((option) => option.value)).toEqual(['', 'viewer'])
    fireEvent.change(relation, { target: { value: 'viewer' } })
    fireEvent.change(screen.getByLabelText(t.subjectForm), { target: { value: 'user:*' } })
    fireEvent.change(screen.getByLabelText(t.resourceId), { target: { value: 'd1' } })
    expect(screen.queryByLabelText(t.subjectId)).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: t.addTuple }))
    await waitFor(() => expect(onCreated).toHaveBeenCalled())
    expect(posts).toEqual([
      {
        writes: [
          {
            resource_type: 'doc',
            resource_id: 'd1',
            relation: 'viewer',
            subject_type: 'user',
            subject_id: '*',
          },
        ],
      },
    ])
  })
})
