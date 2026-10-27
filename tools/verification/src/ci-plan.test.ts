import { describe, expect, it } from 'bun:test'
import { evaluateCIResults, planCI } from './ci-plan.ts'

function selected(paths: string[] | null) {
  return Object.entries(planCI(paths).groups)
    .filter(([, decision]) => decision.selected)
    .map(([group]) => group)
}

describe('CI の検査計画', () => {
  it.each([
    [['work-items/active/wi-1.md', 'docs/development/testing.md'], []],
    [['frontend/src/app.tsx'], ['ui', 'browser']],
    [['frontend/bun.lock'], ['ui', 'browser']],
    [['backend/tenancy/user.go'], ['go', 'browser']],
    [['go.mod'], ['go', 'ui', 'browser', 'schema']],
    [['go.sum'], ['go', 'ui', 'browser', 'schema']],
    [['backend/shared/storage/db_postgres/schema_test.go'], ['go', 'ui', 'browser', 'schema']],
    [['infra/schema/postgres.sql'], ['go', 'ui', 'browser', 'schema']],
    [['backend/cmd/idmagic-dev-infra/main.go'], ['go', 'ui', 'browser', 'schema']],
    [['spec/main.tsp'], ['go', 'ui', 'browser', 'schema']],
    [['docs/modules/tenancy/features/user.md'], ['go', 'ui', 'browser', 'schema']],
    [['docs/development/example.json'], ['go', 'ui', 'browser', 'schema']],
    [['tools/verification/src/main.ts'], ['go', 'ui', 'browser', 'schema']],
    [['mise.toml'], ['go', 'ui', 'browser', 'schema']],
    [['.github/workflows/idmagic-ci.yaml'], ['go', 'ui', 'browser', 'schema']],
    [['unknown/file.md'], ['go', 'ui', 'browser', 'schema']],
    [
      ['frontend/src/app.tsx', 'backend/main.go'],
      ['go', 'ui', 'browser'],
    ],
    [[], ['go', 'ui', 'browser', 'schema']],
    [null, ['go', 'ui', 'browser', 'schema']],
  ] as [string[] | null, string[]][])('%j の選択を %j にする', (paths, groups) => {
    expect(selected(paths)).toEqual(groups)
    for (const decision of Object.values(planCI(paths).groups)) {
      expect(decision.reasons.length).toBeGreaterThan(0)
    }
  })

  it('入力順序と重複に依存せず、混在時に選択を減らさない', () => {
    const paths = ['frontend/a.ts', 'backend/a.go', 'work-items/a.md']
    expect(planCI(paths)).toEqual(planCI([...paths.toReversed(), ...paths]))
    for (const path of paths) {
      for (const group of selected([path])) expect(selected(paths)).toContain(group)
    }
  })
})

function needs(paths: string[] | null) {
  const plan = planCI(paths)
  return {
    plan: { result: 'success', outputs: { plan: JSON.stringify(plan) } },
    spec: { result: 'success' },
    'go-lint': { result: plan.groups.go.selected ? 'success' : 'skipped' },
    'go-test': { result: plan.groups.go.selected ? 'success' : 'skipped' },
    ui: { result: plan.groups.ui.selected ? 'success' : 'skipped' },
    'ui-e2e': { result: plan.groups.browser.selected ? 'success' : 'skipped' },
    'schema-convergence': { result: plan.groups.schema.selected ? 'success' : 'skipped' },
  }
}

describe('CI の結果集約', () => {
  it('選択した群の成功と、理由がある対象外を区別する', () => {
    const result = evaluateCIResults(needs(['docs/development/testing.md']))
    expect(result.success).toBe(true)
    expect(result.lines.join('\n')).toContain('not selected')
    expect(result.lines.join('\n')).toContain('docs/development/testing.md')
    expect(evaluateCIResults(needs(null)).success).toBe(true)
  })

  it('選択した全ジョブの失敗、skip、取消、結果欠落を拒否する', () => {
    const successful = needs(null)
    for (const job of Object.keys(successful)) {
      for (const state of ['failure', 'skipped', 'cancelled', '', 'unknown']) {
        const input = structuredClone(successful)
        input[job as keyof typeof input].result = state
        expect(evaluateCIResults(input).success).toBe(false)
      }
      const input: Record<string, unknown> = structuredClone(successful)
      delete input[job]
      expect(evaluateCIResults(input).success).toBe(false)
    }
  })

  it('対象外の群の失敗も隠さず、欠落と不正な計画を拒否する', () => {
    const input = needs(['work-items/a.md'])
    input['go-test'].result = 'failure'
    expect(evaluateCIResults(input).success).toBe(false)
    for (const plan of ['', '{}', 'null', '{', JSON.stringify({ groups: {} })]) {
      const input = needs(null)
      input.plan.outputs.plan = plan
      expect(evaluateCIResults(input).success).toBe(false)
    }
    expect(evaluateCIResults(null).success).toBe(false)
    const invalid = planCI(null)
    invalid.groups.go.reasons = []
    const malformed = needs(null)
    malformed.plan.outputs.plan = JSON.stringify(invalid)
    expect(evaluateCIResults(malformed).success).toBe(false)
  })
})
