import { afterEach, describe, expect, it, mock } from 'bun:test'
import { fireEvent, screen } from '@testing-library/react'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { ClientSecretRotationPanel } from './ClientSecretRotationPanel'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import type { ClientSecretCredentialMetadata } from '../../types'

const t = adminApplicationsDictionary.en
const ja = adminApplicationsDictionary.ja
const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const credentials: ClientSecretCredentialMetadata[] = [
  {
    credential_id: 'active-credential',
    created_at: '2026-07-01T00:00:00Z',
    expires_at: '2026-10-01T00:00:00Z',
    status: 'Active',
  },
  {
    credential_id: 'revoked-credential',
    created_at: '2026-06-01T00:00:00Z',
    revoked_at: '2026-07-01T00:00:00Z',
    status: 'Revoked',
  },
  {
    credential_id: 'expired-credential',
    created_at: '2026-05-01T00:00:00Z',
    expires_at: '2026-06-01T00:00:00Z',
    status: 'Expired',
  },
]

describe('ClientSecretRotationPanel', () => {
  afterEach(() => restoreGlobals())

  //spec:covers EX-APPLICATION-002-01, EX-APPLICATION-002-03: 資格情報の一覧は作成日、有効期限、
  // `Active` / `Expired` / `Revoked` の 3 状態を示し、個別失効の操作は `Active` の 1 件にだけ現れる。
  it('credential の作成日・有効期限・3状態を一覧し、Active だけ失効できる', async () => {
    await renderWithRouter(
      <ClientSecretRotationPanel
        applicationID="app-1"
        csrfToken="csrf"
        initialCredentials={credentials}
        onError={mock()}
      />,
    )

    expect(screen.getByRole('heading', { name: t.secretManagementHeading })).toBeInTheDocument()
    expect(screen.getByText(t.secretStatusActive)).toBeInTheDocument()
    expect(screen.getByText(t.secretStatusExpired)).toBeInTheDocument()
    expect(screen.getByText(t.secretStatusRevoked)).toBeInTheDocument()
    expect(screen.getByText(t.secretNeverExpires)).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: t.secretRevokeButton })).toHaveLength(1)
  })

  it('日本語の状態バッジは内容に依存しない幅を持ち、失効済みを折り返さない', async () => {
    await renderWithRouter(
      <ClientSecretRotationPanel
        applicationID="app-1"
        csrfToken="csrf"
        initialCredentials={credentials}
        onError={mock()}
      />,
      { locale: 'ja' },
    )

    for (const label of [ja.secretStatusActive, ja.secretStatusRevoked]) {
      expect(screen.getByText(label)).toHaveClass('min-w-20', 'justify-center', 'whitespace-nowrap')
    }
  })

  //spec:covers EX-APPLICATION-002-01: 90 日の有効期限を選んだ追加発行は、新しいシークレットの平文を
  // 一度だけ示し、一覧へ作成日、有効期限、`Active` を持つ行を足す。
  it('期限付き secret を追加発行し、平文を一度限り表示して一覧を更新する', async () => {
    const issued = {
      credential_id: 'new-credential',
      created_at: '2026-08-01T00:00:00Z',
      expires_at: '2026-10-30T00:00:00Z',
      status: 'Active' as const,
    }
    const fetchMock = mock().mockResolvedValue(
      response(201, {
        client_secret: 'one-time-secret',
        credential: issued,
        credentials: [credentials[0], issued],
      }),
    )
    stubGlobal('fetch', fetchMock)
    await renderWithRouter(
      <ClientSecretRotationPanel
        applicationID="app-1"
        csrfToken="csrf"
        initialCredentials={[credentials[0]]}
        onError={mock()}
      />,
    )

    fireEvent.change(screen.getByLabelText(t.secretExpiryLabel), { target: { value: '90' } })
    fireEvent.click(screen.getByRole('button', { name: t.secretIssueButton }))

    expect(await screen.findByText('one-time-secret')).toBeInTheDocument()
    expect(screen.getByText('new-credential')).toBeInTheDocument()
    const call = fetchMock.mock.calls[0]
    expect(call[1].body).toBe(JSON.stringify({ expires_in_days: 90 }))
  })

  //spec:covers EX-APPLICATION-002-01: 個別失効は指定した資格情報だけを `Revoked` にし、失効後は
  // その行の失効操作を出さない。
  it('確認後に指定 credential だけを失効し、再取得結果で一覧を更新する', async () => {
    const revoked = {
      ...credentials[0],
      revoked_at: '2026-08-01T00:00:00Z',
      status: 'Revoked' as const,
    }
    stubGlobal('fetch', mock().mockResolvedValue(response(200, { credentials: [revoked] })))
    await renderWithRouter(
      <ClientSecretRotationPanel
        applicationID="app-1"
        csrfToken="csrf"
        initialCredentials={[credentials[0]]}
        onError={mock()}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: t.secretRevokeButton }))
    expect(screen.getByText(t.secretRevokeConfirmMessage)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: t.secretRevokeConfirmButton }))

    expect(await screen.findByText(t.secretStatusRevoked)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: t.secretRevokeButton })).not.toBeInTheDocument()
  })

  //spec:covers EX-APPLICATION-002-02: `Active` の資格情報が既に 2 件あるとき、追加発行の操作を無効にし、
  // 先に既存の資格情報を失効するよう案内する。
  it('Active credential が2件なら追加発行を無効化して先に失効するよう案内する', async () => {
    const second = { ...credentials[0], credential_id: 'second-active' }
    await renderWithRouter(
      <ClientSecretRotationPanel
        applicationID="app-1"
        csrfToken="csrf"
        initialCredentials={[credentials[0], second]}
        onError={mock()}
      />,
    )

    expect(screen.getByRole('button', { name: t.secretIssueButton })).toBeDisabled()
    expect(screen.getByText(t.secretLimitNotice)).toBeInTheDocument()
  })
})
