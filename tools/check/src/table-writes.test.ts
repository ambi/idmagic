import { describe, expect, it } from 'bun:test'
import {
  extractTableWrites,
  parseSql,
  parseSqlFiles,
  schemaColumns,
  tableWriteViolations,
  type QueryInput,
} from './table-writes.ts'

const SCHEMA = [
  'CREATE TABLE users (id uuid PRIMARY KEY, name text, tenant_id uuid);',
  'CREATE TABLE jobs (id uuid PRIMARY KEY, state text);',
  'CREATE TABLE audit_events (id uuid PRIMARY KEY, tenant_id uuid, body text);',
  'CREATE TABLE counters (key text PRIMARY KEY, hits int);',
].join('\n')

const columns = schemaColumns(await parseSql(SCHEMA))

async function writesOf(sql: string) {
  return extractTableWrites(await parseSql(sql), columns)
}

describe('extractTableWrites', () => {
  it('reads an explicit UPDATE target and its assigned columns', async () => {
    expect(
      await writesOf('-- name: Rename :exec\nUPDATE users SET name = $1 WHERE id = $2;'),
    ).toEqual({
      writes: [{ query: 'Rename', operation: 'update', table: 'users', columns: ['name'] }],
      diagnostics: [],
    })
  })

  it('reads every target of a writing CTE, an upsert, and a multi-table TRUNCATE', async () => {
    const result = await writesOf(
      [
        '-- name: Move :exec',
        'WITH done AS (DELETE FROM jobs j WHERE j.id = $1 RETURNING id)',
        'INSERT INTO public.audit_events AS ae (id, tenant_id) SELECT id, $2 FROM done',
        'ON CONFLICT (id) DO UPDATE SET body = EXCLUDED.body;',
        '',
        '-- name: Reset :exec',
        'TRUNCATE counters, "jobs";',
      ].join('\n'),
    )

    expect(result).toEqual({
      writes: [
        {
          query: 'Move',
          operation: 'insert',
          table: 'audit_events',
          columns: ['body', 'id', 'tenant_id'],
        },
        { query: 'Move', operation: 'delete', table: 'jobs', columns: ['*'] },
        { query: 'Reset', operation: 'truncate', table: 'counters', columns: ['*'] },
        { query: 'Reset', operation: 'truncate', table: 'jobs', columns: ['*'] },
      ],
      diagnostics: [],
    })
  })

  it('resolves the columns of an INSERT without a column list from the schema', async () => {
    expect(
      (await writesOf('-- name: Add :exec\nINSERT INTO jobs VALUES ($1, $2);')).writes,
    ).toEqual([{ query: 'Add', operation: 'insert', table: 'jobs', columns: ['id', 'state'] }])
  })

  it('ignores write-like words in comments and string literals and plain reads', async () => {
    expect(
      await writesOf(
        [
          '-- name: Find :one',
          '-- UPDATE users SET name = 1',
          "SELECT id FROM users WHERE name = 'DELETE FROM jobs' /* INSERT INTO jobs */;",
        ].join('\n'),
      ),
    ).toEqual({ writes: [], diagnostics: [] })
  })

  it('diagnoses a statement that the parser does not convert, such as MERGE', async () => {
    const result = await writesOf(
      '-- name: Sync :exec\nMERGE INTO users u USING jobs j ON u.id = j.id WHEN MATCHED THEN DELETE;',
    )

    expect(result.diagnostics).toEqual([
      'Sync: the SQL parser does not expose this statement; table writes cannot be extracted',
    ])
  })

  it('diagnoses a write target that the schema does not declare or a non-public schema', async () => {
    const result = await writesOf(
      '-- name: Odd :exec\nUPDATE "Users" SET name = 1;\n-- name: Other :exec\nDELETE FROM audit.jobs;',
    )

    expect(result.diagnostics).toEqual([
      'Odd: write target Users is not a table of the schema',
      'Other: write target audit.jobs is outside the public schema',
    ])
  })

  it('fails on SQL that does not parse', async () => {
    await expect(parseSql('UPDAT users SET name = 1;')).rejects.toThrow('syntax error')
  })
})

