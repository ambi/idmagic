import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { BOUNDARY_FIXTURE } from './boundary-fixture.ts'
import { checkBoundaries } from './check-boundaries.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

async function boundaryWorkspace(files: Record<string, string>): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'check-boundaries-acceptance-'))
  cleanup.push(root)
  for (const [path, source] of Object.entries({ ...BOUNDARY_FIXTURE, ...files })) {
    await mkdir(dirname(join(root, path)), { recursive: true })
    await writeFile(join(root, path), source)
  }
  return root
}

async function outcomeFor(files: Record<string, string>) {
  return checkBoundaries(createWorkspaceSnapshot(await boundaryWorkspace(files)))
}

async function failureFor(files: Record<string, string>): Promise<string> {
  const outcome = await outcomeFor(files)
  expect(outcome.ok).toBe(false)
  return outcome.lines.join('\n')
}

describe('checkBoundaries module rules', () => {
  it('accepts a workspace whose imports and table writes follow the declarations', async () => {
    expect(
      await outcomeFor({
        'backend/customer/usecases/service.go':
          'package usecases\nimport "example.com/product/backend/supplier/ports"\n',
      }),
    ).toEqual({ ok: true, lines: ['ok  module boundaries'] })
  })

  it('rejects a cross-module import of a private package', async () => {
    expect(
      await failureFor({
        'backend/customer/usecases/service.go':
          'package usecases\nimport "example.com/product/backend/supplier/usecases"\n',
      }),
    ).toContain('private-import:Customer->Supplier')
  })

  it('rejects an effectful call in a domain package', async () => {
    expect(
      await failureFor({
        'backend/customer/domain/model.go':
          'package domain\nimport "time"\nfunc CreatedAt() time.Time { return time.Now() }\n',
      }),
    ).toContain('domain-effect:customer')
  })

  it('rejects a module query that updates a table another module owns', async () => {
    expect(
      await failureFor({
        'backend/customer/db_postgres/supplies.sql':
          '-- name: Reserve :exec\nUPDATE supplies SET stock = stock - 1 WHERE id = $1;\n',
      }),
    ).toContain('table-write:Customer->Supplier')
  })

  it('rejects a shared library writing a module table', async () => {
    expect(
      await failureFor({
        'backend/shared/counter/db_postgres/orders.sql':
          '-- name: Purge :exec\nDELETE FROM orders;\n',
      }),
    ).toContain('table-write:shared->Customer')
  })

  it('checks a query input added under another directory once sqlc declares it', async () => {
    const lines = await failureFor({
      'sqlc.yaml': `${BOUNDARY_FIXTURE['sqlc.yaml']}  - engine: "postgresql"\n    schema: "infra/schema/postgres.sql"\n    queries: "backend/supplier/reports/db_postgres"\n`,
      'backend/supplier/reports/db_postgres/report.sql':
        '-- name: Touch :exec\nUPDATE orders SET supply_id = NULL;\n',
    })

    expect(lines).toContain('table-write:Supplier->Customer')
  })

  it('diagnoses an undescribed write target owner and an unclassified query input', async () => {
    const lines = await failureFor({
      'docs/design/data/database.md': BOUNDARY_FIXTURE['docs/design/data/database.md']!.replace(
        '| `counters` | Counters | 共通基盤 | `LOGGED` | なし |\n',
        '',
      ),
      'sqlc.yaml': `${BOUNDARY_FIXTURE['sqlc.yaml']}  - engine: "postgresql"\n    schema: "infra/schema/postgres.sql"\n    queries: "backend/orphan/db_postgres"\n`,
      'backend/orphan/db_postgres/q.sql': '-- name: Q :exec\nDELETE FROM supplies;\n',
    })

    expect(lines).toContain('Hit writes counters, which has no owning module')
    expect(lines).toContain(
      'backend/orphan/db_postgres: query input belongs to no module, composition point, or shared library',
    )
  })

  it('diagnoses SQL whose write targets cannot be extracted', async () => {
    expect(
      await failureFor({
        'backend/customer/db_postgres/merge.sql':
          '-- name: Sync :exec\nMERGE INTO orders o USING supplies s ON o.supply_id = s.id WHEN MATCHED THEN DELETE;\n',
      }),
    ).toContain('Sync: the SQL parser does not expose this statement')
  })

  it('rejects a ledger id that no observed violation has', async () => {
    const lines = await failureFor({
      'tools/check/boundary-debt.json': JSON.stringify({
        violations: [
          {
            id: 'private-import:Customer->Supplier',
            sourceModule: 'Customer',
            targetModule: 'Supplier',
            violations: [
              'private-import:Customer:backend/customer/usecases->backend/supplier/usecases',
            ],
            reason: 'Customer still constructs the supplier workflow directly.',
          },
        ],
      }),
    })

    expect(lines).toContain('no observed violation belongs to this entry')
  })

  it('rejects a debt entry without a concrete reason', async () => {
    const root = await boundaryWorkspace({
      'backend/customer/usecases/service.go':
        'package usecases\nimport "example.com/product/backend/supplier/usecases"\n',
      'tools/check/boundary-debt.json': JSON.stringify({
        violations: [
          {
            id: 'private-import:Customer->Supplier',
            sourceModule: 'Customer',
            targetModule: 'Supplier',
            violations: [
              'private-import:Customer:backend/customer/usecases->backend/supplier/usecases',
            ],
          },
        ],
      }),
    })

    await expect(checkBoundaries(createWorkspaceSnapshot(root))).rejects.toThrow(
      'every entry needs id, sourceModule, optional targetModule, violations, and reason',
    )
  })
})
