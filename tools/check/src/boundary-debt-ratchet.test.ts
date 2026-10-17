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

/** 基準に既にある循環、非公開パッケージへの import、所有者外の書き込み。 */
const EXISTING_DEBT: Record<string, string> = {
  'backend/customer/internal/usecases/a.go':
    'package usecases\nimport "example.com/product/backend/supplier/domain"\n',
  'backend/supplier/internal/usecases/a.go':
    'package usecases\nimport "example.com/product/backend/customer/ports"\n',
  'backend/customer/internal/usecases/private.go':
    'package usecases\nimport "example.com/product/backend/supplier/internal/usecases"\n',
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

/** 既存の負債と `baseChanges` を基準 commit にし、既存の負債へ `changes` を重ねた作業ツリーを作る。 */
async function repository(
  changes: Record<string, string>,
  baseChanges: Record<string, string> = {},
): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'boundary-ratchet-'))
  cleanup.push(root)
  await write(root, { ...BOUNDARY_FIXTURE, ...EXISTING_DEBT, ...baseChanges })
  git(root, 'init', '-q', '-b', 'main')
  git(root, 'add', '-A')
  git(root, 'commit', '-q', '-m', 'base')
  await write(root, { ...BOUNDARY_FIXTURE, ...EXISTING_DEBT, ...changes })
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
  it('accepts a workspace that keeps the existing violations of the base revision', async () => {
    const outcome = await ratchet(await repository({}))

    expect(outcome).toEqual({
      ok: true,
      lines: [
        'ok  boundary debt ratchet against main (module-cycle 2, private-import 1, table-write 1 -> module-cycle 2, private-import 1, table-write 1)',
      ],
    })
  })

  it('rejects a new violation even when the ledger records it', async () => {
    const outcome = await ratchet(
      await repository({
        'backend/customer/internal/usecases/b.go':
          'package usecases\nimport "example.com/product/backend/supplier/internal/adapter"\n',
        'tools/check/boundary-debt.json': JSON.stringify({
          violations: [
            {
              id: 'private-import:Customer->Supplier',
              sourceModule: 'Customer',
              targetModule: 'Supplier',
              violations: [
                'private-import:Customer:backend/customer/internal/usecases->backend/supplier/internal/adapter',
              ],
              reason: 'Customer still constructs the supplier workflow directly.',
            },
          ],
        }),
      }),
    )

    expect(outcome.ok).toBe(false)
    expect(outcome.lines[0]).toBe(
      'fail  private-import:Customer:backend/customer/internal/usecases->backend/supplier/internal/adapter: absent from main; remove the new dependency or write instead of growing boundary debt.',
    )
  })

  it('does not read an existing private import moved to another source package as the same violation', async () => {
    const root = await repository({
      'backend/customer/internal/workflow/private.go':
        EXISTING_DEBT['backend/customer/internal/usecases/private.go']!,
    })
    await rm(join(root, 'backend/customer/internal/usecases/private.go'))

    const outcome = await ratchet(root)

    expect(outcome.ok).toBe(false)
    expect(outcome.lines[0]).toContain(
      'private-import:Customer:backend/customer/internal/workflow->backend/supplier/internal/usecases',
    )
  })

  it('fails when the base revision declares the retired legacy publication mode', async () => {
    const root = await repository(
      {},
      {
        'docs/design/architecture/logical.md': BOUNDARY_FIXTURE[
          'docs/design/architecture/logical.md'
        ]!.replace(
          '| `internal` | `backend/supplier/domain`, `backend/supplier/ports` |',
          '| `legacy` | `domain` または `ports` の区画 |',
        ),
      },
    )

    await expect(ratchet(root)).rejects.toThrow('Supplier: publication mode must be internal')
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
        'docs/design/data/database.md': BOUNDARY_FIXTURE['docs/design/data/database.md']!.replace(
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
