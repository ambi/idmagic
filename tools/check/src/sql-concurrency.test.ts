import { describe, expect, it } from 'bun:test'
import { concurrencyFindings, parseSqlQueries } from './sql-concurrency.ts'

const sql = [
  '-- name: EnsureCounter :exec',
  'INSERT INTO counters (id) VALUES ($1) ON CONFLICT (id) DO NOTHING;',
  '',
  '-- name: LockCounter :one',
  '-- FOR UPDATE はコメントでも対象にしない見本である。',
  'SELECT n FROM counters WHERE id = $1 FOR UPDATE;',
  '',
  '-- name: ReadCounter :one',
  'SELECT n FROM counters WHERE id = $1;',
  '',
  '-- name: UpsertCounter :exec',
  'INSERT INTO counters (id, n) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET n = EXCLUDED.n;',
  '',
  '-- name: SetCounter :exec',
  'UPDATE counters SET n = $2, updated_at = now() WHERE id = $1;',
  '',
  '-- name: SetCounterIfUnchanged :execrows',
  'UPDATE counters SET n = $2 WHERE id = $1 AND n = $3;',
  '',
  '-- name: InsertAudit :exec',
  'INSERT INTO audits (id) VALUES ($1);',
  '',
  '-- name: SetAudit :exec',
  'UPDATE audits SET n = $2 WHERE id = $1;',
  '',
  '-- name: AdvisoryLock :exec',
  "SELECT pg_advisory_xact_lock(hashtext('x'));",
  '',
  '-- name: ClaimJobs :many',
  'UPDATE jobs SET s = 1 FROM (SELECT id FROM jobs FOR UPDATE SKIP LOCKED) x RETURNING id;',
  '',
].join('\n')
const queries = parseSqlQueries(sql)

function fn(...body: string[]): string {
  return ['package p', '', 'func (r *R) Do(ctx context.Context) error {', ...body, '}', ''].join(
    '\n',
  )
}

const tx = '\tq := New(tx)'
const lock = '\trow, err := q.LockCounter(ctx, id)'
const read = '\trow, err := q.ReadCounter(ctx, id)'
const upsert = '\tq.UpsertCounter(ctx, id, n)'
const noRowContinues = ['\tif errors.Is(err, pgx.ErrNoRows) {', '\t\tn = 1', '\t}']

const problems = (source: string) => concurrencyFindings(queries, source).map((f) => f.problem)

describe('parseSqlQueries', () => {
  it('reads FOR UPDATE only from SELECT statements outside comments', () => {
    expect(queries.byName.get('LockCounter')?.locksRow).toBe(true)
    expect(queries.byName.get('ClaimJobs')?.locksRow).toBe(false)
    const onlyComment = parseSqlQueries('-- name: Q :one\n-- FOR UPDATE\nSELECT 1 FROM t;\n')
    expect(onlyComment.byName.get('Q')?.locksRow).toBe(false)
  })

  it('treats an update as guarded only when WHERE checks an assigned column', () => {
    expect(queries.byName.get('SetCounter')?.guarded).toBe(false)
    expect(queries.byName.get('SetCounterIfUnchanged')?.guarded).toBe(true)
  })

  it('treats an insert as guarded unless it overwrites on conflict', () => {
    expect(queries.byName.get('EnsureCounter')?.guarded).toBe(true)
    expect(queries.byName.get('InsertAudit')?.guarded).toBe(true)
    expect(queries.byName.get('UpsertCounter')?.guarded).toBe(false)
  })
})

describe('row-lock-gap', () => {
  it('reports the switch case that treats a missing row as new', () => {
    const source = fn(
      tx,
      lock,
      '\tswitch {',
      '\tcase errors.Is(err, pgx.ErrNoRows):',
      '\t\tn = 1',
      '\tcase err != nil:',
      '\t\treturn err',
      '\t}',
      upsert,
    )
    expect(concurrencyFindings(queries, source)).toEqual([
      { line: 5, query: 'LockCounter', table: 'counters', problem: 'row-lock-gap' },
    ])
  })

  it('reports an error check that lets a missing row through', () => {
    const source = fn(
      tx,
      lock,
      '\tif err != nil && !errors.Is(err, pgx.ErrNoRows) {',
      '\t\treturn err',
      '\t}',
      upsert,
    )
    expect(problems(source)).toEqual(['row-lock-gap'])
  })

  it('accepts a missing row that returns, or no special case at all', () => {
    expect(
      problems(
        fn(
          tx,
          lock,
          '\tif errors.Is(err, pgx.ErrNoRows) {',
          '\t\treturn ErrMissing',
          '\t}',
          upsert,
        ),
      ),
    ).toEqual([])
    expect(problems(fn(tx, lock, '\tif err != nil {', '\t\treturn err', '\t}', upsert))).toEqual([])
  })

  it('accepts a function that writes only another table', () => {
    expect(problems(fn(tx, lock, ...noRowContinues, '\tq.InsertAudit(ctx, id)'))).toEqual([])
  })

  it('accepts a row ensured or an advisory lock taken before the lock', () => {
    const gap = [lock, ...noRowContinues, upsert]
    expect(problems(fn(tx, '\tq.EnsureCounter(ctx, id)', ...gap))).toEqual([])
    expect(problems(fn(tx, '\tq.AdvisoryLock(ctx)', ...gap))).toEqual([])
  })

  it('still reports a row inserted only after the lock', () => {
    expect(problems(fn(tx, lock, ...noRowContinues, '\tq.EnsureCounter(ctx, id)'))).toEqual([
      'row-lock-gap',
    ])
  })
})

