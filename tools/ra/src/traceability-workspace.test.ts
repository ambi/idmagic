import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, describe, expect, it } from 'bun:test'
import type { TraceabilityReport } from './traceability.ts'

const HERE = dirname(fileURLToPath(import.meta.url))
const SCRIPT = resolve(HERE, 'traceability-workspace.ts')

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

/**
 * A workspace whose only module is declared by a nested ledger, with a prose
 * design record beside it that carries no ledger frontmatter (ADR-143).
 */
async function workspace(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'ra-traceability-workspace-'))
  cleanup.push(root)
  await mkdir(join(root, 'spec'), { recursive: true })
  await mkdir(join(root, 'backend'), { recursive: true })
  await mkdir(join(root, 'verification'), { recursive: true })
  await writeFile(join(root, 'spec', 'scl.yaml'), 'system: demo\nspec_version: "3.0"\n')
  await writeFile(join(root, 'justfile'), 'test-go:\n    true\n')
  await writeFile(join(root, 'architecture.yaml'), 'context: repo\n')
  await writeFile(
    join(root, 'backend', 'architecture.yaml'),
    'context: repo\nmodules:\n  backend:\n    path: .\n',
  )
  await writeFile(join(root, 'ARCHITECTURE.md'), '# demo\n\n## Overview\n\nProse only.\n')
  await writeFile(
    join(root, 'verification', 'manifest.yaml'),
    [
      'version: 1',
      'policies: []',
      'realizations:',
      '  - id: backend-flows',
      '    module: backend',
      '    targets: []',
      'checks: []',
      'baselines: []',
      '',
    ].join('\n'),
  )
  return root
}

async function runTraceability(root: string): Promise<TraceabilityReport> {
  const proc = Bun.spawn(['bun', 'run', SCRIPT, '--json', '--revision=rev'], {
    env: { ...process.env, RA_WORKSPACE_ROOT: root },
    stdout: 'pipe',
    stderr: 'pipe',
  })
  const stdout = await new Response(proc.stdout).text()
  await proc.exited
  return JSON.parse(stdout) as TraceabilityReport
}

describe('traceability workspace', () => {
  it('resolves realization modules from the architecture ledgers, not the design record', async () => {
    const report = await runTraceability(await workspace())

    expect(report.findings).toEqual([])
    expect(report.passed).toBe(true)
  })

  it('reports a realization module that no ledger declares', async () => {
    const root = await workspace()
    await writeFile(join(root, 'backend', 'architecture.yaml'), 'context: repo\nmodules: {}\n')

    const report = await runTraceability(root)

    expect(report.findings).toEqual([
      {
        code: 'unknown_module',
        target: 'backend-flows',
        detail: "unknown Architecture module 'backend'",
      },
    ])
  })
})
