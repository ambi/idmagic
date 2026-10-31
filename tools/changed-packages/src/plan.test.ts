import { describe, expect, it } from 'bun:test'
import type { GoPackage } from './changed-packages.ts'
import { miseChange } from './mise-change.ts'
import { intersectPlans, planChange } from './plan.ts'

const PACKAGES: GoPackage[] = [
  { importPath: 'app/domain', dir: 'backend/domain', deps: [] },
  { importPath: 'app/handlers', dir: 'backend/handlers', deps: ['app/domain'] },
  { importPath: 'app/jobs', dir: 'backend/jobs', deps: [] },
]

const selected = (plan: ReturnType<typeof planChange>) => ({
  check: plan.check.selected,
  go: plan.go.packages,
  ui: plan.ui.selected,
  tools: plan.tools.selected,
})

describe('planChange', () => {
  it('selects only the layers a change touches', () => {
    expect(selected(planChange(['backend/jobs/run.go'], PACKAGES))).toEqual({
      check: true,
      go: ['app/jobs'],
      ui: false,
      tools: false,
    })
    expect(
      selected(planChange(['frontend/src/App.tsx', 'tools/check/src/x.ts'], PACKAGES)),
    ).toEqual({ check: true, go: [], ui: true, tools: true })
  })

  it('selects every layer for a path no layer owns', () => {
    const plan = planChange(['mise.toml'], PACKAGES)
    expect(selected(plan)).toEqual({
      check: true,
      go: ['app/domain', 'app/handlers', 'app/jobs'],
      ui: true,
      tools: true,
    })
    expect(plan.ui.reasons).toEqual(['mise.toml is outside every layer'])
  })

  it('selects nothing for an empty change', () => {
    expect(selected(planChange([], PACKAGES))).toEqual({
      check: false,
      go: [],
      ui: false,
      tools: false,
    })
  })

  it('mise のタスク変更と通常のファイル変更の影響を合成する', () => {
    const before = '[tasks.deploy]\nrun = "old"'
    const after = '[tasks.deploy]\nrun = "new"'
    const change = miseChange(before, after)
    expect(selected(planChange(['mise.toml'], PACKAGES, change))).toEqual({
      check: true,
      go: [],
      ui: false,
      tools: false,
    })
    expect(
      selected(
        planChange(['mise.toml', 'backend/jobs/run.go', 'frontend/src/App.tsx'], PACKAGES, change),
      ),
    ).toEqual({
      check: true,
      go: ['app/jobs'],
      ui: true,
      tools: false,
    })
  })

  it('Go のタスク変更では全パッケージを選ぶ', () => {
    const change = miseChange('[tasks.test-go]\nrun = "old"', '[tasks.test-go]\nrun = "new"')
    const plan = planChange(['mise.toml'], PACKAGES, change)
    expect(plan.go.all).toBe(true)
    expect(selected(plan)).toEqual({
      check: true,
      go: ['app/domain', 'app/handlers', 'app/jobs'],
      ui: false,
      tools: false,
    })
  })

  it('mise の限定判定があっても別の未知のパスは全層へ戻す', () => {
    const change = miseChange('', '')
    expect(selected(planChange(['mise.toml', '.unknown'], PACKAGES, change))).toEqual({
      check: true,
      go: ['app/domain', 'app/handlers', 'app/jobs'],
      ui: true,
      tools: true,
    })
  })
})

describe('intersectPlans', () => {
  it('依存入力の変更が両側で重なる統合だけ監査を選ぶ', () => {
    const dependency = planChange(['frontend/bun.lock'], PACKAGES)
    const sourceOnly = planChange(['frontend/src/App.tsx'], PACKAGES)
    expect(intersectPlans(dependency, dependency).audit.selected).toBe(true)
    expect(intersectPlans(dependency, sourceOnly).audit.selected).toBe(false)
    expect(intersectPlans(sourceOnly, dependency).audit.selected).toBe(false)
  })
  it('runs only what both sides of a rebase can affect', () => {
    const source = planChange(['backend/domain/user.go', 'frontend/src/App.tsx'], PACKAGES)
    const incoming = planChange(['backend/jobs/run.go', 'backend/handlers/h.go'], PACKAGES)
    expect(selected(intersectPlans(source, incoming))).toEqual({
      check: true,
      go: ['app/handlers'],
      ui: false,
      tools: false,
    })
  })

  it('keeps the other side when one side may reach every package', () => {
    const source = planChange(['backend/jobs/run.go'], PACKAGES)
    const incoming = planChange(['infra/schema/postgres.sql'], PACKAGES)
    expect(intersectPlans(source, incoming).go.packages).toEqual(['app/jobs'])
  })

  it('runs only the repository checks when one side changes documents alone', () => {
    const source = planChange(['backend/jobs/run.go', 'frontend/src/App.tsx'], PACKAGES)
    const incoming = planChange(['docs/runbooks/backup.md'], PACKAGES)
    expect(selected(intersectPlans(source, incoming))).toEqual({
      check: true,
      go: [],
      ui: false,
      tools: false,
    })
  })
})
