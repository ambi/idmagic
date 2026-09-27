import { describe, expect, it } from 'bun:test'
import {
  findBoundaryViolations,
  parseLogicalArchitecture,
  reconcileBoundaryDebt,
  type BoundaryDebtEntry,
  type GoSource,
} from './boundary-fitness.ts'

const LOGICAL_ARCHITECTURE = [
  '# Logical architecture',
  '',
  '## Context Map',
  '',
  '```mermaid',
  'flowchart LR',
  '  Supplier -->|OHS/PL: published model| Customer',
  '  Other[Other]',
  '```',
  '',
  '## Context responsibilities',
  '',
  '| Specification context | Subdomain | Go package | Responsibility |',
  '| --- | --- | --- | --- |',
  '| [Supplier](supplier.md) | Core | `backend/supplier` | Supplies a model |',
  '| [Customer](customer.md) | Core | `backend/customer` | Uses the model |',
  '| [Other](other.md) | Core | `backend/other` | Has no declared relation |',
  '',
].join('\n')

function violations(sources: GoSource[]) {
  return findBoundaryViolations({
    modulePath: 'example.com/product',
    architecture: parseLogicalArchitecture(LOGICAL_ARCHITECTURE),
    sources,
  })
}

describe('parseLogicalArchitecture', () => {
  it('derives context package prefixes and supplier-to-customer relations', () => {
    expect(parseLogicalArchitecture(LOGICAL_ARCHITECTURE)).toEqual({
      contexts: [
        { name: 'Customer', packagePrefixes: ['backend/customer'] },
        { name: 'Other', packagePrefixes: ['backend/other'] },
        { name: 'Supplier', packagePrefixes: ['backend/supplier'] },
      ],
      relations: [{ supplier: 'Supplier', customer: 'Customer', label: 'OHS/PL: published model' }],
    })
  })

  it('fails loudly when the canonical Context table is missing', () => {
    expect(() => parseLogicalArchitecture('# Logical architecture\n')).toThrow(
      'logical architecture must contain a Context responsibility table',
    )
  })

  it('fails loudly when the Context Map names an unknown Context', () => {
    expect(() =>
      parseLogicalArchitecture(LOGICAL_ARCHITECTURE.replace('Supplier -->', 'Unknown -->')),
    ).toThrow('Context Map relation names an unknown Context: Unknown -> Customer')
  })

  it('fails loudly when a Context Map edge leaves the canonical syntax', () => {
    expect(() =>
      parseLogicalArchitecture(
        LOGICAL_ARCHITECTURE.replace(
          'Supplier -->|OHS/PL: published model| Customer',
          'Supplier --> Customer',
        ),
      ),
    ).toThrow('Context Map relation must use Supplier -->|relation| Customer')
  })

  it('fails loudly when a Context loses its Go package mapping', () => {
    expect(() =>
      parseLogicalArchitecture(
        LOGICAL_ARCHITECTURE.replace('`backend/supplier`', 'backend/supplier'),
      ),
    ).toThrow('Supplier: Context responsibility table must declare a backend package')
  })
})

describe('findBoundaryViolations', () => {
  it('rejects private packages even when the context relation allows the dependency', () => {
    const findings = violations([
      {
        path: 'backend/customer/usecases/service.go',
        source: 'package usecases\nimport "example.com/product/backend/supplier/usecases"\n',
      },
    ])

    expect(findings.map((finding) => finding.kind)).toContain('private-import')
  })

  it('rejects a public import in a direction absent from the Context Map', () => {
    const findings = violations([
      {
        path: 'backend/supplier/usecases/service.go',
        source: 'package usecases\nimport "example.com/product/backend/customer/domain"\n',
      },
    ])

    expect(findings.map((finding) => finding.kind)).toContain('undeclared-context-edge')
  })

  it('reports each context edge that participates in a cycle', () => {
    const findings = violations([
      {
        path: 'backend/customer/usecases/service.go',
        source: 'package usecases\nimport "example.com/product/backend/supplier/domain"\n',
      },
      {
        path: 'backend/supplier/usecases/service.go',
        source: 'package usecases\nimport "example.com/product/backend/customer/domain"\n',
      },
    ])

    expect(
      findings.filter((finding) => finding.kind === 'context-cycle').map((finding) => finding.id),
    ).toEqual(['context-cycle:Customer->Supplier', 'context-cycle:Supplier->Customer'])
  })

  it('rejects effects in domain packages while allowing time types', () => {
    const findings = violations([
      {
        path: 'backend/customer/domain/model.go',
        source: [
          'package domain',
          'import (',
          '  "crypto/rand"',
          '  "database/sql"',
          '  "math/rand/v2"',
          '  "net/http"',
          '  "os"',
          '  "time"',
          ')',
          'var _ time.Time',
          'var _ = time.Now()',
          '',
        ].join('\n'),
      },
    ])

    expect(
      findings.filter((finding) => finding.kind === 'domain-effect').map((finding) => finding.id),
    ).toEqual([
      'domain-effect:backend/customer/domain/model.go:crypto/rand',
      'domain-effect:backend/customer/domain/model.go:database/sql',
      'domain-effect:backend/customer/domain/model.go:math/rand',
      'domain-effect:backend/customer/domain/model.go:net',
      'domain-effect:backend/customer/domain/model.go:os',
      'domain-effect:backend/customer/domain/model.go:time.Now',
    ])
  })

  it('rejects a forbidden dependency reached through shared packages', () => {
    const findings = violations([
      {
        path: 'backend/customer/usecases/service.go',
        source: 'package usecases\nimport "example.com/product/backend/shared/bridge"\n',
      },
      {
        path: 'backend/shared/bridge/bridge.go',
        source: 'package bridge\nimport "example.com/product/backend/other/usecases"\n',
      },
    ])

    expect(findings.map((finding) => finding.kind)).toContain('shared-detour')
  })
})

describe('reconcileBoundaryDebt', () => {
  const violation = {
    id: 'private-import:Customer:backend/customer/usecases->backend/supplier/usecases',
    debtId: 'private-import:Customer->Supplier',
    kind: 'private-import' as const,
    sourceContext: 'Customer',
    targetContext: 'Supplier',
    path: 'backend/customer/usecases -> backend/supplier/usecases',
    message: 'Customer reaches a private package in Supplier',
  }

  it('reports an unrecorded violation', () => {
    expect(reconcileBoundaryDebt([violation], [])).toEqual([
      expect.objectContaining({ kind: 'unrecorded-violation', id: violation.debtId }),
    ])
  })

  it('reports stale debt and accepts an exact reasoned entry', () => {
    const matching: BoundaryDebtEntry = {
      id: violation.debtId,
      sourceContext: 'Customer',
      targetContext: 'Supplier',
      violations: [violation.id],
      reason: 'Customer still constructs the legacy supplier workflow directly.',
    }
    expect(reconcileBoundaryDebt([violation], [matching])).toEqual([])
    expect(reconcileBoundaryDebt([], [matching])).toEqual([
      expect.objectContaining({ kind: 'stale-debt', id: violation.debtId }),
    ])
  })
})
