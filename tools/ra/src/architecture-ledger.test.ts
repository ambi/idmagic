import { describe, expect, it } from 'bun:test'
import {
  type ArchitectureLedgerFile,
  ledgerDirectory,
  mergeArchitectureLedgers,
} from './architecture-ledger.ts'

function ledger(path: string, doc: unknown): ArchitectureLedgerFile {
  return { path, doc }
}

const ROOT = ledger('architecture.yaml', {
  context: 'repo',
  updated_at: '2026-01-01',
  contexts: { Example: { spec: 'spec/contexts/example.yaml', summary: 'x' } },
  modules: { shared: { path: 'backend/shared', context: 'Example', layer: 'adapters' } },
  runtime_units: {
    api: { kind: 'api', entrypoint: 'backend/cmd/api/main.go', modules: ['shared'] },
  },
})

describe('ledgerDirectory', () => {
  it('returns . for the root ledger', () => {
    expect(ledgerDirectory('architecture.yaml')).toBe('.')
  })

  it('returns the owning directory for a context ledger', () => {
    expect(ledgerDirectory('backend/jobs/architecture.yaml')).toBe('backend/jobs')
  })
})

describe('mergeArchitectureLedgers', () => {
  it('rebases context-ledger paths onto the workspace root', () => {
    const jobs = ledger('backend/jobs/architecture.yaml', {
      context: 'jobs',
      modules: { 'jobs-domain': { path: 'domain', context: 'Jobs', layer: 'domain' } },
    })
    const { doc, findings } = mergeArchitectureLedgers([ROOT, jobs])
    expect(findings).toEqual([])
    const modules = doc.modules as Record<string, { path: string }>
    expect(modules['jobs-domain']?.path).toBe('backend/jobs/domain')
    expect(modules.shared?.path).toBe('backend/shared')
  })

  it('resolves a ledger root module to its own directory', () => {
    const jobs = ledger('backend/jobs/architecture.yaml', {
      context: 'jobs',
      modules: { jobs: { path: '.', context: 'Jobs', layer: 'domain' } },
    })
    const { doc } = mergeArchitectureLedgers([ROOT, jobs])
    expect((doc.modules as Record<string, { path: string }>).jobs?.path).toBe('backend/jobs')
  })

  it('carries the root ledger cross-cutting fields onto the merged document', () => {
    const { doc } = mergeArchitectureLedgers([ROOT])
    expect(doc.context).toBe('repo')
    expect(doc.contexts).toBeDefined()
    expect(doc.runtime_units).toBeDefined()
  })

  it('rejects a duplicate module id across ledgers', () => {
    const a = ledger('backend/a/architecture.yaml', {
      modules: { dup: { path: 'domain' } },
    })
    const b = ledger('backend/b/architecture.yaml', {
      modules: { dup: { path: 'domain' } },
    })
    const { findings } = mergeArchitectureLedgers([ROOT, a, b])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain("module 'dup' is already declared by")
  })

  it('rejects a module path that escapes its ledger directory', () => {
    const jobs = ledger('backend/jobs/architecture.yaml', {
      modules: { escapee: { path: '../audit/domain' } },
    })
    const { findings } = mergeArchitectureLedgers([ROOT, jobs])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain("outside the ledger directory 'backend/jobs'")
  })

  it('rejects cross-cutting fields declared outside the root ledger', () => {
    const jobs = ledger('backend/jobs/architecture.yaml', {
      contexts: { Jobs: { spec: 'x', summary: 'y' } },
      complexity: { budgets: [], debts: [] },
    })
    const { findings } = mergeArchitectureLedgers([ROOT, jobs])
    expect(findings.map((f) => f.message)).toEqual([
      "architecture: 'contexts' is cross-cutting and belongs to the root ledger, not 'backend/jobs/architecture.yaml'",
      "architecture: 'complexity' is cross-cutting and belongs to the root ledger, not 'backend/jobs/architecture.yaml'",
    ])
  })

  it('reports a workspace with no root ledger', () => {
    const { findings } = mergeArchitectureLedgers([
      ledger('backend/jobs/architecture.yaml', { modules: {} }),
    ])
    expect(findings[0]?.message).toContain('no root ledger found')
  })

  it('is independent of the order ledgers are discovered in', () => {
    const a = ledger('backend/a/architecture.yaml', { modules: { a: { path: 'domain' } } })
    const b = ledger('backend/b/architecture.yaml', { modules: { b: { path: 'domain' } } })
    const forward = mergeArchitectureLedgers([ROOT, a, b])
    const reverse = mergeArchitectureLedgers([b, a, ROOT])
    expect(reverse.doc).toEqual(forward.doc)
    expect(reverse.findings).toEqual(forward.findings)
  })
})
