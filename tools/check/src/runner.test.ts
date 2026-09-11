import { describe, expect, it } from 'bun:test'
import { runChecks, type RepositoryCheck } from './runner.ts'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'

describe('runChecks', () => {
  it('一つの検査が例外を投げても全結果を集める', async () => {
    const checks: RepositoryCheck[] = [
      {
        name: 'first',
        groups: ['standard'],
        run: async () => {
          throw new Error('first failed before returning findings')
        },
      },
      {
        name: 'second',
        groups: ['standard'],
        run: async () => ({ ok: false, lines: ['second reported its finding'] }),
      },
    ]

    const results = await runChecks(checks, createWorkspaceSnapshot(process.cwd()))

    expect(results).toHaveLength(2)
    expect(results[0]).toEqual({
      name: 'first',
      ok: false,
      lines: ['first: first failed before returning findings'],
    })
    expect(results[1]).toEqual({
      name: 'second',
      ok: false,
      lines: ['second reported its finding'],
    })
  })
})
