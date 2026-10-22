import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'
import { selectChecks } from './registry.ts'
import { taskClosure } from './verification-tasks.ts'

const root = resolve(import.meta.dir, '../../..')
const config = Bun.TOML.parse(await Bun.file(resolve(root, 'mise.toml')).text()) as {
  tasks: Record<string, { run?: string }>
}
const workflow = Bun.YAML.parse(
  await Bun.file(resolve(root, '.github/workflows/idmagic-ci.yaml')).text(),
) as {
  jobs: Record<
    string,
    {
      needs?: string[]
      if?: string
      env?: Record<string, string>
      steps?: { run?: string; if?: string; with?: Record<string, string | number | boolean> }[]
    }
  >
}

describe('CI の全体結果', () => {
  it('四群を待ち、先行する失敗があっても集約する', () => {
    const gate = workflow.jobs.verify!
    expect(gate.needs?.toSorted()).toEqual(['go-lint', 'go-test', 'spec', 'ui'])
    expect(gate.if).toContain('always()')
    const run = gate.steps?.find((step) => step.run?.includes('check-ci-results'))?.run ?? ''
    for (const job of gate.needs ?? []) expect(run).toContain(`needs.${job}.result`)
  })

  it('成功だけを受理し、失敗、skip、取消、不足、過剰を拒否する', async () => {
    const script = config.tasks?.['check-ci-results']?.run
    expect(typeof script).toBe('string')
    const execute = async (results: string[]) => {
      const child = Bun.spawn(['sh', '-c', script!, 'check-ci-results', ...results], {
        stdout: 'ignore',
        stderr: 'ignore',
      })
      return child.exited
    }
    expect(await execute(Array(4).fill('success'))).toBe(0)
    for (let index = 0; index < 4; index++) {
      for (const state of ['failure', 'cancelled', 'skipped', '', 'unknown']) {
        const results = Array(4).fill('success')
        results[index] = state
        expect(await execute(results)).not.toBe(0)
      }
    }
    for (const size of [0, 1, 3, 5]) expect(await execute(Array(size).fill('success'))).not.toBe(0)
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
