import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import {
  checkAccess,
  findAuthorizationModel,
  listAccessibleResources,
  listRelationTuples,
  publishAuthorizationModel,
  RELATION_TUPLE_LIST_LIMIT,
  writeRelationTuples,
} from './admin'

function stubFetch(status: number, body: unknown) {
  const fetcher = mock(
    async (_input: string, _init?: RequestInit) =>
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': status < 400 ? 'application/json' : 'application/problem+json' },
      }),
  )
  stubGlobal('fetch', fetcher)
  return fetcher
}

function lastCall(fetcher: ReturnType<typeof stubFetch>) {
  const call = fetcher.mock.calls.at(-1)
  return { url: String(call?.[0]), init: call?.[1] }
}

afterEach(() => {
  restoreGlobals()
})

describe('authorization admin API', () => {
  it('reads the latest model and a named version', async () => {
    const fetcher = stubFetch(200, { authorization_model: { version: 2 }, consistency: 'c' })
    expect((await findAuthorizationModel())?.consistency).toBe('c')
    expect(lastCall(fetcher).url).toEndWith('/api/admin/v1/authorization/model')
    await findAuthorizationModel(2)
    expect(lastCall(fetcher).url).toEndWith('/api/admin/v1/authorization/model?version=2')
  })

  it('treats only the missing model as absent', async () => {
    stubFetch(404, { type: 'urn:idmagic:error:authorization_model_not_found', status: 404 })
    expect(await findAuthorizationModel()).toBeNull()
    stubFetch(403, { type: 'urn:idmagic:error:access_denied', status: 403 })
    await expect(findAuthorizationModel()).rejects.toThrow()
  })

  it('publishes the resource types under their contract name', async () => {
    const fetcher = stubFetch(201, { authorization_model: {}, consistency: 'c' })
    await publishAuthorizationModel('csrf', [{ name: 'user', relations: [] }])
    const { init } = lastCall(fetcher)
    expect(init?.method).toBe('POST')
    expect(JSON.parse(String(init?.body))).toEqual({
      resource_types: [{ name: 'user', relations: [] }],
    })
  })

  it('asks for the limit and only the filled filters', async () => {
    const fetcher = stubFetch(200, { tuples: [], consistency: 'c' })
    await listRelationTuples({ resource_type: 'doc', relation: '' })
    const query = new URL(lastCall(fetcher).url, 'http://localhost').searchParams
    expect(Object.fromEntries(query)).toEqual({
      limit: String(RELATION_TUPLE_LIST_LIMIT),
      resource_type: 'doc',
    })
  })

  it('writes tuples, checks access, and enumerates with their contract paths', async () => {
    const fetcher = stubFetch(200, { result: { permitted: true }, written_count: 1 })
    await writeRelationTuples('csrf', { delete_objects: [{ type: 'doc', id: 'd' }] })
    expect(lastCall(fetcher).url).toEndWith('/api/admin/v1/authorization/relation-tuples')
    const input = { resource_type: 'doc', relation: 'r', subject_type: 'user', subject_id: 'u' }
    expect((await checkAccess('csrf', { ...input, resource_id: 'd' })).permitted).toBe(true)
    expect(lastCall(fetcher).url).toEndWith('/api/admin/v1/authorization/check')
    await listAccessibleResources('csrf', input)
    expect(lastCall(fetcher).url).toEndWith('/api/admin/v1/authorization/list-accessible-resources')
  })
})
