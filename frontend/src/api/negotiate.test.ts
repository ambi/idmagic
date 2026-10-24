import { afterEach, expect, mock, test } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import { loginWithNegotiate } from './authFlow'

afterEach(() => restoreGlobals())

//spec:covers REQ-AUTHENTICATION-043: チケットのない端末でチャレンジを一度だけ再送し、フォームへの復帰を返す。
test('Negotiate retries one challenge and returns normal login fallback', async () => {
  const send = mock()
    .mockResolvedValueOnce(
      new Response('Negotiate', { status: 401, headers: { 'WWW-Authenticate': 'Negotiate' } }),
    )
    .mockResolvedValueOnce(Response.json({}))
  stubGlobal('fetch', send)
  expect(await loginWithNegotiate('csrf', '/admin')).toEqual({})
  expect(send).toHaveBeenCalledTimes(2)
  expect(send.mock.calls[0]?.[1]).toMatchObject({
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': 'csrf' },
  })
})

//spec:covers REQ-AUTHENTICATION-042: 不正なチケットの拒否を通常ログインの成功として扱わず、再要求しない。
test('Negotiate rejection is an error and never triggers a challenge loop', async () => {
  const send = mock().mockResolvedValue(
    new Response('Negotiate authentication rejected', { status: 401 }),
  )
  stubGlobal('fetch', send)
  await expect(loginWithNegotiate('csrf', '/admin')).rejects.toThrow()
  expect(send).toHaveBeenCalledTimes(1)
})
