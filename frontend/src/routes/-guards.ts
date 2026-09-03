import { redirect } from '@tanstack/react-router'
import { request, tenantBasePath, UnauthenticatedError } from '../api/core'
import {
  ensureLoggedIn,
  markPortalAuthenticated,
  type PortalAudience,
  recoverPortalSession,
} from '../api/oidc'

export type AccountContextResponse = {
  csrf_token: string
  sub: string
  preferred_username?: string
  tenant_id?: string
  realm?: string
  roles?: string[]
}

export function hasAdminRole(roles: string[] | undefined): boolean {
  return (roles ?? []).some((role) => role === 'admin' || role === 'system_admin')
}

export function isControlPlaneAccount(account: AccountContextResponse): boolean {
  return account.realm === 'default' && (account.roles ?? []).includes('system_admin')
}

export async function requirePortalAccount(
  audience: PortalAudience,
  pathname: string,
  search: string,
): Promise<AccountContextResponse> {
  const returnTo = `${tenantBasePath()}${pathname}${search}`
  await ensureLoggedIn(audience, returnTo)
  try {
    const account = await request<AccountContextResponse>('/api/auth/account')
    // 有効なセッションを確認できたので復旧ループ抑止マーカーを解除する。
    markPortalAuthenticated(audience)
    return account
  } catch (error) {
    // stale なトークンを提示していた (dev サーバ再起動などでサーバ側セッション/署名鍵が
    // 失われた) 場合、行き止まりにせず保持状態を破棄して 1 回だけ再認可し、元の画面へ戻す。
    if (error instanceof UnauthenticatedError) {
      const recovering = await recoverPortalSession(audience, returnTo)
      if (recovering) {
        // beginLogin がリダイレクトするため通常ここには戻らない。
        return new Promise<AccountContextResponse>(() => {})
      }
    }
    throw error
  }
}

// requireSystemAccount はシステムコンソール (/system) 用ガード。制御面 realm で
// 認証し、有効ロールに system_admin を持つアカウントだけを通す。どちらかを満たさない
// 場合はテナント管理コンソールへ送り返す (サーバー側 API も同じ条件を要求する)。
export async function requireSystemAccount(
  pathname: string,
  search: string,
): Promise<AccountContextResponse> {
  const account = await requirePortalAccount('admin', pathname, search)
  if (!isControlPlaneAccount(account)) {
    throw redirect({ to: '/admin' })
  }
  return account
}
