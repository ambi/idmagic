import { describe, expect, it } from 'bun:test'
import { addedDebtIds } from './coverage-debt-ratchet.ts'

describe('addedDebtIds', () => {
  it('reports an id absent from the base ledger', () => {
    expect(
      addedDebtIds(
        [{ id: 'EX-DEMO-001-01', reason: 'uncovered at ratchet adoption' }],
        [
          { id: 'EX-DEMO-001-01', reason: 'updated investigation' },
          { id: 'EX-DEMO-002-01', reason: 'incorrectly reintroduced' },
        ],
      ),
    ).toEqual(['EX-DEMO-002-01'])
  })

  it('allows removals and reason-only changes', () => {
    expect(
      addedDebtIds(
        [
          { id: 'RFC-A', reason: 'first reason' },
          { id: 'RFC-B', reason: 'second reason' },
        ],
        [{ id: 'RFC-A', reason: 'corrected reason' }],
      ),
    ).toEqual([])
  })
})
