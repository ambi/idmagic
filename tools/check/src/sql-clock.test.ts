import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { checkSqlClock } from './check-sql-clock.ts'
import { clockReadings } from './sql-clock.ts'
import { parseSql } from './table-writes.ts'

async function readings(sql: string): Promise<string[]> {
  return clockReadings(await parseSql(sql)).map(
    (reading) => `${reading.query}: ${reading.expression}`,
  )
}

describe('clockReadings', () => {
  it('代入する値と VALUES の項目で記録する時刻を許す', async () => {
    expect(
      await readings(
        [
          '-- name: Touch :exec',
          'UPDATE t SET updated_at = now(), seen_at = CURRENT_TIMESTAMP WHERE id = $1;',
          '-- name: Insert :exec',
          'INSERT INTO t (id, created_at) VALUES ($1, now())',
          '  ON CONFLICT (id) DO UPDATE SET updated_at = now();',
          '-- name: Complete :exec',
          "UPDATE t SET completed_at = CASE WHEN $2 = 'done' THEN now() ELSE completed_at END WHERE id = $1;",
          '-- name: Reopen :exec',
          "UPDATE t SET reopened_at = CASE WHEN $2 = 'done' THEN reopened_at ELSE now() END WHERE id = $1;",
        ].join('\n'),
      ),
    ).toEqual([])
  })

  it('期限と順序の判定に使う時刻を拒否する', async () => {
    expect(
      await readings(
        [
          '-- name: Live :many',
          'SELECT id FROM t WHERE expires_at <= now();',
          '-- name: Oldest :many',
          'SELECT id FROM t ORDER BY now() - created_at;',
          '-- name: Reserve :one',
          'INSERT INTO t (id) VALUES ($1) ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id',
          '  WHERE t.expires_at <= CURRENT_TIMESTAMP RETURNING id;',
          '-- name: Expire :exec',
          'UPDATE t SET expires_at = now() + interval $2 WHERE id = $1;',
          '-- name: Select :one',
          'SELECT now();',
        ].join('\n'),
      ),
    ).toEqual([
      'Live: now()',
      'Oldest: now()',
      'Reserve: CURRENT_TIMESTAMP',
      'Expire: now()',
      'Select: now()',
    ])
  })

  it('CASE の条件と、別名や型変換で書いた時計を拒否する', async () => {
    expect(
      await readings(
        [
          '-- name: Condition :exec',
          'UPDATE t SET state = CASE WHEN expires_at <= now() THEN 1 ELSE 0 END WHERE id = $1;',
          '-- name: Pick :many',
          'SELECT CASE WHEN $1 THEN now() ELSE expires_at END FROM t;',
          '-- name: Qualified :many',
          'SELECT id FROM t WHERE expires_at <= pg_catalog.now();',
          '-- name: Clock :many',
          'SELECT id FROM t WHERE expires_at <= clock_timestamp();',
          '-- name: Local :many',
          'SELECT id FROM t WHERE expires_at <= LOCALTIMESTAMP;',
          '-- name: Literal :many',
          "SELECT id FROM t WHERE expires_at <= 'now'::timestamptz;",
          '-- name: Cte :many',
          'WITH gone AS (DELETE FROM t WHERE expires_at < now() RETURNING id) SELECT count(*) FROM gone;',
        ].join('\n'),
      ),
    ).toEqual([
      'Condition: now()',
      'Pick: now()',
      'Qualified: pg_catalog.now()',
      'Clock: clock_timestamp()',
      'Local: LOCALTIMESTAMP',
      "Literal: 'now'::",
      'Cte: now()',
    ])
  })

  it('構文木へ変換されない文は、本文が時計を読めば拒否する', async () => {
    expect(
      await readings(
        [
          '-- name: Merge :exec',
          'MERGE INTO t USING s ON t.id = s.id WHEN MATCHED AND t.expires_at < now() THEN DELETE;',
          '-- name: MergeByParameter :exec',
          'MERGE INTO t USING s ON t.id = s.id WHEN MATCHED AND t.expires_at < @now THEN DELETE;',
        ].join('\n'),
      ),
    ).toEqual(['Merge: a clock read the SQL parser does not expose'])
  })

  it('時刻でない SQL の値関数と、パラメーターの比較を許す', async () => {
    expect(
      await readings(
        [
          '-- name: Mine :many',
          'SELECT id FROM t WHERE owner = CURRENT_USER AND expires_at <= @now;',
        ].join('\n'),
      ),
    ).toEqual([])
  })
})

describe('checkSqlClock', () => {
  const cleanup: string[] = []
  afterAll(async () => {
    for (const path of cleanup) await rm(path, { recursive: true, force: true })
  })

  async function workspace(files: Record<string, string>): Promise<string> {
    const root = await mkdtemp(join(tmpdir(), 'check-sql-clock-'))
    cleanup.push(root)
    for (const [path, source] of Object.entries(files)) {
      await mkdir(dirname(join(root, path)), { recursive: true })
      await writeFile(join(root, path), source)
    }
    return root
  }

  it('sqlc.yaml が宣言するクエリのファイルだけを読む', async () => {
    const root = await workspace({
      'sqlc.yaml':
        'version: "2"\nsql:\n  - schema: schema.sql\n    queries: backend/a/db_postgres\n',
      'schema.sql': 'CREATE TABLE t (id TEXT, expires_at TIMESTAMPTZ);\n',
      'backend/a/db_postgres/q.sql':
        '-- name: Live :many\nSELECT id FROM t WHERE expires_at <= now();\n',
      'backend/b/migration.sql': 'SELECT id FROM t WHERE expires_at <= now();\n',
    })
    const outcome = await checkSqlClock(createWorkspaceSnapshot(root))
    expect(outcome.ok).toBe(false)
    expect(outcome.lines).toEqual([
      'fail  backend/a/db_postgres/q.sql: Live reads the database clock with now() outside an assigned value; pass the time as a query parameter',
    ])
  })
})
