/**
 * sqlc のクエリが、データベースの時計を判定に使う箇所を見つける。
 *
 * ストレージのアダプターは、期限や順序の判定に呼び出し側から受け取った時刻を使う
 * (docs/design/application/design-guidelines.md の作用の境界)。SQL の `now()` は、
 * `UPDATE … SET` と `ON CONFLICT DO UPDATE SET` の代入の値、`INSERT … VALUES` の項目として
 * 記録する時刻にだけ許す。代入の値が `CASE` なら、結果の位置にだけ許す。
 * それ以外の位置 (WHERE、ORDER BY、比較、算術、SELECT の列、CASE の条件) は判定とみなす。
 *
 * 構文木は `sqlc parse` の JSON で、ノードの型名を持たないので、型だけが持つフィールドの組で見分ける。
 * 列の既定値のように、クエリに現れない時刻は対象外である。
 */

import type { SqlStatement } from './table-writes.ts'

export type ClockReading = {
  query: string
  expression: string
}

/** 時計を読む関数。schema 修飾 (`pg_catalog.now()`) も同じ関数として扱う。 */
const CLOCK_FUNCTIONS = new Set([
  'now',
  'clock_timestamp',
  'statement_timestamp',
  'transaction_timestamp',
  'timeofday',
])

/**
 * SQLValueFunction の Op のうち時刻を返すもの。sqlc の構文木では CURRENT_DATE が 1、
 * LOCALTIMESTAMP(n) が 9 で、10 以降は CURRENT_ROLE や CURRENT_USER である。
 */
const SQL_VALUE_FUNCTION_TIME_OPS = new Map([
  [1, 'CURRENT_DATE'],
  [2, 'CURRENT_TIME'],
  [3, 'CURRENT_TIME'],
  [4, 'CURRENT_TIMESTAMP'],
  [5, 'CURRENT_TIMESTAMP'],
  [6, 'LOCALTIME'],
  [7, 'LOCALTIME'],
  [8, 'LOCALTIMESTAMP'],
  [9, 'LOCALTIMESTAMP'],
])

/** `'now'::timestamptz` のように、型変換で現在の時刻になる文字列。 */
const CLOCK_LITERALS = new Set(['now', 'today', 'tomorrow', 'yesterday'])

/** 文ごとに、記録の位置の外で時計を読む式を返す。 */
export function clockReadings(statements: readonly SqlStatement[]): ClockReading[] {
  const readings: ClockReading[] = []
  statements.forEach((statement, index) => {
    const query = statement.text.match(/--\s*name:\s*(\S+)/)?.[1] ?? `statement ${index + 1}`
    const visit = (node: unknown, recorded: boolean): void => {
      if (Array.isArray(node)) {
        for (const item of node) visit(item, false)
        return
      }
      if (!isRecord(node)) return
      const clock = clockExpression(node)
      if (clock) {
        if (!recorded) readings.push({ query, expression: clock })
        return
      }
      if (recorded && isCaseExpression(node)) {
        visitCase(node)
        return
      }
      for (const [key, child] of Object.entries(node)) {
        if (key === 'TargetList' && assignsColumns(node)) visitAssignments(child)
        else if (key === 'ValuesLists') visitValues(child)
        else if (key === 'Ctequery' && isOpaque(child)) opaque = true
        else visit(child, false)
      }
    }
    // sqlc が構文木へ変換しない文や CTE は空のオブジェクトになる。位置を確かめられないので、
    // 本文が時計を読む式を含めば判定として扱い、見逃さない。
    let opaque = isOpaque(statement.tree)
    const visitAssignments = (list: unknown): void => {
      for (const target of items(list)) {
        if (!isRecord(target)) continue
        for (const [key, child] of Object.entries(target)) visit(child, key === 'Val')
      }
    }
    const visitValues = (list: unknown): void => {
      for (const row of items(list)) {
        for (const value of items(row)) visit(value, true)
      }
    }
    const visitCase = (node: Record<string, unknown>): void => {
      for (const [key, child] of Object.entries(node)) {
        if (key === 'Defresult') visit(child, true)
        else if (key === 'Args') {
          for (const when of items(child)) {
            if (!isRecord(when)) continue
            for (const [whenKey, part] of Object.entries(when)) visit(part, whenKey === 'Result')
          }
        } else visit(child, false)
      }
    }
    visit(statement.tree, false)
    if (opaque && CLOCK_TEXT.test(statement.text)) {
      readings.push({ query, expression: 'a clock read the SQL parser does not expose' })
    }
  })
  return readings
}

/** 構文木から位置を確かめられない文で、時計を読む式を本文から探す。コメントも含めて保守的に一致させる。 */
const CLOCK_TEXT =
  /\b(?:now|clock_timestamp|statement_timestamp|transaction_timestamp|timeofday)\s*\(|\b(?:current_date|current_time|current_timestamp|localtime|localtimestamp)\b|'(?:now|today|tomorrow|yesterday)'/i

function isOpaque(node: unknown): boolean {
  return isRecord(node) && Object.keys(node).length === 0
}

function clockExpression(node: Record<string, unknown>): string | undefined {
  if ('Funcname' in node) {
    const names = items(node.Funcname).flatMap((part) =>
      isRecord(part) && typeof part.Str === 'string' ? [part.Str] : [],
    )
    const name = names.at(-1)
    if (name && CLOCK_FUNCTIONS.has(name.toLowerCase())) return `${names.join('.')}()`
    return undefined
  }
  if (isSqlValueFunction(node) && typeof node.Op === 'number') {
    return SQL_VALUE_FUNCTION_TIME_OPS.get(node.Op)
  }
  if ('TypeName' in node && isRecord(node.Arg) && isRecord(node.Arg.Val)) {
    const literal = node.Arg.Val.Str
    if (typeof literal === 'string' && CLOCK_LITERALS.has(literal.trim().toLowerCase())) {
      return `'${literal}'::`
    }
  }
  return undefined
}

/** UPDATE の文と ON CONFLICT DO UPDATE の節だけが、TargetList を列への代入として持つ。 */
function assignsColumns(node: Record<string, unknown>): boolean {
  const isUpdate = 'Relations' in node && 'WhereClause' in node && 'FromClause' in node
  const isOnConflict = 'Action' in node && 'Infer' in node
  return isUpdate || isOnConflict
}

function isCaseExpression(node: Record<string, unknown>): boolean {
  return 'Casetype' in node && 'Args' in node && 'Defresult' in node
}

function isSqlValueFunction(node: Record<string, unknown>): boolean {
  const keys = Object.keys(node).sort().join(',')
  return keys === 'Location,Op,Type,Typmod,Xpr'
}

function items(list: unknown): unknown[] {
  if (Array.isArray(list)) return list
  if (isRecord(list) && Array.isArray(list.Items)) return list.Items
  return []
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
