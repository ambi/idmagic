import { describe, expect, it } from 'bun:test'
import { directTasks, parseMiseTasks, taskClosure } from './verification-tasks.ts'

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