describe('parseSqlFiles', () => {
  it('assigns each statement of a combined parse to the file that contains it', async () => {
    const parsed = await parseSqlFiles([
      {
        path: 'a.sql',
        sql: '-- name: A :exec\nDELETE FROM jobs;\n-- name: B :exec\nDELETE FROM users;',
      },
      { path: 'b.sql', sql: '-- name: C :exec\n-- 日本語のコメント\nUPDATE jobs SET state = 1;\n' },
    ])

    expect(
      [...parsed].map(([path, statements]) => [
        path,
        extractTableWrites(statements, columns).writes.map((write) => write.query),
      ]),
    ).toEqual([
      ['a.sql', ['A', 'B']],
      ['b.sql', ['C']],
    ])
  })

  it('names the file whose SQL does not parse', async () => {
    await expect(
      parseSqlFiles([
        { path: 'good.sql', sql: 'DELETE FROM jobs;' },
        { path: 'bad.sql', sql: 'DELET FROM jobs;' },
      ]),
    ).rejects.toThrow('bad.sql: sqlc parse')
  })
})

describe('tableWriteViolations', () => {
  const owners = new Map([
    ['users', 'IdManagement'],
    ['jobs', 'Jobs'],
    ['counters', '共通基盤'],
  ])

  function input(writer: QueryInput['writer'], table: string, operation = 'update' as const) {
    return {
      path: 'backend/x/db_postgres/q.sql',
      writer,
      writes: [{ query: 'Q', operation, table, columns: ['name'] }],
    }
  }

  it('rejects a module writing a table another module owns', () => {
    expect(
      tableWriteViolations([input({ kind: 'module', name: 'Jobs' }, 'users')], owners).violations,
    ).toEqual([
      expect.objectContaining({
        id: 'table-write:Jobs:backend/x/db_postgres/q.sql:Q:update:users:name',
        debtId: 'table-write:Jobs->IdManagement',
        sourceModule: 'Jobs',
        targetModule: 'IdManagement',
      }),
    ])
  })

  it('rejects a module writing shared infrastructure and a shared library writing a module table', () => {
    expect(
      tableWriteViolations(
        [
          input({ kind: 'module', name: 'Jobs' }, 'counters'),
          input({ kind: 'shared' }, 'jobs'),
          input({ kind: 'composition' }, 'jobs'),
        ],
        owners,
      ).violations.map((violation) => violation.debtId),
    ).toEqual([
      'table-write:Jobs->共通基盤',
      'table-write:shared->Jobs',
      'table-write:System->Jobs',
    ])
  })

  it('accepts writes to owned tables and shared infrastructure written by a shared library', () => {
    expect(
      tableWriteViolations(
        [input({ kind: 'module', name: 'Jobs' }, 'jobs'), input({ kind: 'shared' }, 'counters')],
        owners,
      ),
    ).toEqual({ violations: [], diagnostics: [] })
  })

  it('distinguishes a write that starts touching another column', () => {
    const base = input({ kind: 'module', name: 'Jobs' }, 'users')
    const widened = { ...base, writes: [{ ...base.writes[0]!, columns: ['name', 'tenant_id'] }] }

    expect(tableWriteViolations([widened], owners).violations[0]?.id).toBe(
      'table-write:Jobs:backend/x/db_postgres/q.sql:Q:update:users:name,tenant_id',
    )
  })

  it('diagnoses a written table without an owner', () => {
    expect(
      tableWriteViolations([input({ kind: 'module', name: 'Jobs' }, 'audit_events')], owners)
        .diagnostics,
    ).toEqual(['backend/x/db_postgres/q.sql: Q writes audit_events, which has no owning module'])
  })
})
