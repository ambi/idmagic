import { describe, expect, it } from 'bun:test'
import { existsSync } from 'node:fs'
import { resolve } from 'node:path'

const path = resolve(import.meta.dir, '../../..', '.github/workflows/idmagic-race.yaml')

type Workflow = {
  on: Record<string, unknown>
  permissions?: Record<string, string>
  jobs: Record<
    string,
    {
      env?: Record<string, string>
      steps?: { run?: string; if?: string; with?: Record<string, string | number | boolean> }[]
    }
  >
}

async function workflow(): Promise<Workflow> {
  expect(existsSync(path)).toBe(true)
  return Bun.YAML.parse(await Bun.file(path).text()) as Workflow
}

describe('main に対する Go race 検査の定期実行', () => {
  it('push と PR からは起動せず、定期と手動だけで起動する', async () => {
    const parsed = await workflow()
    expect(Object.keys(parsed.on).sort()).toEqual(['schedule', 'workflow_dispatch'])
    expect(parsed.permissions).toEqual({ contents: 'read' })
  })

  it('DB 必須で全体の race 検査を結果キャッシュなしで実行し、その証拠を失敗時も保存する', async () => {
    const jobs = Object.values((await workflow()).jobs)
    expect(jobs).toHaveLength(1)
    const job = jobs[0]!
    expect(job.env?.VERIFICATION_REQUIRE_DB).toBe('1')
    expect(job.env?.GOFLAGS).toBe('-count=1')
    const steps = job.steps ?? []
    expect(steps.some((step) => step.run === 'mise run test-go-race')).toBe(true)
    const report = steps.find((step) => step.run?.includes('report-verification'))!
    expect(report.run).toBe('mise run report-verification -- .verification/test-go-race.json')
    expect(report.if).toContain('!cancelled()')
    const artifact = steps.find((step) => step.with?.path === '.verification/')!
    expect(artifact.if).toContain('!cancelled()')
    expect(artifact.with?.['include-hidden-files']).toBe(true)
  })
})
