import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { DelegatedAuthenticationPage } from './DelegatedAuthenticationPage'
import { delegatedAuthenticationDictionary } from './DelegatedAuthenticationPage.i18n'

const t = delegatedAuthenticationDictionary.en
const connector = {
  id: 'connector/1',
  display_name: 'Corp AD',
  status: 'active' as const,
  created_at: '2026-11-12T00:00:00Z',
  encryption_key_id: 'encryption-key',
  delegated_authentication_enabled: false,
  accidental_deletion_count_threshold: 500,
  accidental_deletion_percent_threshold: 20,
}
const response = (status: number, body: unknown) => ({
  ok: status < 300,
  status,
  json: async () => body,
  headers: new Headers(),
})
afterEach(restoreGlobals)

describe('委譲認証の設定と診断', () => {
  //spec:covers REQ-SOURCING-028: 試験中にパスワードを消し、返された分類だけを表示する。
  it('試験の秘密を送信直後に消す', async () => {
    let finish!: (value: ReturnType<typeof response>) => void
    const fetch = mock(
      (_url: unknown, _options: unknown) =>
        new Promise<ReturnType<typeof response>>((resolve) => {
          finish = resolve
        }),
    )
    stubGlobal('fetch', fetch)
    await renderWithRouter(<DelegatedAuthenticationPage csrfToken="csrf" connector={connector} />)
    fireEvent.change(screen.getByLabelText(t.userID), { target: { value: 'directory-user' } })
    fireEvent.change(screen.getByLabelText(t.password), { target: { value: 'directory-secret' } })
    fireEvent.click(screen.getByRole('button', { name: t.test }))
    await waitFor(() =>
      expect((screen.getByLabelText(t.password) as HTMLInputElement).value).toBe(''),
    )
    expect(fetch.mock.calls[0]?.[0]).toContain('connector%2F1/delegated-authentication/test')
    finish(response(200, { result: 'directory_unavailable' }))
    expect(await screen.findByRole('status')).toHaveTextContent(t.directory_unavailable)
    expect(document.body.textContent).not.toContain('directory-secret')
  })

  //spec:covers REQ-SOURCING-027: 管理者が変更を確認してから保存し、返された状態を表示する。
  it('設定変更の影響を確認してから有効にする', async () => {
    const fetch = mock().mockResolvedValue(
      response(200, { ...connector, delegated_authentication_enabled: true }),
    )
    stubGlobal('fetch', fetch)
    await renderWithRouter(<DelegatedAuthenticationPage csrfToken="csrf" connector={connector} />)
    fireEvent.click(screen.getByRole('button', { name: t.enable }))
    expect(fetch).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: t.confirm }))
    expect(await screen.findByRole('button', { name: t.disable })).toBeTruthy()
    expect(fetch.mock.calls[0]?.[1].method).toBe('PUT')
  })

  //spec:covers REQ-SOURCING-028: 試験の失敗を成功として表示せず、入力した秘密を残さない。
  it('試験の失敗を持続するエラーで表示する', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(403, { detail: 'Forbidden' })))
    await renderWithRouter(<DelegatedAuthenticationPage csrfToken="csrf" connector={connector} />)
    fireEvent.change(screen.getByLabelText(t.userID), { target: { value: 'directory-user' } })
    fireEvent.change(screen.getByLabelText(t.password), { target: { value: 'directory-secret' } })
    fireEvent.click(screen.getByRole('button', { name: t.test }))
    expect(await screen.findByRole('alert')).toHaveTextContent(t.testFailed)
    expect((screen.getByLabelText(t.password) as HTMLInputElement).value).toBe('')
    expect(screen.queryByRole('status')).toBeNull()
  })
})
