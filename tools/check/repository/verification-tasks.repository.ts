import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'
import { selectChecks } from '../src/registry.ts'
import { directTasks, parseMiseTasks } from '../src/verification-tasks.ts'

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
      'state-machines',
      'documents',
      'specification-rules',
      'document-layout',
    ])
    const all = selectChecks(['all']).map((check) => check.name)
    for (const name of specification) expect(all).toContain(name)
  })
})
