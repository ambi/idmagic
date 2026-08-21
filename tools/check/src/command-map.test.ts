import { describe, expect, it } from 'bun:test'
import { invokedTasks, taskNames, verifyCommandMap } from './command-map.ts'

const miseToml = [
  '[tools]',
  'go = "1.26.5"',
  '',
  '[tasks.verify]',
  'depends = ["check", "test-go"]',
  '',
  '[tasks.test-go-package]',
  'run = "go test $1"',
  '',
  '[tasks.check]',
  'run = "bun run check.ts"',
  '',
  '[tasks.test-go]',
  'run = "go test ./..."',
].join('\n')

describe('taskNames', () => {
  it('reads task names without treating tools as tasks', () => {
    expect([...taskNames(miseToml)].sort()).toEqual([
      'check',
      'test-go',
      'test-go-package',
      'verify',
    ])
  })
})

describe('invokedTasks', () => {
  it('finds tasks called from a workflow step', () => {
    expect(
      invokedTasks('run: |\n  mise run check\n  mise run test-go-package -- ./backend\n'),
    ).toEqual(['check', 'test-go-package'])
  })
})

describe('verifyCommandMap', () => {
  it('accepts a workflow that only calls declared tasks', () => {
    const workflow = { file: 'ci.yaml', source: 'run: mise run check\nrun: mise run verify\n' }
    expect(verifyCommandMap(miseToml, [workflow])).toEqual([])
  })

  it('reports a task mise.toml no longer declares', () => {
    const workflow = {
      file: 'ci.yaml',
      source: 'run: |\n  mise run check\n  mise run traceability-strict\n',
    }
    expect(verifyCommandMap(miseToml, [workflow])).toEqual([
      { file: 'ci.yaml', task: 'traceability-strict' },
    ])
  })

  it('reports each missing task once', () => {
    const workflows = [
      { file: 'a.yaml', source: 'run: mise run gone\n' },
      { file: 'b.yaml', source: 'run: mise run gone\n' },
    ]
    expect(verifyCommandMap(miseToml, workflows)).toEqual([{ file: 'a.yaml', task: 'gone' }])
  })
})
