import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { BOUNDARY_FIXTURE } from './boundary-fixture.ts'
import { checkBoundaries } from './check-boundaries.ts'
import { checkSpecificationRules } from './check-specification-rules.ts'

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
  it('テスト専用の環境読み取りを許し、本番から到達した場合は拒否する', async () => {
    const support = {
      'backend/shared/testing_fixture/harness.go':
        'package fixture\nimport "os"\nfunc AllowSkip() bool { return os.Getenv("PGTEST_ALLOW_SKIP") == "1" }\n',
    }
    expect((await outcomeFor(support)).ok).toBe(true)
    const lines = await failureFor({
      ...support,
      'backend/cmd/server/main.go':
        'package main\nimport _ "example.com/product/backend/shared/testing_fixture"\nfunc main() {}\n',
    })
    expect(lines).toContain('harness.go: startup configuration must be read through bootstrap')
  })

  const implementationTable = `${BOUNDARY_FIXTURE['docs/modules/README.md']!.replace(
    'Go パッケージ',
    '実装パス',
  )}| [Frontend](frontend/README.md) | \`HTTP\` | なし | \`frontend\` | 画面と中継 |\n`

  it('非 Go の Frontend 宣言と既存の Go モジュールを同時に受け入れる', async () => {
    expect(
      await outcomeFor({
        'docs/modules/README.md': implementationTable,
        'frontend/src/main.ts': 'export const title = "IdMagic"\n',
      }),
    ).toEqual({ ok: true, lines: ['ok  module boundaries'] })
  })

  it('Frontend 宣言の追加後も未宣言の Go パッケージを拒否する', async () => {
    expect(
      await failureFor({
        'docs/modules/README.md': implementationTable,
        'backend/orphan/model.go': 'package orphan\n',
      }),
    ).toContain('belongs to no module')
  })

  it('Frontend と別のモジュールによる重複した実装の分類を拒否する', async () => {
    await expect(
      outcomeFor({
        'docs/modules/README.md': `${implementationTable}| [OtherUI](other/README.md) | \`HTTP\` | なし | \`frontend\` | 重複 |\n`,
      }),
    ).rejects.toThrow('frontend is assigned more than once')
  })

  it('accepts a workspace whose imports and table writes follow the declarations', async () => {
    expect(
      await outcomeFor({
        'backend/customer/internal/usecases/service.go':
          'package usecases\nimport "example.com/product/backend/supplier/ports"\n',
      }),
    ).toEqual({ ok: true, lines: ['ok  module boundaries'] })
  })

  it('rejects a cross-module import of a private package', async () => {
    expect(
      await failureFor({
        'backend/customer/internal/usecases/service.go':
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
              'private-import:Customer:backend/customer/internal/usecases->backend/supplier/usecases',
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
      'backend/customer/internal/usecases/service.go':
        'package usecases\nimport "example.com/product/backend/supplier/usecases"\n',
      'tools/check/boundary-debt.json': JSON.stringify({
        violations: [
          {
            id: 'private-import:Customer->Supplier',
            sourceModule: 'Customer',
            targetModule: 'Supplier',
            violations: [
              'private-import:Customer:backend/customer/internal/usecases->backend/supplier/usecases',
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

describe('Frontend へ移した仕様の検査', () => {
  it.each([
    ['利用者が画面を開いたとき、System は、画面を表示する。', true],
    ['利用者が画面を開いたとき、画面を表示する。', false],
  ])('既存の System の要件文にも EARS の検査を保つ: %s', async (statement, ok) => {
    const root = await boundaryWorkspace({
      'docs/modules/frontend/README.md': '# Frontend\n',
      'docs/modules/frontend/features/localization.md': [
        '# 表示言語',
        '## 概要',
        '画面を表示する。',
        '## 操作',
        '### 画面の表示',
        '#### REQ-SYSTEM-003 表示言語で画面を表示する',
        `- ${statement}`,
        '',
      ].join('\n'),
    })
    const outcome = await checkSpecificationRules(createWorkspaceSnapshot(root))
    expect(outcome.ok).toBe(ok)
    if (!ok) expect(outcome.lines.join('\n')).toContain('statement names no responder')
  })
})
