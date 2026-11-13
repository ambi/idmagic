// ポータルが管理 API とアカウント API を呼ぶときは、共通の呼び出し処理を通してベアラー
// トークンを付ける (docs/design/application/frontend.md の API の呼び出し)。個別に fetch を
// 書くとセッションの Cookie だけで呼ばれ、サーバーが Cookie も受けるので動いてしまい、
// ステップアップ認証の根拠が操作ごとに分かれる。Cookie で呼ぶべき操作は
// `// session-cookie: <理由>` で明示する。

export type PortalApiFetchProblem = 'unmarked' | 'missing-reason'

const PORTAL_API = /\/api\/(admin|account)\/|\/api\/auth\/(account|change-password)\b/
const MARKER = /^\s*\/\/\s*session-cookie:(.*)$/
// fetch の URL は同じ行か、続く数行に書かれる。
const URL_WINDOW_LINES = 3
// 印は呼び出しを含む関数の宣言の直前か、関数の中に置く。前の関数の終わりより先は見ない。
const MARKER_LOOKBACK_LINES = 6

export function isPortalApiCallTarget(path: string): boolean {
  return (
    path.startsWith('frontend/src/') &&
    /\.(ts|tsx)$/.test(path) &&
    !/\.test\.(ts|tsx)$/.test(path) &&
    path !== 'frontend/src/api/core.ts'
  )
}

export function unmarkedPortalApiFetches(
  source: string,
): Array<{ line: number; problem: PortalApiFetchProblem }> {
  const lines = source.split('\n')
  const problems: Array<{ line: number; problem: PortalApiFetchProblem }> = []
  lines.forEach((text, index) => {
    if (!/\bfetch\(/.test(text)) return
    const call = lines.slice(index, index + URL_WINDOW_LINES).join('\n')
    if (!PORTAL_API.test(call)) return
    const marker = findMarker(lines, index)
    if (marker === undefined) problems.push({ line: index + 1, problem: 'unmarked' })
    else if (marker.trim() === '') problems.push({ line: index + 1, problem: 'missing-reason' })
  })
  return problems
}

function findMarker(lines: string[], callIndex: number): string | undefined {
  const earliest = Math.max(0, callIndex - MARKER_LOOKBACK_LINES)
  for (const line of lines.slice(earliest, callIndex).reverse()) {
    const marker = line.match(MARKER)
    if (marker) return marker[1] ?? ''
    if (/^}/.test(line)) return undefined
  }
  return undefined
}
