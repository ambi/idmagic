import { redirect } from '@tanstack/react-router'
import { request, tenantBasePath } from '../api/core'
import { ensureLoggedIn, type PortalAudience } from '../api/oidc'

export type AccountContextResponse = {
  csrf_token: string
  sub: string
  preferred_username?: string
  tenant_id?: string
  roles?: string[]
}

export function hasAdminRole(roles: string[] | undefined): boolean {
  return (roles ?? []).some((role) => role === 'admin' || role === 'system_admin')
}

export async function requirePortalAccount(
  audience: PortalAudience,
  pathname: string,
  search: string,
): Promise<AccountContextResponse> {
  await ensureLoggedIn(audience, `${tenantBasePath()}${pathname}${search}`)
  return request<AccountContextResponse>('/api/auth/account')
}

// requireSystemAccount はシステムコンソール (/system) 用ガード。admin ポータルで
// 認証したうえで system_admin ロールを必須とし、持たなければテナント管理コンソール
// へ送り返す。path ではなく role でゲートするため、誤って全テナント画面が露出する
// ことを防ぐ (サーバ側 API も system_admin を要求しており多層防御)。
export async function requireSystemAccount(
  pathname: string,
  search: string,
): Promise<AccountContextResponse> {
  const account = await requirePortalAccount('admin', pathname, search)
  if (!(account.roles ?? []).includes('system_admin')) {
    throw redirect({ to: '/admin' })
  }
  return account
}
