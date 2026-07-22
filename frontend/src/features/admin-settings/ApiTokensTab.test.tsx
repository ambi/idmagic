import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderWithRouter } from '../../test/renderWithRouter'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { ApiTokensTab } from './ApiTokensTab'

const t = adminSettingsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

describe('ApiTokensTab', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('selects scopes, issues a prefixed token once, lists scopes, and revokes it', async () => {
    const meta = {
      id: 'token-1',
      description: 'Okta',
      scopes: ['scim:users:read', 'scim:users:write'],
      created_at: '2026-07-23T00:00:00Z',
    }
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(response(200, { tokens: [] }))
        .mockResolvedValueOnce(response(201, { token: `idmagic_pat_${'a'.repeat(64)}`, meta }))
        .mockResolvedValueOnce(response(200, { tokens: [meta] }))
        .mockResolvedValueOnce(response(204)),
    )

    await renderWithRouter(<ApiTokensTab csrfToken="csrf" tenantRealm="default" />)
    await screen.findByText(t.noTokensNotice)
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))
    fireEvent.change(screen.getByLabelText(t.tokenDescriptionLabel), { target: { value: 'Okta' } })
    fireEvent.click(screen.getByLabelText('scim:users:read'))
    fireEvent.click(screen.getByLabelText('scim:users:write'))
    fireEvent.click(screen.getByRole('button', { name: t.issueToken }))

    expect(await screen.findByDisplayValue(`idmagic_pat_${'a'.repeat(64)}`)).toBeInTheDocument()
    expect(await screen.findByText('scim:users:read')).toBeInTheDocument()
    const post = vi.mocked(fetch).mock.calls.find(([, init]) => init?.method === 'POST')
    expect(post?.[1]?.body).toBe(
      JSON.stringify({
        description: 'Okta',
        scopes: ['scim:users:read', 'scim:users:write'],
        expiry_days: 7,
      }),
    )

    fireEvent.click(screen.getByRole('button', { name: t.revoke }))
    await waitFor(() => expect(screen.queryByText('scim:users:read')).not.toBeInTheDocument())
  })
})
