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
  rate_limited: 'rateLimited',
} as const

// stable な backend error code だけを UI の辞書へ対応付ける。未登録 code の本文は
// backend が英語で返す契約を維持し、勝手に翻訳・置換しない。retryAfterSeconds は
// rate_limited のときだけ、完結した1文として末尾に足す (ADR-157)。
export function localizedErrorMessage(
  locale: Locale,
  code: string | undefined,
  fallback: string,
  retryAfterSeconds?: number,
): string {
  if (!code || !(code in knownErrorKeys)) return fallback
  const message = commonDictionary[locale][knownErrorKeys[code as keyof typeof knownErrorKeys]]
  if (code === 'rate_limited' && typeof retryAfterSeconds === 'number') {
    return `${message} ${formatRetryAfter(locale, retryAfterSeconds)}`
  }
  return message
}

function formatRetryAfter(locale: Locale, seconds: number): string {
  return locale === 'ja'
    ? `${seconds}秒後にもう一度お試しください。`
    : `Try again in ${seconds} seconds.`
}
