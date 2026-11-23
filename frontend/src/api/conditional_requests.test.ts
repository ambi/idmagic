import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import { commonDictionary } from '../lib/i18n/common.i18n'
import { getCurrentLocale } from '../lib/i18n/currentLocale'
import { conditionalAdminRequest, replaceResource, request } from './core'

function response(body: unknown, tag?: string, status = 200) {
  return {
    ok: status < 400,
    status,
    headers: new Headers(tag ? { ETag: tag } : {}),
    json: async () => body,
  }
}

describe('編集の本文と ETag', () => {
  afterEach(() => restoreGlobals())

  it('新しい GET のタグを古い編集の本文へ流用しない', async () => {
    const fetch = mock()
      .mockResolvedValueOnce(response({ group: { name: 'First' } }, '"first"'))
      .mockResolvedValueOnce(response({ group: { name: 'Second' } }, '"second"'))
    stubGlobal('fetch', fetch)
    const first = await request<{ group: { name: string } }>('/api/admin/v1/groups/1')
    const second = await request<{ group: { name: string } }>('/api/admin/v1/groups/1')
    expect(
      new Headers(conditionalAdminRequest('csrf', 'PATCH', {}, first.group).headers).get(
        'If-Match',
      ),
    ).toBe('"first"')
    expect(
      new Headers(conditionalAdminRequest('csrf', 'PATCH', {}, second.group).headers).get(
        'If-Match',
      ),
    ).toBe('"second"')
  })

  it('PUT の後は GET の本文とタグを一緒に採用する', async () => {
    const fetch = mock()
      .mockResolvedValueOnce(response({ attributes: ['before'] }, '"before"'))
      .mockResolvedValueOnce(response({ attributes: ['normalized'] }))
      .mockResolvedValueOnce(response({ attributes: ['another writer'] }, '"after"'))
    stubGlobal('fetch', fetch)
    const initial = await request<object>('/schema')
    const saved = await replaceResource<{ attributes: string[] }>(
      '/schema',
      'csrf',
      { attributes: ['input'] },
      initial,
    )
    expect(saved.attributes).toEqual(['another writer'])
    expect(fetch.mock.calls[1]?.[1]).toMatchObject({
      method: 'PUT',
      headers: expect.objectContaining({ 'If-Match': '"before"' }),
    })
    expect(
      new Headers(conditionalAdminRequest('csrf', 'PUT', {}, saved).headers).get('If-Match'),
    ).toBe('"after"')
    expect(fetch.mock.calls).toHaveLength(3)
  })

  it('保存後の取得失敗を保存自体の失敗と区別する', async () => {
    const fetch = mock()
      .mockResolvedValueOnce(response({}))
      .mockRejectedValueOnce(new Error('offline'))
    stubGlobal('fetch', fetch)
    await expect(replaceResource('/schema', 'csrf', {})).rejects.toMatchObject({
      message: commonDictionary[getCurrentLocale()].savedRefreshFailed,
    })
    expect(fetch.mock.calls).toHaveLength(2)
  })

  it('競合を表示し、自動再試行もタグだけの読み直しも行わない', async () => {
    const fetch = mock().mockResolvedValue(
      response({ type: 'urn:idmagic:error:precondition_failed', detail: 'stale' }, undefined, 412),
    )
    stubGlobal('fetch', fetch)
    await expect(
      request('/schema', conditionalAdminRequest('csrf', 'PATCH', {})),
    ).rejects.toMatchObject({
      code: 'precondition_failed',
      message: commonDictionary[getCurrentLocale()].editConflict,
    })
    expect(fetch.mock.calls).toHaveLength(1)
  })
})