describe('unguarded-write', () => {
  it('reports an unconditional update after an unlocked read of the table', () => {
    const source = fn(tx, read, '\tq.SetCounter(ctx, id, n)')
    expect(concurrencyFindings(queries, source)).toEqual([
      { line: 6, query: 'SetCounter', table: 'counters', problem: 'unguarded-write' },
    ])
  })

  it('reports an overwriting upsert after an unlocked read, once per write', () => {
    expect(problems(fn(tx, read, '\tq.ReadCounter(ctx, id)', upsert))).toEqual(['unguarded-write'])
  })

  it('accepts a conditional update, and an insert that cannot overwrite', () => {
    expect(problems(fn(tx, read, '\tq.SetCounterIfUnchanged(ctx, id, n, row.N)'))).toEqual([])
    expect(problems(fn(tx, read, '\tq.EnsureCounter(ctx, id)', read))).toEqual([])
  })

  it('accepts a write to another table', () => {
    expect(problems(fn(tx, read, '\tq.SetAudit(ctx, id, n)'))).toEqual([])
  })

  it('accepts a lock taken before the write, even through a helper', () => {
    const write = '\tq.SetCounter(ctx, id, n)'
    expect(problems(fn(tx, '\tq.AdvisoryLock(ctx)', read, write))).toEqual([])
    expect(problems(fn(tx, read, '\tq.AdvisoryLock(ctx)', write))).toEqual([])
    const helper =
      'func lockSet(ctx context.Context, q *Queries) error {\n\treturn q.AdvisoryLock(ctx)\n}\n'
    expect(problems(helper + fn(tx, read, '\tlockSet(ctx, q)', write))).toEqual([])
  })

  it('does not count a lock taken only after the write', () => {
    expect(problems(fn(tx, read, '\tq.SetCounter(ctx, id, n)', '\tq.AdvisoryLock(ctx)'))).toEqual([
      'unguarded-write',
    ])
  })
})

describe('lock-outside-tx', () => {
  it('reports FOR UPDATE through the pool', () => {
    expect(problems(fn('\trow, err := New(r.Pool).LockCounter(ctx, id)'))).toEqual([
      'lock-outside-tx',
    ])
  })

  it('accepts FOR UPDATE in a transaction, and a helper handed its queries', () => {
    expect(problems(fn('\ttx, err := r.Pool.Begin(ctx)', tx, lock))).toEqual([])
    expect(
      problems(
        fn('\tq := New(r.Pool)', '\tpgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {', lock),
      ),
    ).toEqual([])
    expect(problems(fn(lock))).toEqual([])
  })
})

describe('exceptions and scope', () => {
  const write = '\tq.SetCounter(ctx, id, n)'

  it('requires a reason on the marker above the reported call', () => {
    expect(
      problems(fn(tx, read, '\t//sql:concurrency-ok the caller holds the connector lock', write)),
    ).toEqual([])
    expect(problems(fn(tx, read, '\t//sql:concurrency-ok', write))).toEqual(['missing-reason'])
  })

  it('judges each function on its own', () => {
    expect(problems([fn(tx, '\tq.AdvisoryLockless(ctx)', read), fn(tx, write)].join('\n'))).toEqual(
      [],
    )
    expect(
      problems(
        [fn(tx, '\tq.EnsureCounter(ctx, id)'), fn(tx, lock, ...noRowContinues, upsert)].join('\n'),
      ),
    ).toEqual(['row-lock-gap'])
  })

  it('skips generated code', () => {
    expect(problems(`// Code generated by sqlc. DO NOT EDIT.\n${fn(tx, read, write)}`)).toEqual([])
  })
})
