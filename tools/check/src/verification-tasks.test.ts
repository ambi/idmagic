import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'
import { selectChecks } from './registry.ts'
import { directTasks, parseMiseTasks, taskClosure } from './verification-tasks.ts'

describe('実際の仕様ゲートと全体ゲートの包含関係', () => {
  it('生成検査の入口を共有し、仕様の registry 集合を全体の部分集合にする', async () => {
    const tasks = parseMiseTasks(
      await Bun.file(resolve(import.meta.dir, '../../../mise.toml')).text(),
    )
    expect(directTasks(tasks, 'check')).toContain('check-spec-inputs')
    expect(directTasks(tasks, 'check-spec')).toContain('check-spec-inputs')
    expect(directTasks(tasks, 'check-spec-inputs')).toEqual([
      'compile-spec',
      'check-generated-contract',
      'check-site',
    ])
    for (const entry of ['check', 'check-spec']) {
      expect(directTasks(tasks, entry)).not.toContain('check-generated-contract')
      expect(directTasks(tasks, entry)).not.toContain('check-site')
    }
    expect(tasks['check-spec']?.run).toBe('bun run check/src/runner.ts spec')
    const specification = selectChecks(['spec']).map((check) => check.name)
    expect(specification).toEqual([
      'executable-examples',
      'documents',
      'specification-rules',
      'document-layout',
    ])
    const all = selectChecks(['all']).map((check) => check.name)
    for (const name of specification) expect(all).toContain(name)
  })
})

const CONFIG = `
[tasks.compile-spec]
run = "bunx tsp compile"

[tasks.check-spec]
depends = ["compile-spec"]
run = ["mise run check-generated-contract", "cd tools && bun run check.ts"]

[tasks.check-generated-contract]
run = "bun run generate-contract/src/main.ts --check"

[tasks.check-links]
run = "bun run check-links.ts"

[tasks.check]
run = """
mise run -c \\
  check-spec ::: \\
  check-links
"""

[tasks.lint-go]
run = "golangci-lint run ./..."

[tasks.verify]
run = "mise run -c check-spec ::: check-links ::: lint-go"

[tasks.test-go-package]
run = "echo 'usage: mise run test-go-package -- <package>'; go test -race \\"$1\\""
`

describe('directTasks', () => {
  const tasks = parseMiseTasks(CONFIG)

  it('reads the tasks a dependency list declares', () => {
    expect(directTasks(tasks, 'check-spec')).toContain('compile-spec')
  })

  it('reads the tasks a run command invokes', () => {
    expect(directTasks(tasks, 'check-spec')).toContain('check-generated-contract')
  })

  /**
   * The continue-on-error aggregate writes its members as `:::`-separated
   * arguments across continued lines. Reading only `depends` would report an
   * aggregate with no members, and every consumer of this list — the timing
   * table, the primary-use-case task reachability check — would then be
   * silently empty rather than wrong in a visible way.
   */
  it('reads every member of a continued continue-on-error invocation', () => {
    expect(directTasks(tasks, 'check')).toEqual(['check-spec', 'check-links'])
  })

  it('keeps the declared order of a single-line invocation', () => {
    expect(directTasks(tasks, 'verify')).toEqual(['check-spec', 'check-links', 'lint-go'])
  })

  /** Arguments after `--` are values, not task names. */
  it('does not read the argument placeholder of a usage message as a task', () => {
    expect(directTasks(tasks, 'test-go-package')).toEqual(['test-go-package'])
  })

  it('reports no members for a leaf task', () => {
    expect(directTasks(tasks, 'lint-go')).toEqual([])
  })

  it('reports no members for a task the configuration does not declare', () => {
    expect(directTasks(tasks, 'absent')).toEqual([])
  })
})

describe('taskClosure', () => {
  const tasks = parseMiseTasks(CONFIG)

  it('includes the entry point and everything it reaches', () => {
    expect([...taskClosure(tasks, 'verify')].sort()).toEqual([
      'check-generated-contract',
      'check-links',
      'check-spec',
      'compile-spec',
      'lint-go',
      'verify',
    ])
  })

  it('terminates on a self-referencing task', () => {
    expect([...taskClosure(tasks, 'test-go-package')]).toEqual(['test-go-package'])
  })
})
