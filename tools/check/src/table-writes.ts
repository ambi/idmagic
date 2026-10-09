/**
 * sqlc のクエリ入力が明示的に書き込むテーブルを求め、データベース設計の所有モジュールと照合する。
 *
 * SQL は `sqlc parse` が返す PostgreSQL の構文木から読む。コメント、文字列、引用識別子、
 * スキーマ修飾、別名の区別は構文解析器に任せ、単語の照合では書き込み先を決めない。
 * 構文木の JSON はノードの型名を持たないので、文の種類はその型だけが持つフィールドの組で見分ける。
 * sqlc が構文木へ変換しない文（MERGE など）は空のオブジェクトになるため、書き込みなしとは扱わず診断にする。
 *
 * 判定するのは SQL に書いた直接の対象だけであり、外部キーの cascade、トリガー、関数の呼び出し、
 * `//sql:raw` の動的な SQL による書き込みは対象外である。
 */

import { SHARED_INFRASTRUCTURE_OWNER } from './schema-tables.ts'

export type SqlStatement = {
  text: string
  tree: unknown
}

export type WriteOperation = 'insert' | 'update' | 'delete' | 'truncate'

export type TableWrite = {
  query: string
  operation: WriteOperation
  table: string
  /** 書き込む列の昇順。行を消す操作は列を限定しないので `*` とする。 */
  columns: string[]
}

/** クエリ入力を置いたパッケージの分類。テーブルの所有者と照合する名前を決める。 */
export type QueryWriter =
  | { kind: 'module'; name: string }
  | { kind: 'shared' }
  | { kind: 'composition' }

export type QueryInput = {
  path: string
  writer: QueryWriter
  writes: readonly TableWrite[]
}

export type TableWriteViolation = {
  id: string
  debtId: string
  kind: 'table-write'
  sourceModule: string
  targetModule: string
  path: string
  message: string
}

const ALL_COLUMNS = '*'

/** `sqlc parse` で SQL を文ごとの構文木にする。構文の誤りは例外にする。 */
export async function parseSql(sql: string): Promise<SqlStatement[]> {
  return (await parseLocatedSql(sql)).map(({ text, tree }) => ({ text, tree }))
}

/**
 * 複数のクエリ入力を一度の `sqlc parse` で読み、文をファイルへ振り分ける。
 * 起動が一回ごとに約 0.5 秒かかるため、ファイルごとには呼ばない。
 * 連結した入力が解析できないときだけ、ファイルごとに読み直して誤りのあるファイルを示す。
 */
export async function parseSqlFiles(
  files: readonly { path: string; sql: string }[],
): Promise<Map<string, SqlStatement[]>> {
  const separator = '\n;\n'
  const ranges: { path: string; start: number; end: number }[] = []
  let offset = 0
  for (const file of files) {
    const length = Buffer.byteLength(file.sql)
    ranges.push({ path: file.path, start: offset, end: offset + length })
    offset += length + Buffer.byteLength(separator)
  }
  let statements: LocatedStatement[]
  try {
    statements = await parseLocatedSql(files.map((file) => file.sql).join(separator))
  } catch (error) {
    for (const file of files) {
      await parseLocatedSql(file.sql).catch((fileError: Error) => {
        throw new Error(`${file.path}: ${fileError.message}`)
      })
    }
    throw error
  }
  const byPath = new Map(files.map((file): [string, SqlStatement[]] => [file.path, []]))
  for (const statement of statements) {
    // 文の開始位置は直前の `;` の直後なので区切りの中に入りうる。終了位置で所属を決める。
    const range = ranges.find((r) => statement.end > r.start && statement.end <= r.end)
    if (!range) continue
    byPath.get(range.path)!.push({ text: statement.text, tree: statement.tree })
  }
  return byPath
}

type LocatedStatement = SqlStatement & { end: number }

