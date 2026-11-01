import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, mock } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { adminSettingsDictionary } from '../admin-settings/AdminSettingsPage.i18n'
import { AccountApiTokensPage } from './AccountApiTokensPage'

afterEach(() => restoreGlobals())
const t = adminSettingsDictionary.en

//spec:covers REQ-APITOKENS-005: 一般ユーザーの画面は本人のスコープだけを本人用 API へ送信する。
it('issues a personal token with only account scope choices', async () => {
  const fetcher = mock(
    async (_url: RequestInfo | URL, _init?: RequestInit) =>
      new Response(JSON.stringify({ token: 'one-time-personal-token', meta: { id: 'p1' } }), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      }),
  )
  stubGlobal('fetch', fetcher)
  await renderWithRouter(
    <AccountApiTokensPage csrfToken="csrf" username="ordinary-user" isAdmin={false} create />,
  )
  expect(screen.queryByLabelText('users:write')).not.toBeInTheDocument()
  expect(screen.queryByLabelText('scim:users:read')).not.toBeInTheDocument()
  fireEvent.change(screen.getByLabelText(t.tokenDescriptionLabel), {
    target: { value: 'My script' },
  })
  fireEvent.click(screen.getByLabelText('account:read'))
  fireEvent.click(screen.getByRole('button', { name: t.issueToken }))
  await waitFor(() =>
    expect(screen.getByDisplayValue('one-time-personal-token')).toBeInTheDocument(),
  )
  const post = fetcher.mock.calls.find(([, init]) => init?.method === 'POST')
  expect(post?.[0]).toBe('/api/account/v1/api-tokens')
  expect(JSON.parse(String(post?.[1]?.body))).toEqual({
    description: 'My script',
    scopes: ['account:read'],
    expiry_days: 7,
  })
})
