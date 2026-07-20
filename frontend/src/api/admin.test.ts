import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as adminAPI from './admin'
import {
  addAdminGroupMember,
  assignApplication,
  attachProtocolBinding,
  bindAdminAgentCredential,
  clearAdminUserRequiredAction,
  createAdminApplication,
  createAdminGroup,
  deleteAdminAgent,
  deleteAdminApplication,
  deleteAdminGroup,
  deleteAdminUser,
  detachProtocolBinding,
  disableAdminAgent,
  disableTenantKey,
  enableAdminAgent,
  killAdminAgent,
  listAdminUserSessions,
  removeAdminGroupMember,
  restoreAdminUser,
  revokeAdminUserSession,
  revokeAllAdminUserSessions,
  revokeScimToken,
  rotateTenantSigningKey,
  setAdminUserDisabled,
  setAdminUserRequiredAction,
  unassignApplication,
  unbindAdminAgentCredential,
  updateAdminAgent,
  updateAdminApplication,
  updateAdminGroup,
  updateAdminSettings,
  updateAdminUser,
  updateApplicationOidcConfig,
  updateApplicationSamlConfig,
  updateApplicationWsFedConfig,
  updateTenantUserAttributeSchema,
} from './admin'
import type { AuthenticationAPIError } from './core'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

