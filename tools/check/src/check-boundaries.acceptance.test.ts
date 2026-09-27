import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { checkBoundaries } from './check-boundaries.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

const logicalArchitecture = [
  '# Logical architecture',
  '',
  '## Context Map',
  '',
  '```mermaid',
  'flowchart LR',
  '  Supplier -->|OHS/PL: published model| Customer',
  '```',
  '',
  '## Context responsibilities',
  '',
  '| Specification context | Subdomain | Go package | Responsibility |',
  '| --- | --- | --- | --- |',
  '| [Supplier](supplier.md) | Core | `backend/supplier` | Supplies a model |',
  '| [Customer](customer.md) | Core | `backend/customer` | Uses the model |',
  '| [Other](other.md) | Core | `backend/other` | Has no declared relation |',
  '',
].join('\n')

async function boundaryWorkspace(
  sources: Record<string, string>,
  debt = '{"violations":[]}\n',
): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'check-boundaries-acceptance-'))
  cleanup.push(root)
  await mkdir(join(root, 'docs', 'design', 'architecture'), { recursive: true })
  await mkdir(join(root, 'tools', 'check'), { recursive: true })
  await writeFile(join(root, 'go.mod'), 'module example.com/product\n')
  await writeFile(join(root, 'docs', 'design', 'architecture', 'logical.md'), logicalArchitecture)
  await writeFile(join(root, 'tools', 'check', 'boundary-debt.json'), debt)
  for (const [path, source] of Object.entries(sources)) {
    await mkdir(dirname(join(root, path)), { recursive: true })
    await writeFile(join(root, path), source)
  }
  return root
}

async function outcomeFor(sources: Record<string, string>, debt?: string): Promise<string> {
  const outcome = await checkBoundaries(
    createWorkspaceSnapshot(await boundaryWorkspace(sources, debt)),
  )
  expect(outcome.ok).toBe(false)
  return outcome.lines.join('\n')
}

describe('checkBoundaries context fitness functions', () => {
  it('rejects a cross-context import of a private package', async () => {
    const lines = await outcomeFor({
      'backend/customer/usecases/service.go':
        'package usecases\nimport "example.com/product/backend/supplier/usecases"\n',
    })

    expect(lines).toContain('private-import:Customer->Supplier')
  })

  it('rejects an undeclared Context dependency', async () => {
    const lines = await outcomeFor({
      'backend/supplier/usecases/service.go':
        'package usecases\nimport "example.com/product/backend/customer/domain"\n',
    })

    expect(lines).toContain('undeclared-context-edge:Supplier->Customer')
  })

  it('rejects an effectful call in a domain package', async () => {
    const lines = await outcomeFor({
      'backend/customer/domain/model.go':
        'package domain\nimport "time"\nfunc CreatedAt() time.Time { return time.Now() }\n',
    })

    expect(lines).toContain('domain-effect:customer')
  })

  it('rejects a forbidden dependency hidden behind shared code', async () => {
    const lines = await outcomeFor({
      'backend/customer/usecases/service.go':
        'package usecases\nimport "example.com/product/backend/shared/bridge"\n',
      'backend/shared/bridge/bridge.go':
        'package bridge\nimport "example.com/product/backend/other/usecases"\n',
    })

    expect(lines).toContain('shared-detour:Customer->Other')
  })

  it('rejects a debt entry without a concrete reason', async () => {
    const root = await boundaryWorkspace(
      {
        'backend/customer/usecases/service.go':
          'package usecases\nimport "example.com/product/backend/supplier/usecases"\n',
      },
      JSON.stringify({
        violations: [
          {
            id: 'private-import:Customer->Supplier',
            sourceContext: 'Customer',
            targetContext: 'Supplier',
            violations: [
              'private-import:Customer:backend/customer/usecases->backend/supplier/usecases',
            ],
          },
        ],
      }),
    )

    await expect(checkBoundaries(createWorkspaceSnapshot(root))).rejects.toThrow(
      'every entry needs id, sourceContext, optional targetContext, violations, and reason',
    )
  })
})
