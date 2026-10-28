import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'
import { selectChecks } from '../src/registry.ts'
import { taskClosure } from '../src/verification-tasks.ts'
import { planCI } from '../../verification/src/ci-plan.ts'

const root = resolve(import.meta.dir, '../../..')
const config = Bun.TOML.parse(await Bun.file(resolve(root, 'mise.toml')).text()) as {
  tasks: Record<string, { run?: string }>
}
const workflow = Bun.YAML.parse(
  await Bun.file(resolve(root, '.github/workflows/idmagic-ci.yaml')).text(),
) as {
  on: Record<string, Record<string, unknown> | null>
  jobs: Record<
    string,
    {
      needs?: string[]
      name?: string
      outputs?: Record<string, string>
      if?: string
      env?: Record<string, string>
      steps?: {
        id?: string
        run?: string
        if?: string
        env?: Record<string, string>
        with?: Record<string, string | number | boolean>
      }[]
    }
  >
}

describe('CI の全体結果', () => {
  it('計画と全検査を待ち、先行する失敗があっても既存名で集約する', () => {
    const gate = workflow.jobs.verify!
    expect(gate.needs?.toSorted()).toEqual([
      'go-lint',
      'go-test',
      'plan',
      'schema-convergence',
      'spec',
      'ui',
      'ui-e2e',
    ])
    expect(gate.name).toBe('Verify specification, backend, and UI')
    expect(gate.if).toContain('always()')
    const step = gate.steps!.find((step) => step.run?.includes('check-ci-results'))!
    expect(step.env?.CI_NEEDS).toBe(`\${{ toJSON(needs) }}`)
  })

  it('mise の正式入口が文書変更を除外し、選択した skip と計画の欠落を拒否する', async () => {
    const script = config.tasks?.['check-ci-results']?.run
    expect(typeof script).toBe('string')
    const execute = async (needs: unknown) => {
      const child = Bun.spawn(['sh', '-c', script!], {
        cwd: resolve(root, 'tools'),
        env: { ...process.env, CI_NEEDS: JSON.stringify(needs), GITHUB_STEP_SUMMARY: '' },
        stdout: 'pipe',
        stderr: 'pipe',
      })
      const output = await new Response(child.stdout).text()
      return { code: await child.exited, output }
    }
    for (const paths of [
      ['work-items/a.md'],
      ['frontend/src/app.tsx'],
      ['backend/tenancy/user.go'],
      ['spec/main.tsp'],
    ]) {
      const plan = planCI(paths)
      const needs: Record<string, unknown> = {
        plan: { result: 'success', outputs: { plan: JSON.stringify(plan) } },
        spec: { result: 'success' },
      }
      for (const job of ['go-lint', 'go-test', 'ui', 'ui-e2e', 'schema-convergence']) {
        const config = workflow.jobs[job]!
        expect(config.needs).toEqual(['plan'])
        const group = {
          'go-lint': 'go',
          'go-test': 'go',
          ui: 'ui',
          'ui-e2e': 'browser',
          'schema-convergence': 'schema',
        }[job]!
        expect(config.if).toBe(`\${{ needs.plan.outputs.${group} == 'true' }}`)
        const selected = plan.groups[group as keyof typeof plan.groups].selected
        needs[job] = { result: selected ? 'success' : 'skipped' }
      }
      expect((await execute(needs)).code).toBe(0)
      const input = structuredClone(needs)
      input.spec = { result: 'skipped' }
      expect((await execute(input)).code).toBe(1)
      if (plan.groups.browser.selected) {
        input.spec = { result: 'success' }
        input['ui-e2e'] = { result: 'skipped' }
        expect((await execute(input)).code).toBe(1)
      } else expect((await execute(needs)).output).toContain('not selected')
    }
    expect((await execute({})).code).toBe(1)
  })

  it('計画だけに差分条件を置き、イベントの base と head を環境変数で渡す', () => {
    const planner = workflow.jobs.plan!
    expect(planner.if).toBeUndefined()
    const step = planner.steps!.find((step) => step.run?.includes('mise run plan-ci'))!
    expect(step.id).toBe('selection')
    expect(step.env?.CI_BASE).toContain('pull_request.base.sha')
    expect(step.env?.CI_BASE).toContain('github.event.before')
    expect(step.env?.CI_HEAD).toContain('pull_request.head.sha')
    expect(step.env?.CI_HEAD).toContain('github.sha')
    expect(step.run).toBe('mise run plan-ci -- "$CI_BASE" "$CI_HEAD"')
    expect(planner.steps?.some((step) => step.with?.['fetch-depth'] === 0)).toBe(true)
    for (const group of ['plan', 'go', 'ui', 'browser', 'schema']) {
      expect(planner.outputs?.[group]).toBe(`\${{ steps.selection.outputs.${group} }}`)
    }
    expect(workflow.jobs.spec!.if).toBeUndefined()
    for (const event of ['push', 'pull_request']) {
      expect(workflow.on[event]?.paths).toBeUndefined()
      expect(workflow.on[event]?.['paths-ignore']).toBeUndefined()
    }
    for (const job of ['sast', 'dependency-audit']) {
      expect(workflow.jobs[job]!.if).toBeUndefined()
      expect(workflow.jobs[job]!.needs).toBeUndefined()
    }
  })

  it('公開 API の互換性は全体検査へ一度だけ接続する', () => {
    expect(
      selectChecks(['all']).filter((check) => check.name === 'published-api-compat'),
    ).toHaveLength(1)
    const calls = Object.values(workflow.jobs)
      .flatMap((job) => job.steps ?? [])
      .map((step) => step.run ?? '')
    expect(calls.some((run) => run.includes('mise run check-ci-spec --'))).toBe(true)
    expect(taskClosure(config.tasks, 'check-ci-spec').has('check-repository')).toBe(true)
    expect(calls.some((run) => run.includes('mise run check-published-api-compat'))).toBe(false)
  })

  it('Go の参照検査を通常テストの後に一度だけ実行する', () => {
    const steps = workflow.jobs['go-test']!.steps ?? []
    const tests = steps.findIndex((step) => step.run === 'mise run test-go')
    for (const task of [
      'check-config-reference',
      'check-route-reference',
      'check-gateway-routes',
    ]) {
      const references = steps.findIndex((step) => step.run?.includes(`mise run ${task}`))
      expect(references).toBeGreaterThan(tests)
      expect(taskClosure(config.tasks, 'check-ci-spec').has(task)).toBe(false)
      const calls = Object.values(workflow.jobs).flatMap((job) => job.steps ?? [])
      expect(calls.filter((step) => step.run?.includes(`mise run ${task}`))).toHaveLength(1)
    }
  })

  it('DB 必須の通常テストを実行し、その実行の証拠を失敗時も保存する', () => {
    const job = workflow.jobs['go-test']!
    expect(job.env?.VERIFICATION_REQUIRE_DB).toBe('1')
    expect(job.env?.VERIFICATION_BASE_REVISION).toContain('pull_request.base.sha')
    const steps = job.steps ?? []
    expect(steps.some((step) => step.with?.['fetch-depth'] === 0)).toBe(true)
    expect(steps.some((step) => step.run === 'mise run test-go')).toBe(true)
    const calls = Object.values(workflow.jobs)
      .flatMap((group) => group.steps ?? [])
      .map((step) => step.run ?? '')
    expect(calls.some((run) => run.includes('test-go-race'))).toBe(false)
    const report = steps.find((step) => step.run?.includes('report-verification'))!
    expect(report.run).toBe('mise run report-verification -- .verification/test-go.json')
    expect(report.if).toContain('!cancelled()')
    const artifact = steps.find((step) => step.with?.name === 'verification-evidence')!
    expect(artifact.if).toContain('!cancelled()')
    expect(artifact.with?.path).toBe('.verification/')
    expect(artifact.with?.['include-hidden-files']).toBe(true)
  })
})
