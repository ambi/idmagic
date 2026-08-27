import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'

type MiseConfig = {
  tools?: Record<string, unknown>
  env?: Record<string, unknown>
  tasks?: Record<string, { depends?: string[]; run?: unknown; tools?: Record<string, unknown> }>
}

const root = resolve(import.meta.dir, '../../..')
const config = Bun.TOML.parse(await Bun.file(resolve(root, 'mise.toml')).text()) as MiseConfig

describe('mise operational tool boundary', () => {
  it('does not provision PostgreSQL client tools', () => {
    expect(config.tools?.postgres).toBeUndefined()
    expect(Object.keys(config.env ?? {}).filter((name) => name.startsWith('POSTGRES_'))).toEqual([])
    for (const task of Object.values(config.tasks ?? {})) {
      expect(task.tools?.postgres).toBeUndefined()
    }
  })
})

describe('mise generated OpenAPI dependencies', () => {
  it('compiles the specification before every parallel verification consumer', () => {
    for (const task of ['check-spec', 'check-admin-scopes', 'check-api-compat']) {
      expect(config.tasks?.[task]?.depends).toContain('compile-spec')
    }
  })
})

describe('mise change-resistance boundary', () => {
  it('pins mutation tooling without adding it to the universal verification suite', () => {
    expect(config.tools?.['go:github.com/go-gremlins/gremlins/cmd/gremlins']).toBe('0.6.0')
    expect(config.tasks?.['test-go-mutation-package']?.run).toBeDefined()
    expect(config.tasks?.verify?.depends).not.toContain('test-go-mutation-package')
  })
})

describe('mise agent-guidance boundary', () => {
  it('runs repository-local guidance checks from the standard check suite', () => {
    expect(config.tasks?.['check-agent-guidance']?.run).toBeDefined()
    expect(config.tasks?.check?.depends).toContain('check-agent-guidance')
  })
})

describe('mise Markdown link boundary', () => {
  it('runs the Markdown link checker from the standard check suite', () => {
    expect(config.tasks?.['check-links']?.run).toBeDefined()
    expect(config.tasks?.check?.depends).toContain('check-links')
  })
})
