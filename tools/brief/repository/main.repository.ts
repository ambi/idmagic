import { describe, expect, it } from 'bun:test'
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'

const root = resolve(import.meta.dir, '../../..')
function brief(...args: string[]) {
  const result = Bun.spawnSync(['bun', 'tools/brief/src/main.ts', ...args], { cwd: root })
  return {
    status: result.exitCode,
    output: result.stdout.toString(),
    error: result.stderr.toString(),
  }
}

describe('brief CLI', () => {
  it('retains direct normative declarations and TypeSpec symbols without claiming verification', () => {
    const fixture = mkdtempSync(resolve(tmpdir(), 'idmagic-brief-'))
    try {
      for (const path of [
        'work-items/active',
        'work-items/done',
        'docs/modules/demo',
        'spec/modules/demo',
        'backend/demo',
      ])
        mkdirSync(resolve(fixture, path), { recursive: true })
      writeFileSync(
        resolve(fixture, 'work-items/active/wi-1-demo.md'),
        '---\nchange_kind: feature\naffected_spec:\n  - {path: docs/modules/demo/README.md, requirement: REQ-DEMO-001}\n  - {path: spec/modules/demo/models.tsp, symbol: Demo.Task}\n---\n# デモ\n',
      )
      writeFileSync(
        resolve(fixture, 'docs/modules/demo/README.md'),
        '# デモ\n\n## REQ-DEMO-001 開始する\n\n- 開始すると保存する。\n',
      )
      writeFileSync(
        resolve(fixture, 'spec/modules/demo/models.tsp'),
        'namespace Demo;\nmodel Task {}\n',
      )
      writeFileSync(
        resolve(fixture, 'backend/demo/start_test.go'),
        '// REQ-DEMO-001 に言及するだけのファイル\n',
      )
      writeFileSync(resolve(fixture, 'mise.toml'), '')
      const result = Bun.spawnSync(['bun', 'tools/brief/src/main.ts', 'wi-1'], {
        cwd: root,
        env: { ...process.env, SPEC_WORKSPACE_ROOT: fixture },
      })
      const output = result.stdout.toString()
      expect(result.exitCode).toBe(0)
      expect(output).toContain('## REQ-DEMO-001 開始する')
      expect(output).toContain('spec/modules/demo/models.tsp:2')
      expect(output).toContain('Named by tests: backend/demo/start_test.go')
      expect(output).toContain('not proof that the behavior is reachable')
      expect(output).not.toContain('## Recent history')
    } finally {
      rmSync(fixture, { recursive: true, force: true })
    }
  })
  it('passes explicit targets through the mise entry and works without affected_spec', () => {
    const result = Bun.spawnSync(
      ['mise', 'run', 'brief', '--', 'wi-69747', '--target', 'tools/brief'],
      { cwd: root },
    )
    const output = result.stdout.toString()
    expect(result.exitCode).toBe(0)
    expect(output).toContain('docs/development/coding-style.md#使い方')
    expect(output).toContain('test-tools-file')
    expect(output).toContain('origin:')
    expect(output).not.toContain('nothing to resolve')
    expect(output).not.toContain('## Recent history')
  })

  it('keeps history opt-in and reports the seed', () => {
    const result = brief('wi-69747', '--target', 'tools/brief', '--include', 'history')
    expect(result.status).toBe(0)
    expect(result.output).toContain('## Recent history')
    expect(result.output).toContain('tools/brief')
  })

  it('rejects unknown options and escaped or missing targets', () => {
    for (const args of [
      ['--unknown'],
      ['--target'],
      ['--target', '../outside'],
      ['--target', 'tools/not-a-real-tool'],
      ['--concern', 'mystery'],
      ['--include', 'everything'],
    ]) {
      const result = brief('wi-69747', ...args)
      expect(result.status).toBe(2)
      expect(result.error).not.toBe('')
    }
  })
})
