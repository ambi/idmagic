import { commonDictionary } from './common.i18n'
import type { Locale } from './locale'

const knownErrorKeys = {
  invalid_credentials: 'invalidCredentials',
  password_policy: 'passwordPolicy',
  password_reuse: 'passwordReuse',
  invalid_request: 'invalidRequest',
  access_denied: 'accessDenied',
  csrf_token: 'csrfToken',
  session_not_found: 'sessionNotFound',
} as const

// stable な backend error code だけを UI の辞書へ対応付ける。未登録 code の本文は
// backend が英語で返す契約を維持し、勝手に翻訳・置換しない。
export function localizedErrorMessage(
  locale: Locale,
  code: string | undefined,
  fallback: string,
): string {
  if (!code || !(code in knownErrorKeys)) return fallback
  return commonDictionary[locale][knownErrorKeys[code as keyof typeof knownErrorKeys]]
}
