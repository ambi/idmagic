import { redirect } from '@tanstack/react-router'
import { request, StepUpRequiredError, tenantBasePath, UnauthenticatedError } from '../api/core'
import {
  ensureLoggedIn,
  markPortalAuthenticated,
  type PortalAudience,
  reauthorize,
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
  await ensureControlPlaneAssurance(pathname, search)
  return account
}

// ensureControlPlaneAssurance は、システムの経路の読み出しが求める MFA の認証を満たしているかを、
// 入場の時点で一度だけ確かめる (REQ-SYSTEM-033)。満たさなければ、利用者の操作がないので
// ポップアップを開けず、応答の条件を付けて画面全体で再認可する（戻らない）。ほかの失敗は
// 各画面の読み込みに任せる。認可の正はサーバー側のゲートにあり、ここは早い案内だけを行う。
export async function ensureControlPlaneAssurance(pathname: string, search: string): Promise<void> {
  try {
    await request('/api/admin/v1/tenants/default')
  } catch (error) {
    if (error instanceof StepUpRequiredError) {
      await reauthorize(error.challenge, null, `${tenantBasePath()}${pathname}${search}`)
      return new Promise<void>(() => {})
    }
  }
}
