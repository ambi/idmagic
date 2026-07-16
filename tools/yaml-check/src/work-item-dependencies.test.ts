import { describe, expect, it } from 'bun:test'
import { verifyWorkItemDependencies } from './work-item-dependencies.ts'

const record = (id: string, depends_on: string[] = []) => ({
  id,
  depends_on,
  path: `work-items/${id}.md`,
  depends_on_line: 6,
})

describe('verifyWorkItemDependencies', () => {
  it('accepts an acyclic graph including completed prerequisites', () => {
    expect(
      verifyWorkItemDependencies([
        record('wi-1-foundation'),
        record('wi-2-feature', ['wi-1-foundation']),
      ]),
    ).toEqual([])
  })

  it('reports an unknown prerequisite at the dependency field', () => {
    const findings = verifyWorkItemDependencies([record('wi-2-feature', ['wi-9-missing'])])
    expect(findings[0]).toMatchObject({
      line: 6,
      message: expect.stringContaining('unknown work item'),
    })
  })

  it('reports self dependency', () => {
    const findings = verifyWorkItemDependencies([record('wi-2-feature', ['wi-2-feature'])])
    expect(findings[0]?.message).toContain('must not depend on itself')
  })

  it('reports indirect cycles once', () => {
    const findings = verifyWorkItemDependencies([
      record('wi-1-a', ['wi-2-b']),
      record('wi-2-b', ['wi-3-c']),
      record('wi-3-c', ['wi-1-a']),
    ])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('wi-1-a -> wi-2-b -> wi-3-c -> wi-1-a')
  })
})
