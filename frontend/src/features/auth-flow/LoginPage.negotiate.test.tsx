import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, mock, test } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { LoginPage } from './LoginPage'
import { loginPageDictionary } from './LoginPage.i18n'

afterEach(() => {
  restoreGlobals()
})

function network(automatic: boolean) {
  const post = mock().mockResolvedValue(Response.json({}))
  stubGlobal(
    'fetch',
    mock((input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith('/api/auth/negotiate')) {
        return init?.method === 'POST'
          ? post(input, init)
          : Promise.resolve(Response.json({ enabled: true, automatic }))
      }
      return Promise.resolve(Response.json({ providers: [] }))
    }),
  )
  return post
}

//spec:covers REQ-AUTHENTICATION-043: 利用可能性の取得が応答しなくても、非参加端末からフォームを使える。
test('normal login is visible while Windows availability is pending', () => {
  stubGlobal(
    'fetch',
    mock(() => new Promise<Response>(() => {})),
  )
  render(<LoginPage csrfToken="csrf" />)
  expect(screen.getByLabelText(loginPageDictionary.en.usernameLabel)).toBeEnabled()
})

//spec:covers REQ-AUTHENTICATION-043: 既定では自動要求せず、利用者が Windows を選んだときだけ試行する。
test('Windows authentication is an explicit choice by default', async () => {
  const post = network(false)
  render(<LoginPage csrfToken="csrf" returnTo="/admin" />)
  const button = await screen.findByRole('button', { name: loginPageDictionary.en.windowsSignIn })
  expect(post).toHaveBeenCalledTimes(0)
  fireEvent.click(button)
  await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
  expect(screen.getByLabelText(loginPageDictionary.en.usernameLabel)).toBeEnabled()
})

//spec:covers REQ-AUTHENTICATION-043: チケットを送れなかった自動試行からフォームへ戻り、再描画で再試行しない。
test('automatic authentication falls back once and keeps normal login available', async () => {
  const post = network(true)
  const page = render(<LoginPage csrfToken="csrf" returnTo="/admin" />)
  await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
  await waitFor(() =>
    expect(screen.getByLabelText(loginPageDictionary.en.usernameLabel)).toBeEnabled(),
  )
  page.rerender(<LoginPage csrfToken="csrf" returnTo="/admin" />)
  expect(post).toHaveBeenCalledTimes(1)
  expect(screen.getByRole('link', { name: loginPageDictionary.en.standardSignIn })).toHaveAttribute(
    'href',
    '/login?negotiate=skip&return_to=%2Fadmin',
  )
})

//spec:covers REQ-AUTHENTICATION-043: 通常ログインを選択した試行では自動 Negotiate を省く。
test('normal login escape skips automatic authentication', async () => {
  stubGlobal('location', { ...window.location, search: '?negotiate=skip' })
  const post = network(true)
  render(<LoginPage csrfToken="csrf" />)
  await screen.findByRole('button', { name: loginPageDictionary.en.windowsSignIn })
  expect(post).toHaveBeenCalledTimes(0)
  expect(screen.getByLabelText(loginPageDictionary.en.usernameLabel)).toBeEnabled()
})
