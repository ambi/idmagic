// SQL 文字列をデータベースへ直接渡す呼び出しを見つける。
// 静的な文は sqlc の入力にする。sqlc で表せない呼び出し（構造が実行時まで決まる文、
// DDL の適用、sqlc の生成コードから受け取った SQL の中継、接続の設定）だけが、
// 直前の行の `//sql:raw <理由>` によって例外になる。

export type RawSqlProblem = 'unmarked' | 'missing-reason'

export type RawSqlCall = {
  line: number
  problem: RawSqlProblem
}

// pgx の問い合わせメソッドのうち、第 1 引数に context を取る呼び出しだけを拾う。
// `url.Values` を返す引数のない `Query()` を除くための条件である。
const QUERY_CALL =
  /\.(?:Exec|Query|QueryRow|SendBatch|CopyFrom)\(\s*(?:[A-Za-z_]*[cC]tx\b|context\.\w+\(\))/g
const GENERATED_HEADER = /^\/\/ Code generated .* DO NOT EDIT\.$/m
const MARKER = /^\s*\/\/sql:raw(?:\s+(\S.*))?$/

export function isRawSqlTarget(path: string): boolean {
  return path.startsWith('backend/') && path.endsWith('.go') && !path.endsWith('_test.go')
}

export function rawSqlCalls(source: string): RawSqlCall[] {
  if (GENERATED_HEADER.test(source)) return []
  const lines = source.split('\n')
  const calls: RawSqlCall[] = []
  for (const match of source.matchAll(QUERY_CALL)) {
    const line = source.slice(0, match.index).split('\n').length
    const marker = MARKER.exec(lines[line - 2] ?? '')
    if (!marker) calls.push({ line, problem: 'unmarked' })
    else if (!marker[1]) calls.push({ line, problem: 'missing-reason' })
  }
  return calls
}