async function parseLocatedSql(sql: string): Promise<LocatedStatement[]> {
  const child = Bun.spawn(['sqlc', 'parse', '--dialect', 'postgresql'], {
    stdin: new TextEncoder().encode(sql),
    stdout: 'pipe',
    stderr: 'pipe',
  })
  const [stdout, stderr, exitCode] = await Promise.all([
    new Response(child.stdout).text(),
    new Response(child.stderr).text(),
    child.exited,
  ])
  if (exitCode !== 0) throw new Error(`sqlc parse: ${stderr.trim()}`)
  const bytes = Buffer.from(sql)
  // 出力は整形した JSON の連続であり、文の境界だけが行頭の `{` から始まる。
  return stdout
    .split(/\n(?=\{)/)
    .filter((chunk) => chunk.trim() !== '')
    .map((chunk) => {
      const raw = JSON.parse(chunk) as { Stmt: unknown; StmtLocation: number; StmtLen: number }
      const end = raw.StmtLen === 0 ? bytes.length : raw.StmtLocation + raw.StmtLen
      return { text: bytes.subarray(raw.StmtLocation, end).toString(), tree: raw.Stmt, end }
    })
}

/** スキーマの `CREATE TABLE` から、テーブルごとの列を宣言順に読む。 */
export function schemaColumns(statements: readonly SqlStatement[]): Map<string, string[]> {
  const tables = new Map<string, string[]>()
  for (const { tree } of statements) {
    if (!isRecord(tree) || !isRecord(tree.Name) || !Array.isArray(tree.Cols)) continue
    if (!('ReferTable' in tree && 'Inherits' in tree)) continue
    const name = tree.Name.Name
    if (typeof name !== 'string') continue
    tables.set(
      name,
      tree.Cols.flatMap((column) =>
        isRecord(column) && typeof column.Colname === 'string' ? [column.Colname] : [],
      ),
    )
  }
  return tables
}

/** 文ごとの書き込み先と、抽出できなかった理由を返す。 */
export function extractTableWrites(
  statements: readonly SqlStatement[],
  columnsByTable: ReadonlyMap<string, readonly string[]>,
): { writes: TableWrite[]; diagnostics: string[] } {
  const writes: TableWrite[] = []
  const diagnostics: string[] = []
  statements.forEach((statement, index) => {
    const query = statement.text.match(/--\s*name:\s*(\S+)/)?.[1] ?? `statement ${index + 1}`
    const visit = (node: unknown): void => {
      if (Array.isArray(node)) {
        node.forEach(visit)
        return
      }
      if (!isRecord(node)) return
      for (const target of writeTargets(node)) {
        const table = resolveTable(target.relation, columnsByTable)
        if (typeof table !== 'string') {
          diagnostics.push(`${query}: ${table.problem}`)
          continue
        }
        const columns =
          target.columns === 'all-declared'
            ? [...(columnsByTable.get(table) ?? [])]
            : target.columns
        writes.push({
          query,
          operation: target.operation,
          table,
          columns: [...new Set(columns)].sort(),
        })
      }
      if (isRecord(node.Ctequery) && Object.keys(node.Ctequery).length === 0) {
        diagnostics.push(
          `${query}: the SQL parser does not expose a CTE statement; table writes cannot be extracted`,
        )
      }
      Object.values(node).forEach(visit)
    }
    if (isRecord(statement.tree) && Object.keys(statement.tree).length === 0) {
      diagnostics.push(
        `${query}: the SQL parser does not expose this statement; table writes cannot be extracted`,
      )
      return
    }
    visit(statement.tree)
  })
  return {
    writes: writes.sort((left, right) =>
      `${left.query}\0${left.table}\0${left.operation}`.localeCompare(
        `${right.query}\0${right.table}\0${right.operation}`,
      ),
    ),
    diagnostics,
  }
}

/** 書き込みを所有者と照合する。所有者のないテーブルは違反ではなく診断にする。 */
export function tableWriteViolations(
  inputs: readonly QueryInput[],
  owners: ReadonlyMap<string, string>,
): { violations: TableWriteViolation[]; diagnostics: string[] } {
  const violations: TableWriteViolation[] = []
  const diagnostics: string[] = []
  for (const input of inputs) {
    const writerName = writerDisplayName(input.writer)
    const ownedName =
      input.writer.kind === 'module'
        ? input.writer.name
        : input.writer.kind === 'shared'
          ? SHARED_INFRASTRUCTURE_OWNER
          : undefined
    for (const write of input.writes) {
      const owner = owners.get(write.table)
      if (!owner) {
        diagnostics.push(
          `${input.path}: ${write.query} writes ${write.table}, which has no owning module`,
        )
        continue
      }
      if (owner === ownedName) continue
      violations.push({
        id: [
          'table-write',
          writerName,
          input.path,
          write.query,
          write.operation,
          write.table,
          write.columns.join(','),
        ].join(':'),
        debtId: `table-write:${writerName}->${owner}`,
        kind: 'table-write',
        sourceModule: writerName,
        targetModule: owner,
        path: `${input.path}#${write.query}`,
        message: `${writerName} ${write.operation}s ${write.table} owned by ${owner} in ${input.path} ${write.query}`,
      })
    }
  }
  return { violations, diagnostics }
}

function writerDisplayName(writer: QueryWriter): string {
  if (writer.kind === 'module') return writer.name
  return writer.kind === 'shared' ? 'shared' : 'System'
}

type WriteTarget = {
  operation: WriteOperation
  relation: unknown
  columns: string[] | 'all-declared'
}

/** 書き込み文のノードなら、その対象を返す。フィールドの組は sqlc の ast パッケージの型に対応する。 */
function writeTargets(node: Record<string, unknown>): WriteTarget[] {
  if ('Relation' in node && 'Cols' in node && 'SelectStmt' in node && 'OnConflictClause' in node) {
    const declared = names(listItems(node.Cols))
    const updated = isRecord(node.OnConflictClause)
      ? names(listItems(node.OnConflictClause.TargetList))
      : []
    return [
      {
        operation: 'insert',
        relation: node.Relation,
        columns: declared.length === 0 ? 'all-declared' : [...declared, ...updated],
      },
    ]
  }
  if ('Relations' in node && 'Behavior' in node && 'RestartSeqs' in node) {
    return listItems(node.Relations).map((relation) => ({
      operation: 'truncate',
      relation,
      columns: [ALL_COLUMNS],
    }))
  }
  if ('Relations' in node && 'UsingClause' in node) {
    return listItems(node.Relations).map((relation) => ({
      operation: 'delete',
      relation,
      columns: [ALL_COLUMNS],
    }))
  }
  if ('Relations' in node && 'TargetList' in node && 'FromClause' in node) {
    const columns = names(listItems(node.TargetList))
    return listItems(node.Relations).map((relation) => ({ operation: 'update', relation, columns }))
  }
  return []
}

function resolveTable(
  relation: unknown,
  columnsByTable: ReadonlyMap<string, readonly string[]>,
): string | { problem: string } {
  if (!isRecord(relation) || typeof relation.Relname !== 'string') {
    return { problem: 'a write target is not a plain table reference' }
  }
  const schema = typeof relation.Schemaname === 'string' ? relation.Schemaname : undefined
  if (schema !== undefined && schema !== 'public') {
    return { problem: `write target ${schema}.${relation.Relname} is outside the public schema` }
  }
  if (!columnsByTable.has(relation.Relname)) {
    return { problem: `write target ${relation.Relname} is not a table of the schema` }
  }
  return relation.Relname
}

function listItems(list: unknown): unknown[] {
  if (Array.isArray(list)) return list
  return isRecord(list) && Array.isArray(list.Items) ? list.Items : []
}

function names(items: readonly unknown[]): string[] {
  return items.flatMap((item) =>
    isRecord(item) && typeof item.Name === 'string' ? [item.Name] : [],
  )
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
