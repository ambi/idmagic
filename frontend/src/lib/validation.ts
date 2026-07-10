/**
 * 再認証時間（秒）をバリデーションし、パースします。
 * @param value 入力文字列
 * @returns パースされた数値、またはエラーメッセージ
 */
export function validateReauthMaxAge(
  value: string,
): { isValid: true; parsed?: number } | { isValid: false; error: string } {
  const text = value.trim()
  if (text === '') {
    return { isValid: true, parsed: undefined }
  }
  const parsed = Number.parseInt(text, 10)
  if (Number.isNaN(parsed) || parsed < 1 || String(parsed) !== text) {
    return {
      isValid: false,
      error: '再認証を求めるまでの時間には 1 以上の秒数を入力してください。',
    }
  }
  return { isValid: true, parsed }
}

/**
 * ネットワークCIDRの改行区切りテキストを配列に変換します。
 * @param value 改行区切りテキスト
 * @returns トリムされた非空CIDRの配列
 */
export function parseNetworkCIDRs(value: string): string[] {
  return value
    .split('\n')
    .map((entry) => entry.trim())
    .filter((entry) => entry !== '')
}
