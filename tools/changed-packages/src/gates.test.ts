import { describe, expect, it } from 'bun:test'
import type { GoPackage } from './changed-packages.ts'
import { gatesFor } from './gates.ts'
import { planChange } from './plan.ts'
import { miseChange } from './mise-change.ts'

const PACKAGES: GoPackage[] = [
  { importPath: 'app/domain', dir: 'backend/domain', deps: [] },
  { importPath: 'app/jobs', dir: 'backend/jobs', deps: [] },
]

describe('gatesFor', () => {
  it('監査タスク自体を変えたときも監査を選ぶ', () => {
    const change = miseChange(
      '[tasks.audit-dependencies]\nrun = "old"',
      '[tasks.audit-dependencies]\nrun = "new"',
    )
    const gates = gatesFor(planChange(['mise.toml'], PACKAGES, change), PACKAGES)
    expect(gates.some((gate) => gate.command.join(' ') === 'mise run audit-dependencies')).toBe(
      true,
    )
  })
  it('依存入力が変わったときだけ既存の脆弱性監査を選ぶ', () => {
    const inputs = [
      'go.mod',
      'go.sum',
      'frontend/package.json',
      'frontend/bun.lock',
      'tools/package.json',
      'tools/bun.lock',
      'osv-scanner.toml',
    ]
    for (const path of inputs) {
      const gates = gatesFor(planChange([path], PACKAGES), PACKAGES)
      expect(gates.some((gate) => gate.command.join(' ') === 'mise run audit-dependencies')).toBe(
        true,
      )
    }
    for (const path of [
      'backend/jobs/run.go',
      'frontend/src/App.tsx',
      'docs/development/testing.md',
    ]) {
      const gates = gatesFor(planChange([path], PACKAGES), PACKAGES)
      expect(gates.some((gate) => gate.command.join(' ') === 'mise run audit-dependencies')).toBe(
        false,
      )
    }
  })
  /**
   * The race detector runs on a schedule against main (wi-47187), and on
   * request while building concurrent code. The selected gates use the
   * ordinary build, which is faster and shares the ordinary build cache.
   */
  it('lints and tests only the selected packages without the race detector', () => {
    expect(gatesFor(planChange(['backend/jobs/run.go'], PACKAGES), PACKAGES)).toEqual([
      { layer: 'check', command: ['mise', 'run', 'check'] },
      { layer: 'go', command: ['golangci-lint', 'run', './backend/jobs'] },
      {
        layer: 'go',
        command: [
          'bun',
          'run',
          'tools/verification/src/main.ts',
          '--report',
          '.verification/test-go-selected.json',
          '--',
          'app/jobs',
        ],
      },
    ])
  })

  it('falls back to the whole Go gates when the change may reach every package', () => {
    const commands = gatesFor(planChange(['infra/schema/postgres.sql'], PACKAGES), PACKAGES)
      .filter((gate) => gate.layer === 'go')
      .map((gate) => gate.command.join(' '))
    expect(commands).toEqual(['mise run lint-go', 'mise run test-go'])
  })

  it('runs the UI and tools suites whole when their layer is selected', () => {
    const commands = gatesFor(planChange(['mise.toml'], PACKAGES), PACKAGES).map((gate) =>
      gate.command.join(' '),
    )
    expect(commands).toContain('mise run verify-ui')
    expect(commands).toContain('mise run test-tools')
    expect(commands).toContain('mise run typecheck-tools')
    expect(commands).toContain('mise run lint-tools')
  })

  it('runs nothing for a plan that selects nothing', () => {
    expect(gatesFor(planChange([], PACKAGES), PACKAGES)).toEqual([])
  })
})
