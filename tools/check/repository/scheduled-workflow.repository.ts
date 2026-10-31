import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'

const root = resolve(import.meta.dir, '../../..')

type Workflow = {
  on: Record<string, unknown>
  jobs: Record<string, { steps?: { run?: string; uses?: string }[] }>
}

async function parse(name: string): Promise<Workflow> {
  return Bun.YAML.parse(await Bun.file(resolve(root, '.github/workflows', name)).text()) as Workflow
}

const steps = (workflow: Workflow) => Object.values(workflow.jobs).flatMap((job) => job.steps ?? [])

describe('重い検査の定期実行', () => {
  it('push と PR からは起動せず、定期と手動だけで起動する', async () => {
    const parsed = await parse('idmagic-scheduled.yaml')
    expect(Object.keys(parsed.on).sort()).toEqual(['schedule', 'workflow_dispatch'])
  })

  it('Envoy の試験と CodeQL を定期実行だけが持つ', async () => {
    const scheduled = steps(await parse('idmagic-scheduled.yaml'))
    expect(scheduled.some((step) => step.run === 'mise run test-envoy-mcp-gateway')).toBe(true)
    expect(scheduled.some((step) => step.uses?.startsWith('github/codeql-action/analyze'))).toBe(
      true,
    )
    const ci = steps(await parse('idmagic-ci.yaml'))
    expect(ci.some((step) => step.run?.includes('test-envoy-mcp-gateway'))).toBe(false)
    expect(ci.some((step) => step.uses?.startsWith('github/codeql-action/analyze'))).toBe(false)
  })
})
