import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { checkBoundaryDebtRatchet } from './boundary-debt-ratchet.ts'
import { BOUNDARY_FIXTURE } from './boundary-fixture.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

/** 責務表と所有の列を改める前の形式。基準 revision がこの形を持つ。 */
const RETIRED_DOCUMENTS: Record<string, string> = {
  'docs/design/architecture/logical.md': [
    '# 論理アーキテクチャ',
    '',
    '## Context Map',
    '',
    '```mermaid',
    'flowchart LR',
    '  Supplier -->|OHS/PL: model| Customer',
    '```',
    '',
    '## Context の責務',
    '',
    '| 仕様上の Context | Subdomain | Go パッケージ | 責務 |',
    '| --- | --- | --- | --- |',
    '| [System](system.md) | Supporting | `backend/cmd/internal/bootstrap` | Wiring |',
    '| [Supplier](supplier.md) | Core | `backend/supplier` | Supplies a model |',
    '| [Customer](customer.md) | Core | `backend/customer` | Uses the model |',
    '',
  ].join('\n'),
  'docs/design/data/database.md': BOUNDARY_FIXTURE['docs/design/data/database.md']!.replace(
    '所有モジュール',
    '所有 Context',
  ),
}

/** 基準に既にある循環と所有者外の書き込み。 */
const EXISTING_DEBT: Record<string, string> = {
  'backend/customer/usecases/a.go':
    'package usecases\nimport "example.com/product/backend/supplier/domain"\n',
  'backend/supplier/usecases/a.go':
    'package usecases\nimport "example.com/product/backend/customer/ports"\n',
  'backend/customer/db_postgres/supplies.sql':
    '-- name: Reserve :exec\nUPDATE supplies SET stock = stock - 1 WHERE id = $1;\n',
}

async function write(root: string, files: Record<string, string>): Promise<void> {
  for (const [path, source] of Object.entries(files)) {
    await mkdir(dirname(join(root, path)), { recursive: true })
    await writeFile(join(root, path), source)
  }
}

function git(root: string, ...args: string[]): void {
  const result = Bun.spawnSync(
    ['git', '-c', 'user.name=t', '-c', 'user.email=t@example.com', ...args],
    { cwd: root },
  )
  if (result.exitCode !== 0) throw new Error(result.stderr.toString())
}

/** 旧形式の文書と既存の負債を基準 commit にし、現在の作業ツリーへ `changes` を書く。 */
async function repository(
  changes: Record<string, string>,
  baseChanges: Record<string, string> = {},
): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'boundary-ratchet-'))
  cleanup.push(root)
  await write(root, { ...BOUNDARY_FIXTURE, ...RETIRED_DOCUMENTS, ...EXISTING_DEBT, ...baseChanges })
  git(root, 'init', '-q', '-b', 'main')
  git(root, 'add', '-A')
  git(root, 'commit', '-q', '-m', 'base')
  await write(root, {
    'docs/design/architecture/logical.md': BOUNDARY_FIXTURE['docs/design/architecture/logical.md']!,
    'docs/design/data/database.md': BOUNDARY_FIXTURE['docs/design/data/database.md']!,
    ...changes,
  })
  return root
}

async function ratchet(root: string, baseRevision = 'main') {
  return checkBoundaryDebtRatchet(createWorkspaceSnapshot(root), {
    verbose: false,
    listUnresolved: false,
    baseRevision,
  })
}

describe('checkBoundaryDebtRatchet', () => {
  it('accepts moving the cycle and the direct write of a retired-format base to new ids', async () => {
    const outcome = await ratchet(await repository({}))

    expect(outcome).toEqual({
      ok: true,
      lines: [
        'ok  boundary debt ratchet against main (module-cycle 2, table-write 1 -> module-cycle 2, table-write 1)',
      ],
    })
  })

  it('rejects a new violation added in the migration diff even when the ledger records it', async () => {
    const outcome = await ratchet(
      await repository({
        'backend/customer/usecases/b.go':
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
              reason: 'Customer still constructs the supplier workflow directly.',
            },
          ],
        }),
      }),
    )

    expect(outcome.ok).toBe(false)
    expect(outcome.lines[0]).toBe(
      'fail  private-import:Customer:backend/customer/usecases->backend/supplier/usecases: absent from main; remove the new dependency or write instead of growing boundary debt.',
    )
  })

  it('rejects an existing out-of-owner write that starts writing another column', async () => {
    const outcome = await ratchet(
      await repository({
        'backend/customer/db_postgres/supplies.sql':
          "-- name: Reserve :exec\nUPDATE supplies SET stock = stock - 1, name = 'x' WHERE id = $1;\n",
      }),
    )

    expect(outcome.ok).toBe(false)
    expect(outcome.lines[0]).toContain(
      'table-write:Customer:backend/customer/db_postgres/supplies.sql:Reserve:update:supplies:name,stock',
    )
  })

  it('fails when the base revision declarations cannot be analyzed', async () => {
    const root = await repository(
      {},
      {
        'docs/design/data/database.md': RETIRED_DOCUMENTS['docs/design/data/database.md']!.replace(
          '| `supplies` | Supplies | Supplier | `LOGGED` | なし |\n',
          '',
        ),
      },
    )

    const outcome = await ratchet(root)

    expect(outcome.ok).toBe(false)
    expect(outcome.lines).toEqual([
      'fail  main cannot be analyzed: backend/customer/db_postgres/supplies.sql: Reserve writes supplies, which has no owning module',
    ])
  })

  it('fails when the base revision cannot be read', async () => {
    const root = await repository({})

    await expect(ratchet(root, 'no-such-revision')).rejects.toThrow(
      'cannot resolve Git revision no-such-revision',
    )
  })

  it('requires a base revision', async () => {
    expect(
      await checkBoundaryDebtRatchet(createWorkspaceSnapshot(await repository({})), {
        verbose: false,
        listUnresolved: false,
      }),
    ).toEqual({
      ok: false,
      lines: ['boundary-debt-ratchet requires --base-revision <git-revision>'],
    })
  })
})
