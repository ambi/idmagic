// sqlc のクエリと同じディレクトリの Go の関数を照合し、読んだ後の書き込みが守られていない形を見つける。
//
// - row-lock-gap: `SELECT ... FOR UPDATE` は行がないと何もロックしない。行がない経路で処理を続けて
//   同じテーブルへ挿入すると、同時の要求が互いの結果を上書きする。
// - unguarded-write: ロックなしで読んだテーブルへ無条件に書くと、読んだ後の変更を書き戻す。
// - lock-outside-tx: トランザクションの外の `FOR UPDATE` は、文の終わりでロックを失う。
//
// 関数の中だけを見るテキストの照合であり、呼び出し元のロックは直前の行の
// `//sql:concurrency-ok <理由>` で宣言する。

export type SqlKind = 'select' | 'insert' | 'update'

export type SqlQuery = {
  kind: SqlKind
  table: string
  // SELECT ... FOR UPDATE。ほかの文の副問い合わせのロックは含めない。
  locksRow: boolean
  // 既存の行を上書きしない INSERT か、更新する列を WHERE でも確かめる UPDATE。
  guarded: boolean
  // 既存の行があっても失敗しない INSERT。行を先に作る形に使う。
  ensuresRow: boolean
}

export type SqlQueries = {
  byName: Map<string, SqlQuery>
  advisoryLocks: Set<string>
}

export type ConcurrencyProblem =
  | 'row-lock-gap'
  | 'unguarded-write'
  | 'lock-outside-tx'
  | 'missing-reason'

export type ConcurrencyFinding = {
  line: number
  query: string
  table: string
  problem: ConcurrencyProblem
}

const NAMED_QUERY = /^-- name:\s*(\w+)\s+:\w+[^\n]*\n([\s\S]*?)(?=^-- name:|(?![\s\S]))/gm
const GENERATED_HEADER = /^\/\/ Code generated .* DO NOT EDIT\.$/m
const MARKER = /^\s*\/\/sql:concurrency-ok(?:\s+(\S.*))?$/
const FUNCTION_NAME = /^func (?:\([^)]*\) )?(\w+)/

export function isConcurrencyGoTarget(path: string): boolean {
  return path.startsWith('backend/') && path.endsWith('.go') && !path.endsWith('_test.go')
}

export function isConcurrencySqlTarget(path: string): boolean {
  return path.startsWith('backend/') && path.endsWith('.sql')
}

function updateIsGuarded(body: string): boolean {
  const parts = /\bSET\b([\s\S]*?)\bWHERE\b([\s\S]*)$/i.exec(body)
  if (!parts?.[1] || !parts[2]) return false
  const assigned = [...parts[1].matchAll(/(?:^|,)\s*(\w+)\s*=/g)].map((match) => match[1])
  const where = parts[2]
  return assigned.some((column) => new RegExp(`\\b${column}\\b`, 'i').test(where))
}

export function parseSqlQueries(sql: string): SqlQueries {
  const queries: SqlQueries = { byName: new Map(), advisoryLocks: new Set() }
  for (const [, name, rawBody] of sql.matchAll(NAMED_QUERY)) {
    if (!name || rawBody === undefined) continue
    const body = rawBody.replace(/--[^\n]*/g, '').trim()
    if (/\bpg_(?:try_)?advisory_xact_lock/i.test(body)) {
      queries.advisoryLocks.add(name)
      continue
    }
    const from = /^(?:WITH\b[\s\S]*?\)\s*)?SELECT\b[\s\S]*?\bFROM\s+(\w+)/i.exec(body)
    if (/^SELECT\b/i.test(body) && from?.[1]) {
      const locksRow = /\bFOR\s+(?:NO\s+KEY\s+)?UPDATE\b/i.test(body)
      queries.byName.set(name, {
        kind: 'select',
        table: from[1],
        locksRow,
        guarded: false,
        ensuresRow: false,
      })
      continue
    }
    const into = /^INSERT\s+INTO\s+(\w+)/i.exec(body)
    if (into?.[1]) {
      const conflict = /\bON\s+CONFLICT\b/i.test(body)
      const overwrites = /\bDO\s+UPDATE\b/i.test(body)
      queries.byName.set(name, {
        kind: 'insert',
        table: into[1],
        locksRow: false,
        guarded: !overwrites,
        ensuresRow: conflict,
      })
      continue
    }
    const update = /^UPDATE\s+(\w+)/i.exec(body)
    if (update?.[1]) {
      queries.byName.set(name, {
        kind: 'update',
        table: update[1],
        locksRow: false,
        guarded: updateIsGuarded(body),
        ensuresRow: false,
      })
    }
  }
  return queries
}

export function mergeSqlQueries(into: SqlQueries, from: SqlQueries): void {
  for (const [name, query] of from.byName) into.byName.set(name, query)
  for (const name of from.advisoryLocks) into.advisoryLocks.add(name)
}

function matchingBrace(source: string, open: number): number {
  let depth = 0
  for (let i = open; i < source.length; i++) {
    if (source[i] === '{') depth++
    if (source[i] === '}' && --depth === 0) return i
  }
  return source.length
}

