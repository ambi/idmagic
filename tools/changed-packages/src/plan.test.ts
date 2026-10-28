import { describe, expect, it } from 'bun:test'
import type { GoPackage } from './changed-packages.ts'
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
})

describe('intersectPlans', () => {
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
