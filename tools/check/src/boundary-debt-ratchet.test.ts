import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { checkBoundaryDebtRatchet, type RevisionReader } from './boundary-debt-ratchet.ts'
import { addedBoundaryDebtViolations, type BoundaryDebtEntry } from './boundary-fitness.ts'

const before: BoundaryDebtEntry[] = [
  {
    id: 'private-import:Customer->Supplier',
    sourceContext: 'Customer',
    targetContext: 'Supplier',
    violations: ['private-import:Customer:customer/a->supplier/private'],
    reason: 'Customer still constructs the supplier workflow through its private package.',
  },
]

describe('addedBoundaryDebtViolations', () => {
  it('reports a concrete violation added inside an existing Context-pair entry', () => {
    expect(
      addedBoundaryDebtViolations(before, [
        {
          ...before[0]!,
          violations: [
            ...before[0]!.violations,
            'private-import:Customer:customer/b->supplier/private',
          ],
        },
      ]),
    ).toEqual(['private-import:Customer:customer/b->supplier/private'])
  })

  it('allows removals and reason-only changes', () => {
    expect(
      addedBoundaryDebtViolations(before, [
        { ...before[0]!, violations: [], reason: 'The final private import was removed.' },
      ]),
    ).toEqual([])
  })
})

describe('checkBoundaryDebtRatchet', () => {
  it('allows initial adoption when the base revision has no boundary ledger', async () => {
    const reader: RevisionReader = { read: () => undefined }
    const outcome = await checkBoundaryDebtRatchet(
      createWorkspaceSnapshot(resolve(process.cwd(), '..')),
      { verbose: false, listUnresolved: false, baseRevision: 'base' },
      reader,
    )

    expect(outcome).toEqual({
      ok: true,
      lines: ['ok  boundary debt ratchet adopted against base'],
    })
  })

  it('rejects violations added after the base revision', async () => {
    const reader: RevisionReader = { read: () => '{"violations":[]}' }
    const outcome = await checkBoundaryDebtRatchet(
      createWorkspaceSnapshot(resolve(process.cwd(), '..')),
      { verbose: false, listUnresolved: false, baseRevision: 'base' },
      reader,
    )

    expect(outcome.ok).toBe(false)
    expect(outcome.lines[0]).toContain('absent from base')
  })
})