// 行がない経路で、呼び出しの後の処理へ進むかを判定する。ErrNoRows に触れなければ、
// エラーとして返すものとみなす。
function continuesWithoutRow(afterCall: string): boolean {
  const mention = /ErrNoRows/.exec(afterCall)
  if (!mention) return false
  const lineStart = afterCall.lastIndexOf('\n', mention.index) + 1
  const lineEnd = afterCall.indexOf('\n', mention.index)
  const line = afterCall.slice(lineStart, lineEnd === -1 ? undefined : lineEnd)
  if (/!errors\.Is\(/.test(line)) return true
  if (/^\s*case\b/.test(line)) {
    const rest = afterCall.slice(lineEnd + 1)
    const next = /^\s*(?:case\b|default:)/m.exec(rest)
    return !/\breturn\b/.test(next ? rest.slice(0, next.index) : rest)
  }
  const open = afterCall.indexOf('{', mention.index)
  if (open === -1) return false
  return !/\breturn\b/.test(afterCall.slice(open, matchingBrace(afterCall, open)))
}

type GoFunction = { name: string; start: number; body: string }
type QueryCall = SqlQuery & { name: string; at: number }

function goFunctions(source: string): GoFunction[] {
  const starts = [...source.matchAll(/^func [^\n]*/gm)]
  return starts.map((match, index) => {
    const start = match.index
    const body = source.slice(start, starts[index + 1]?.index ?? source.length)
    return { name: FUNCTION_NAME.exec(match[0])?.[1] ?? '', start, body }
  })
}

function queryCalls(queries: SqlQueries, body: string): QueryCall[] {
  const calls: QueryCall[] = []
  for (const match of body.matchAll(/\.(\w+)\(/g)) {
    const name = match[1]
    const query = name ? queries.byName.get(name) : undefined
    if (name && query) calls.push({ ...query, name, at: match.index })
  }
  return calls
}

// ロックのクエリを呼ぶ、同じファイルの補助関数の名前を集める。呼び出しは一段だけ追う。
function lockHelpers(queries: SqlQueries, functions: GoFunction[]): Set<string> {
  const helpers = new Set<string>()
  for (const fn of functions) {
    const locks =
      queryCalls(queries, fn.body).some((call) => call.locksRow) ||
      [...queries.advisoryLocks].some((name) => fn.body.includes(`.${name}(`))
    if (locks) helpers.add(fn.name)
  }
  return helpers
}

function lockedBefore(
  queries: SqlQueries,
  helpers: Set<string>,
  fn: GoFunction,
  at: number,
): boolean {
  const before = fn.body.slice(0, at)
  const body = before.slice(before.indexOf('\n') + 1)
  if ([...queries.advisoryLocks].some((name) => body.includes(`.${name}(`))) return true
  if (queryCalls(queries, body).some((call) => call.locksRow)) return true
  return [...helpers].some((name) =>
    new RegExp(`(?:^|[^\\w.])(?:\\w+\\.)?${name}\\(`, 'm').test(body),
  )
}

function outsideTransaction(body: string): boolean {
  const pool = /\bNew\([\w.]*(?:Pool|pool|DB|db)\)/.test(body)
  return pool && !/\bBegin(?:Func|Tx)?\(|\bWithTx\(|\bNew\(tx\)/.test(body)
}

export function concurrencyFindings(queries: SqlQueries, source: string): ConcurrencyFinding[] {
  if (GENERATED_HEADER.test(source)) return []
  const lines = source.split('\n')
  const functions = goFunctions(source)
  const helpers = lockHelpers(queries, functions)
  const findings: ConcurrencyFinding[] = []
  const report = (fn: GoFunction, call: QueryCall, problem: ConcurrencyProblem): void => {
    const line = source.slice(0, fn.start + call.at).split('\n').length
    const marker = MARKER.exec(lines[line - 2] ?? '')
    if (marker?.[1]) return
    findings.push({
      line,
      query: call.name,
      table: call.table,
      problem: marker ? 'missing-reason' : problem,
    })
  }
  for (const fn of functions) {
    const calls = queryCalls(queries, fn.body)
    const reported = new Set<number>()
    for (const read of calls.filter((call) => call.kind === 'select')) {
      const writes = calls.filter(
        (call) => call.at > read.at && call.kind !== 'select' && call.table === read.table,
      )
      if (read.locksRow) {
        if (outsideTransaction(fn.body)) report(fn, read, 'lock-outside-tx')
        const insert = writes.find((call) => call.kind === 'insert')
        if (!insert || !continuesWithoutRow(fn.body.slice(read.at))) continue
        const ensured = calls.some(
          (call) => call.at < read.at && call.ensuresRow && call.table === read.table,
        )
        if (!ensured && !lockedBefore(queries, helpers, fn, read.at))
          report(fn, read, 'row-lock-gap')
        continue
      }
      for (const write of writes) {
        if (write.guarded || reported.has(write.at)) continue
        if (lockedBefore(queries, helpers, fn, write.at)) continue
        reported.add(write.at)
        report(fn, write, 'unguarded-write')
      }
    }
  }
  return findings
}