describe('admin API client', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(204))))
  afterEach(() => vi.unstubAllGlobals())

  it('MCP resource server 管理契約を公開する', () => {
    expect(adminAPI).toEqual(
      expect.objectContaining({
        listMcpResourceServers: expect.any(Function),
        createMcpResourceServer: expect.any(Function),
        updateMcpResourceServer: expect.any(Function),
        deleteMcpResourceServer: expect.any(Function),
      }),
    )
  })

  it('MCP resource server 管理契約を正しい envelope・URL・CSRF で送受信する', async () => {
    const resourceServer = {
      tenant_id: 'tenant-1',
      resource_server_id: 'resource/a b',
      resource: 'https://mcp.example.com',
      name: 'Example MCP',
      scopes: ['mcp.read'],
      state: 'Active' as const,
      created_at: '2026-07-20T00:00:00Z',
      updated_at: '2026-07-20T00:00:00Z',
    }
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(response(200, { resource_servers: [resourceServer] }))
        .mockResolvedValueOnce(response(201, resourceServer))
        .mockResolvedValueOnce(response(200, resourceServer))
        .mockResolvedValueOnce(response(204)),
    )

    await expect(adminAPI.listMcpResourceServers()).resolves.toEqual([resourceServer])
    await adminAPI.createMcpResourceServer('csrf', {
      resource: resourceServer.resource,
      name: resourceServer.name,
      scopes: resourceServer.scopes,
      state: resourceServer.state,
    })
    await adminAPI.updateMcpResourceServer('csrf', resourceServer.resource_server_id, {
      name: 'Renamed MCP',
      scopes: ['mcp.read', 'mcp.write'],
      state: 'Disabled',
    })
    await adminAPI.deleteMcpResourceServer('csrf', resourceServer.resource_server_id)

    const calls = vi.mocked(fetch).mock.calls
    expect(calls.map(([url]) => url)).toEqual([
      expect.stringContaining('/api/admin/mcp-resource-servers'),
      expect.stringContaining('/api/admin/mcp-resource-servers'),
      expect.stringContaining('/api/admin/mcp-resource-servers/resource%2Fa%20b'),
      expect.stringContaining('/api/admin/mcp-resource-servers/resource%2Fa%20b'),
    ])
    expect(calls[1][1]).toEqual(expect.objectContaining({ method: 'POST' }))
    expect(calls[2][1]).toEqual(
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({
          name: 'Renamed MCP',
          scopes: ['mcp.read', 'mcp.write'],
          state: 'Disabled',
        }),
      }),
    )
    expect(calls[3][1]).toEqual(expect.objectContaining({ method: 'DELETE' }))
    expect(
      calls.slice(1).every(([, init]) => new Headers(init?.headers).get('X-CSRF-Token') === 'csrf'),
    ).toBe(true)
  })

  it('ユーザーとグループの破壊的な管理操作を CSRF 保護して送る', async () => {
    const id = 'user/a b'
    await updateAdminUser('csrf', id, { roles: ['admin'] })
    await setAdminUserRequiredAction('csrf', id, 'reset/password')
    await clearAdminUserRequiredAction('csrf', id, 'reset/password')
    await setAdminUserDisabled('csrf', id, true)
    await setAdminUserDisabled('csrf', id, false)
    await deleteAdminUser('csrf', id)
    await deleteAdminUser('csrf', id, { purge: true })
    await restoreAdminUser('csrf', id)
    await createAdminGroup('csrf', { name: 'operators' })
    await updateAdminGroup('csrf', 'group/a b', { description: 'updated' })
    await addAdminGroupMember('csrf', 'group/a b', id)
    await removeAdminGroupMember('csrf', 'group/a b', id)
    await deleteAdminGroup('csrf', 'group/a b')

    const calls = vi.mocked(fetch).mock.calls
    expect(calls.map(([url]) => url)).toEqual(
      expect.arrayContaining([
        expect.stringContaining('/api/admin/users/user%2Fa%20b'),
        expect.stringContaining('/required_actions/reset%2Fpassword'),
        expect.stringContaining('/api/admin/users/user%2Fa%20b/disable'),
        expect.stringContaining('/api/admin/users/user%2Fa%20b/enable'),
        expect.stringContaining('/api/admin/users/user%2Fa%20b?purge=true'),
        expect.stringContaining('/api/admin/groups/group%2Fa%20b/members/user%2Fa%20b'),
      ]),
    )
    expect(
      calls.every(([, init]) => new Headers(init?.headers).get('X-CSRF-Token') === 'csrf'),
    ).toBe(true)
    expect(
      calls.find(([url]) => String(url).includes('/users/user%2Fa%20b?purge=true'))?.[1],
    ).toEqual(expect.objectContaining({ method: 'DELETE' }))
  })

  it('セッション管理 (wi-28 T007) が正しい URL に CSRF 保護付きで送る', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, { sessions: [] })))
    await listAdminUserSessions('user/a b')
    await revokeAdminUserSession('csrf', 'user/a b', 'session/1')
    await revokeAllAdminUserSessions('csrf', 'user/a b')

    const calls = vi.mocked(fetch).mock.calls
    expect(calls.map(([url]) => url)).toEqual([
      expect.stringContaining('/api/admin/users/user%2Fa%20b/sessions'),
      expect.stringContaining('/api/admin/users/user%2Fa%20b/sessions/session%2F1/revoke'),
      expect.stringContaining('/api/admin/users/user%2Fa%20b/sessions/revoke_all'),
    ])
    expect(new Headers(calls[0][1]?.headers).get('X-CSRF-Token')).toBeNull()
    expect(calls[1][1]).toEqual(expect.objectContaining({ method: 'POST' }))
    expect(calls[2][1]).toEqual(expect.objectContaining({ method: 'POST' }))
    expect(
      calls.slice(1).every(([, init]) => new Headers(init?.headers).get('X-CSRF-Token') === 'csrf'),
    ).toBe(true)
  })

  it('エージェントとアプリケーションの高リスク操作を正しい URL と本文へ直列化する', async () => {
    const id = 'app/alpha beta'
    await updateAdminAgent('csrf', 'agent/a b', { name: 'Agent' })
    await disableAdminAgent('csrf', 'agent/a b')
    await enableAdminAgent('csrf', 'agent/a b')
    await killAdminAgent('csrf', 'agent/a b')
    await bindAdminAgentCredential('csrf', 'agent/a b', 'client/a b')
    await unbindAdminAgentCredential('csrf', 'agent/a b', 'client/a b')
    await deleteAdminAgent('csrf', 'agent/a b')
    await createAdminApplication('csrf', {
      name: 'Portal',
      type: 'oidc',
      redirect_uris: ['https://app/callback'],
    })
    await updateAdminApplication('csrf', id, { status: 'disabled' })
    await updateApplicationOidcConfig('csrf', id, { scope: 'openid' })
    await updateApplicationWsFedConfig('csrf', id, { audience: 'urn:app' })
    await updateApplicationSamlConfig('csrf', id, { audience: 'urn:app' })
    await attachProtocolBinding('csrf', id, { type: 'oidc', client_id: 'client' })
    await detachProtocolBinding('csrf', id, 'oidc')
    await assignApplication('csrf', id, { subject_type: 'user', subject_id: 'user/a b' })
    await unassignApplication('csrf', id, 'user', 'user/a b')
    await deleteAdminApplication('csrf', id)

    const calls = vi.mocked(fetch).mock.calls
    expect(calls.map(([url]) => url)).toEqual(
      expect.arrayContaining([
        expect.stringContaining('/api/admin/agents/agent%2Fa%20b/credentials/client%2Fa%20b'),
        expect.stringContaining('/api/admin/applications/app%2Falpha%20beta/oidc'),
        expect.stringContaining('/api/admin/applications/app%2Falpha%20beta/wsfed'),
        expect.stringContaining('/api/admin/applications/app%2Falpha%20beta/saml'),
        expect.stringContaining('/assignments/user/user%2Fa%20b'),
      ]),
    )
    expect(calls.find(([url]) => String(url).endsWith('/applications'))?.[1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          name: 'Portal',
          type: 'oidc',
          redirect_uris: ['https://app/callback'],
        }),
      }),
    )
  })

  it('鍵・設定の更新を CSRF 保護し、管理 API の失敗を呼び出し元へ伝える', async () => {
    await rotateTenantSigningKey('csrf')
    await disableTenantKey('csrf', 'kid/a b')
    await updateAdminSettings('csrf', { display_name: 'New tenant' })
    await updateTenantUserAttributeSchema('csrf', [])
    await revokeScimToken('csrf', 'token/a b')

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/admin/keys/kid%2Fa%20b/disable'),
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ 'X-CSRF-Token': 'csrf' }),
      }),
    )
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/admin/tenant/user_attribute_schema'),
      expect.objectContaining({ method: 'PUT', body: JSON.stringify({ attributes: [] }) }),
    )

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(403, { error: 'forbidden', message: 'denied' })),
    )
    await expect(deleteAdminUser('csrf', 'user')).rejects.toEqual(
      expect.objectContaining<Partial<AuthenticationAPIError>>({
        name: 'AuthenticationAPIError',
        code: 'forbidden',
        message: 'denied',
      }),
    )
  })
})
