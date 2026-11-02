import { afterEach, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import { listAdminRoleUsers, listAdminUsers } from './admin'

afterEach(() => restoreGlobals())

//spec:covers REQ-FRONTEND-001: ロールの集計はカーソルを辿り、先頭ページのユーザーだけに切り詰めない。
it.each([
  ['role membership', listAdminRoleUsers],
  ['agent owner choices', () => listAdminUsers({ all: true })],
] as const)('reads every user page for %s', async (_purpose, listUsers) => {
  const urls: string[] = []
  stubGlobal(
    'fetch',
    mock(async (url: string) => {
      urls.push(url)
      const last = url.includes('cursor=second')
      return new Response(
        JSON.stringify({
          users: [{ id: last ? 'last-user' : 'first-user', roles: ['support'] }],
          total_users: 201,
        }),
        {
          headers: {
            'Content-Type': 'application/json',
            'Pagination-Total-Items': '201',
            'Pagination-Total-Pages': '2',
            'Pagination-Current-Page': last ? '2' : '1',
            'Pagination-Page-Size': '200',
            ...(!last ? { Link: '</api/admin/v1/users?cursor=second>; rel="next"' } : {}),
          },
        },
      )
    }),
  )
  expect((await listUsers()).map((user) => user.id)).toEqual(['first-user', 'last-user'])
  expect(urls).toHaveLength(2)
  expect(urls[1]).toContain('cursor=second')
})
