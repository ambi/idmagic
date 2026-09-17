/**
 * データベース設計のテーブル一覧が、物理スキーマの宣言と一致することを確かめる。
 *
 * 照合するのは SQL から機械的に決まるもの（テーブル名、テーブル種別、`tenant_id` 列の
 * 区分）に限る。役割と所有 Context は判断を書いた列であり、SQL からは決まらない。
 */

export type TenantIdPlacement = 'primary-key' | 'primary-key-part' | 'column' | 'absent'

/** `CREATE TABLE` 一つから読み取った事実。 */
export interface DeclaredTable {
  name: string
  unlogged: boolean
  tenantId: TenantIdPlacement
}

/** テーブル一覧の 1 行。セルの値は照合前の文字列のまま保持する。 */
export interface DescribedTable {
  name: string
  line: number
  kind: string
  tenantId: string
}

export interface Finding {
  message: string
}

const LOGGED_KIND = '`LOGGED`'
const UNLOGGED_KIND = '`UNLOGGED`'
const KIND_HEADER = 'テーブル種別'
const TENANT_ID_HEADER = '`tenant_id` 列'

const TENANT_ID_LABELS: Record<TenantIdPlacement, string> = {
  'primary-key': '単独主キー',
  'primary-key-part': '複合主キーの一部',
  column: '非キー列',
  absent: 'なし',
}

const CREATE_TABLE = /^CREATE\s+(UNLOGGED\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-z0-9_]+)/

/** スキーマの `CREATE TABLE` を宣言順に読む。行コメントは読み飛ばす。 */
export function declaredTables(sql: string): DeclaredTable[] {
  const tables: DeclaredTable[] = []
  let current: { name: string; unlogged: boolean; body: string[] } | undefined
  for (const raw of sql.split('\n')) {
    const line = raw.replace(/--.*$/, '')
    const start = line.match(CREATE_TABLE)
    if (start?.[2]) {
      current = { name: start[2], unlogged: Boolean(start[1]), body: [] }
      continue
    }
    if (!current) continue
    if (/^\)/.test(line)) {
      tables.push({
        name: current.name,
        unlogged: current.unlogged,
        tenantId: tenantIdPlacement(current.body),
      })
      current = undefined
      continue
    }
    current.body.push(line)
  }
  return tables
}

function tenantIdPlacement(body: readonly string[]): TenantIdPlacement {
  const column = body.find((line) => /^\s+tenant_id\s/.test(line))
  if (!column) return 'absent'
  if (/\bPRIMARY\s+KEY\b/.test(column)) return 'primary-key'
  const key = body.join(' ').match(/\bPRIMARY\s+KEY\s*\(([^)]*)\)/)
  if (!key?.[1]) return 'column'
  const columns = key[1].split(',').map((name) => name.trim())
  if (!columns.includes('tenant_id')) return 'column'
  return columns.length === 1 ? 'primary-key' : 'primary-key-part'
}

function cells(line: string): string[] {
  return line
    .trim()
    .replace(/^\|/, '')
    .replace(/\|$/, '')
    .split('|')
    .map((cell) => cell.trim())
}

/** 先頭の見出しが「テーブル」である Markdown 表から行を集める。 */
export function describedTables(markdown: string): DescribedTable[] {
  const rows: DescribedTable[] = []
  let columns: { kind: number; tenantId: number } | undefined
  let inTable = false
  markdown.split('\n').forEach((line, index) => {
    if (!line.trimStart().startsWith('|')) {
      inTable = false
      columns = undefined
      return
    }
    const row = cells(line)
    if (!inTable) {
      inTable = true
      columns =
        row[0] === 'テーブル'
          ? { kind: row.indexOf(KIND_HEADER), tenantId: row.indexOf(TENANT_ID_HEADER) }
          : undefined
      return
    }
    if (!columns || row.every((cell) => /^:?-+:?$/.test(cell))) return
    rows.push({
      name: (row[0] ?? '').replace(/`/g, ''),
      line: index + 1,
      kind: row[columns.kind] ?? '',
      tenantId: row[columns.tenantId] ?? '',
    })
  })
  return rows
}

/** 宣言と説明の食い違いを、欠落、重複と余剰と分類違いの順に返す。 */
export function compareTables(
  declared: readonly DeclaredTable[],
  described: readonly DescribedTable[],
): Finding[] {
  if (declared.length === 0) return [{ message: 'the schema declares no table' }]
  const byName = new Map(declared.map((table) => [table.name, table]))
  const describedNames = new Set(described.map((table) => table.name))
  const findings: Finding[] = declared
    .filter((table) => !describedNames.has(table.name))
    .map((table) => ({ message: `${table.name} is declared in the schema but not described` }))

  const seen = new Set<string>()
  for (const row of described) {
    const at = `line ${row.line}: ${row.name}`
    if (seen.has(row.name)) {
      findings.push({ message: `${at} is described more than once` })
      continue
    }
    seen.add(row.name)
    const table = byName.get(row.name)
    if (!table) {
      findings.push({ message: `${at} is described but not declared in the schema` })
      continue
    }
    const kind = table.unlogged ? UNLOGGED_KIND : LOGGED_KIND
    if (row.kind !== kind) {
      const actual = table.unlogged ? 'an UNLOGGED' : 'a logged'
      findings.push({ message: `${at} is ${actual} table but is described as ${row.kind}` })
    }
    const tenantId = TENANT_ID_LABELS[table.tenantId]
    if (row.tenantId !== tenantId) {
      findings.push({
        message: `${at} has tenant_id as ${tenantId} but is described as ${row.tenantId}`,
      })
    }
  }
  return findings
}
