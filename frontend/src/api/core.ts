import { commonDictionary } from '../lib/i18n/common.i18n'
import { getCurrentLocale } from '../lib/i18n/currentLocale'

export type APIError = {
  error?: string
  message?: string
  error_description?: string
  retry_after_seconds?: number
  type?: string
  title?: string
  status?: number
  detail?: string
  instance?: string
}

const PROBLEM_TYPE_PREFIX = 'urn:idmagic:error:'

export class AuthenticationAPIError extends Error {
  code?: string
  retryAfterSeconds?: number

  constructor(message: string, code?: string, retryAfterSeconds?: number) {
    super(message)
    this.name = 'AuthenticationAPIError'
    this.code = code
    this.retryAfterSeconds = retryAfterSeconds
  }
}

export class UnauthenticatedError extends AuthenticationAPIError {
  constructor(message: string, code?: string) {
    super(message, code)
    this.name = 'UnauthenticatedError'
  }
}

export function apiErrorCode(body: APIError): string | undefined {
  if (body.error) return body.error
  if (!body.type?.startsWith(PROBLEM_TYPE_PREFIX)) return undefined
  const code = body.type.slice(PROBLEM_TYPE_PREFIX.length)
  return code || undefined
}

export function apiErrorMessage(
  body: APIError,
  fallback = commonDictionary[getCurrentLocale()].networkError,
): string {
  return body.message ?? body.error_description ?? body.detail ?? body.title ?? fallback
}

export function authenticationAPIError(body: APIError, fallback?: string): AuthenticationAPIError {
  return new AuthenticationAPIError(
    apiErrorMessage(body, fallback),
    apiErrorCode(body),
    body.retry_after_seconds,
  )
}

export async function responseAPIError(
  response: Response,
  fallback?: string,
): Promise<AuthenticationAPIError> {
  const body = (await response.json().catch(() => ({}))) as APIError
  return authenticationAPIError(body, fallback)
}

// bearerTokenProvider は OIDC RP モジュール (api/oidc) が登録する access token 取得関数。
// core → oidc の循環 import を避けるため、依存方向を逆 (oidc が core に登録) にする。
let bearerTokenProvider: () => string | null = () => null

export function setBearerTokenProvider(provider: () => string | null) {
  bearerTokenProvider = provider
}

async function doFetch<T>(
  url: string,
  init?: RequestInit,
): Promise<{ body: T; response: Response }> {
  const token = bearerTokenProvider()
  const headers = token
    ? { ...(init?.headers ?? {}), Authorization: `Bearer ${token}` }
    : init?.headers
  const response = await fetch(tenantURL(url), {
    credentials: 'same-origin',
    cache: 'no-store',
    ...init,
    ...(headers ? { headers } : {}),
  })
  const body = (await response.json().catch(() => ({}))) as T & APIError
  if (!response.ok) {
    const message = apiErrorMessage(body)
    const code = apiErrorCode(body)
    if (response.status === 401) {
      throw new UnauthenticatedError(message, code)
    }
    throw new AuthenticationAPIError(message, code, body.retry_after_seconds)
  }
  return { body, response }
}

export async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const { body } = await doFetch<T>(url, init)
  return body
}

// Page ラップは cursor pagination (ADR-159) の応答: body は素のドメインデータのまま、
// 前後ページの有無・cursor は Link レスポンスヘッダにだけ出てくる。
export type Page<T> = {
  body: T
  previousCursor: string | null
  nextCursor: string | null
}

// requestPage は request() と同じ fetch/エラー処理を共有しつつ、Link ヘッダから
// rel="prev" / rel="next" の cursor query param を抜き出して返す (ADR-159)。
export async function requestPage<T>(url: string, init?: RequestInit): Promise<Page<T>> {
  const { body, response } = await doFetch<T>(url, init)
  const link = response.headers.get('Link')
  return {
    body,
    previousCursor: parseCursorForRel(link, 'prev'),
    nextCursor: parseCursorForRel(link, 'next'),
  }
}

function parseCursorForRel(link: string | null, rel: 'prev' | 'next'): string | null {
  if (!link) return null
  const match = link.match(new RegExp(`<([^>]+)>;\\s*rel="${rel}"`))
  if (!match) return null
  try {
    return new URL(match[1], window.location.origin).searchParams.get('cursor')
  } catch {
    return null
  }
}

export function adminRequest(csrfToken: string, method: string, body?: unknown): RequestInit {
  return {
    method,
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': csrfToken,
    },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  }
}

export function tenantBasePath(path = window.location.pathname): string {
  const match = path.match(/^\/realms\/([a-z0-9][a-z0-9-]{0,62})(?:\/|$)/)
  return match ? `/realms/${match[1]}` : ''
}

export function tenantLocalPath(): string {
  const base = tenantBasePath()
  const path = window.location.pathname.slice(base.length)
  return path === '' ? '/' : path
}

// TanStack Router はテナント基底を basepath として別管理するため、Link には
// tenantURL が生成した URL 自身から基底を除いたローカルパスを渡す。
export function tenantRouterPath(path: string): string {
  const base = tenantBasePath(path)
  const localPath = path.slice(base.length)
  return localPath === '' ? '/' : localPath
}

// ローカル開発でこの補完を必要とする既知の bare origin: Vite dev server (5173)、
// E2E 専用ルート (5174)、docker-compose の frontend (8080)。ユニットテストなどの
// 他 localhost ポートへこの補完を広げない。
const LOCAL_DEFAULT_TENANT_PORTS = new Set(['5173', '5174', '8080'])

export function tenantURL(path: string): string {
  const base = tenantBasePath()
  // バックエンドの bare route は fail-closed のまま、クライアント生成 URL のみ補完する。
  const localDefaultTenant =
    base === '' &&
    window.location.hostname === 'localhost' &&
    LOCAL_DEFAULT_TENANT_PORTS.has(window.location.port)
  return `${localDefaultTenant ? '/realms/default' : base}${path}`
}

// validReturnTo は login 後に戻ってよい同一オリジンの内部パスかを判定する。
// 管理 UI (/admin 配下) と WS-Federation passive エンドポイント (/wsfed) を許可する (wi-61)。
export function validReturnTo(returnTo: string): boolean {
  if (!returnTo.startsWith('/') || returnTo.includes('\\')) return false
  const parsed = new URL(returnTo, window.location.origin)
  if (parsed.origin !== window.location.origin) return false
  const adminRoot = tenantURL('/admin')
  const wsfedPath = tenantURL('/wsfed')
  return (
    parsed.pathname === adminRoot ||
    parsed.pathname.startsWith(`${adminRoot}/`) ||
    parsed.pathname === wsfedPath
  )
}

export function base64URL(value: Uint8Array) {
  let binary = ''
  for (const byte of value) {
    binary += String.fromCharCode(byte)
  }
  return btoa(binary).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '')
}
