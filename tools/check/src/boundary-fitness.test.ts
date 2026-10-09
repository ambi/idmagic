import { describe, expect, it } from 'bun:test'
import {
  findBoundaryViolations,
  parseLogicalArchitecture,
  reconcileBoundaryDebt,
  type BoundaryDebtEntry,
  type GoSource,
} from './boundary-fitness.ts'

const HEADER = '| モジュール | 公開方式 | 公開パッケージ | Go パッケージ | 責務 |'

function logicalArchitecture(rows: string[]): string {
  return [
    '# Logical architecture',
    '',
    '## モジュールの責務',
    '',
    HEADER,
    '| --- | --- | --- | --- | --- |',
    '| [System](system.md) | 組み立て地点 | なし | `backend/cmd`, `backend/shared/http/server_http`, `frontend/` | Wiring |',
    ...rows,
    '',
  ].join('\n')
}

const LEGACY_ROWS = [
  '| [Supplier](supplier.md) | `legacy` | `domain` または `ports` の区画 | `backend/supplier` | Supplies a model |',
  '| [Customer](customer.md) | `legacy` | `domain` または `ports` の区画 | `backend/customer` | Uses the model |',
]

const INTERNAL_SUPPLIER_ROW =
  '| [Supplier](supplier.md) | `internal` | `backend/supplier/api` | `backend/supplier` | Supplies a model |'

function analyze(sources: GoSource[], rows = LEGACY_ROWS) {
  return findBoundaryViolations({
    modulePath: 'example.com/product',
    architecture: parseLogicalArchitecture(logicalArchitecture(rows)),
    sources,
  })
}

function goFile(path: string, ...imports: string[]): GoSource {
  const name = path.split('/').at(-2) ?? 'main'
  return {
    path,
    source: [
      `package ${name}`,
      ...imports.map((imported) => `import "example.com/product/${imported}"`),
      '',
    ].join('\n'),
  }
}

function kinds(sources: GoSource[], rows?: string[]): string[] {
  return analyze(sources, rows).violations.map((violation) => violation.kind)
}

describe('parseLogicalArchitecture', () => {
  it('reads modules, publication modes, and composition prefixes from the responsibility table', () => {
    expect(
      parseLogicalArchitecture(logicalArchitecture([LEGACY_ROWS[1]!, INTERNAL_SUPPLIER_ROW])),
    ).toEqual({
      compositionPrefixes: ['backend/cmd', 'backend/shared/http/server_http'],
      modules: [
        {
          name: 'Customer',
          packagePrefixes: ['backend/customer'],
          publication: 'legacy',
          publicPackages: [],
        },
        {
          name: 'Supplier',
          packagePrefixes: ['backend/supplier'],
          publication: 'internal',
          publicPackages: ['backend/supplier/api'],
        },
      ],
    })
  })

  it('reads the retired Context table as legacy modules with the initial composition prefixes', () => {
    const retired = [
      '| 仕様上の Context | Subdomain | Go パッケージ | 責務 |',
      '| --- | --- | --- | --- |',
      '| [System](system.md) | Supporting | `backend/cmd/internal/bootstrap`, `frontend/` | Wiring |',
      '| [Supplier](supplier.md) | Core | `backend/supplier` | Supplies |',
      '',
    ].join('\n')

    expect(parseLogicalArchitecture(retired)).toEqual({
      compositionPrefixes: [
        'backend/cmd',
        'backend/shared/http/server_http',
        'backend/shared/http/testing_stack',
      ],
      modules: [
        {
          name: 'Supplier',
          packagePrefixes: ['backend/supplier'],
          publication: 'legacy',
          publicPackages: [],
        },
      ],
    })
  })

  it('fails when the responsibility table is missing', () => {
    expect(() => parseLogicalArchitecture('# Logical architecture\n')).toThrow(
      'logical architecture must contain the module responsibility table',
    )
  })

  it('fails when a module loses its Go package mapping', () => {
    expect(() =>
      parseLogicalArchitecture(
        logicalArchitecture([LEGACY_ROWS[0]!.replace('`backend/supplier`', 'backend/supplier')]),
      ),
    ).toThrow('Supplier: the responsibility table must declare a backend package')
  })

  it('fails when two rows claim the same package prefix', () => {
    expect(() =>
      parseLogicalArchitecture(
        logicalArchitecture([
          LEGACY_ROWS[0]!,
          LEGACY_ROWS[1]!.replace('`backend/customer`', '`backend/supplier`'),
        ]),
      ),
    ).toThrow('backend/supplier is assigned more than once')
  })

  it('fails on an unknown publication mode', () => {
    expect(() =>
      parseLogicalArchitecture(logicalArchitecture([LEGACY_ROWS[0]!.replace('legacy', 'open')])),
    ).toThrow('Supplier: publication mode must be legacy or internal')
  })

  it('fails when a legacy module lists public packages', () => {
    expect(() =>
      parseLogicalArchitecture(
        logicalArchitecture([INTERNAL_SUPPLIER_ROW.replace('`internal`', '`legacy`')]),
      ),
    ).toThrow('Supplier: a legacy module derives its public packages from domain and ports')
  })

  it('fails when an internal module declares a package below internal/', () => {
    expect(() =>
      parseLogicalArchitecture(
        logicalArchitecture([
          INTERNAL_SUPPLIER_ROW.replace('backend/supplier/api', 'backend/supplier/internal/api'),
        ]),
      ),
    ).toThrow('Supplier: backend/supplier/internal/api is below internal/ and cannot be public')
  })
})

describe('findBoundaryViolations dependency rules', () => {
  it('accepts a legacy module importing another module ports package', () => {
    expect(kinds([goFile('backend/customer/usecases/a.go', 'backend/supplier/ports')])).toEqual([])
  })

  it('rejects a legacy module importing another module usecases package', () => {
    expect(
      analyze([goFile('backend/customer/usecases/a.go', 'backend/supplier/usecases')]).violations,
    ).toEqual([
      expect.objectContaining({
        id: 'private-import:Customer:backend/customer/usecases->backend/supplier/usecases',
        debtId: 'private-import:Customer->Supplier',
      }),
    ])
  })

  it('keeps legacy naming when a legacy module has started an internal/ directory', () => {
    expect(
      kinds([
        goFile('backend/supplier/internal/store/a.go'),
        goFile('backend/customer/usecases/a.go', 'backend/supplier/usecases'),
        goFile('backend/customer/usecases/b.go', 'backend/supplier/internal/domain'),
      ]),
    ).toEqual(['private-import', 'private-import'])
  })

  it('accepts a declared public package of an internal module', () => {
    expect(
      kinds(
        [
          goFile('backend/supplier/api/a.go'),
          goFile('backend/customer/usecases/a.go', 'backend/supplier/api'),
        ],
        [LEGACY_ROWS[1]!, INTERNAL_SUPPLIER_ROW],
      ),
    ).toEqual([])
  })

  it('diagnoses an undeclared package left outside internal/ and a missing public package', () => {
    const result = analyze(
      [goFile('backend/supplier/usecases/a.go'), goFile('backend/supplier/internal/store/a.go')],
      [LEGACY_ROWS[1]!, INTERNAL_SUPPLIER_ROW],
    )

    expect(result.diagnostics).toEqual([
      'Supplier: backend/supplier/api is declared public but has no production Go file',
      'Supplier: backend/supplier/usecases is outside internal/ but is neither the root nor a declared public package',
    ])
  })

  it('rejects a module importing another module root package', () => {
    expect(kinds([goFile('backend/customer/usecases/a.go', 'backend/supplier')])).toEqual([
      'private-import',
    ])
  })

  it('rejects modules and shared libraries importing a composition point', () => {
    expect(
      analyze([
        goFile('backend/customer/usecases/a.go', 'backend/shared/http/server_http'),
        goFile('backend/shared/clock/a.go', 'backend/cmd/internal/bootstrap'),
      ]).violations.map((violation) => violation.id),
    ).toEqual([
      'composition-import:backend/customer/usecases->backend/shared/http/server_http',
      'composition-import:backend/shared/clock->backend/cmd/internal/bootstrap',
    ])
  })

  it('reports each module edge that participates in a cycle of public imports', () => {
    expect(
      analyze([
        goFile('backend/customer/usecases/a.go', 'backend/supplier/domain'),
        goFile('backend/supplier/usecases/a.go', 'backend/customer/ports'),
      ]).violations.map((violation) => violation.id),
    ).toEqual(['module-cycle:Customer->Supplier', 'module-cycle:Supplier->Customer'])
  })

  it('accepts an acyclic public dependency that no retired Context Map edge declared', () => {
    expect(kinds([goFile('backend/supplier/usecases/a.go', 'backend/customer/domain')])).toEqual([])
  })

  it('accepts a composition point importing a legacy private package during migration', () => {
    expect(kinds([goFile('backend/cmd/idmagic/main.go', 'backend/supplier/usecases')])).toEqual([])
  })

  it('accepts a composition point importing only the root and public packages of an internal module', () => {
    expect(
      kinds(
        [
          goFile('backend/supplier/a.go'),
          goFile('backend/supplier/api/a.go'),
          goFile('backend/cmd/idmagic/main.go', 'backend/supplier', 'backend/supplier/api'),
        ],
        [LEGACY_ROWS[1]!, INTERNAL_SUPPLIER_ROW],
      ),
    ).toEqual([])
  })

  it('rejects a composition point importing an undeclared package of an internal module', () => {
    expect(
      analyze(
        [
          goFile('backend/supplier/api/a.go'),
          goFile('backend/cmd/idmagic/main.go', 'backend/supplier/internal/store'),
        ],
        [LEGACY_ROWS[1]!, INTERNAL_SUPPLIER_ROW],
      ).violations.map((violation) => violation.id),
    ).toEqual(['private-import:System:backend/cmd/idmagic->backend/supplier/internal/store'])
  })

  it('counts a shared library importing a module once, whatever reaches it', () => {
    expect(
      analyze([
        goFile('backend/customer/usecases/a.go', 'backend/shared/bridge'),
        goFile('backend/supplier/usecases/a.go', 'backend/shared/bridge'),
        goFile('backend/shared/bridge/a.go', 'backend/customer/domain', 'backend/customer/ports'),
      ]).violations.map((violation) => violation.id),
    ).toEqual(['shared-dependency:backend/shared/bridge->Customer'])
  })

  it('diagnoses a production package that belongs to no module, composition point, or shared library', () => {
    expect(analyze([goFile('backend/orphan/a.go')]).diagnostics).toEqual([
      'backend/orphan: production package belongs to no module, composition point, or shared library',
    ])
  })

  it('rejects effects in domain packages while allowing time types', () => {
    const result = analyze([
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

    expect(result.violations.map((violation) => violation.id)).toEqual([
      'domain-effect:backend/customer/domain/model.go:crypto/rand',
      'domain-effect:backend/customer/domain/model.go:database/sql',
      'domain-effect:backend/customer/domain/model.go:math/rand',
      'domain-effect:backend/customer/domain/model.go:net',
      'domain-effect:backend/customer/domain/model.go:os',
      'domain-effect:backend/customer/domain/model.go:time.Now',
    ])
  })
})

describe('reconcileBoundaryDebt', () => {
  const violation = {
    id: 'private-import:Customer:backend/customer/usecases->backend/supplier/usecases',
    debtId: 'private-import:Customer->Supplier',
    kind: 'private-import' as const,
    sourceModule: 'Customer',
    targetModule: 'Supplier',
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
      sourceModule: 'Customer',
      targetModule: 'Supplier',
      violations: [violation.id],
      reason: 'Customer still constructs the legacy supplier workflow directly.',
    }
    expect(reconcileBoundaryDebt([violation], [matching])).toEqual([])
    expect(reconcileBoundaryDebt([], [matching])).toEqual([
      expect.objectContaining({ kind: 'stale-debt', id: violation.debtId }),
    ])
  })

  it('reports a ledger id that no observed violation has', () => {
    const padded: BoundaryDebtEntry = {
      id: violation.debtId,
      sourceModule: 'Customer',
      targetModule: 'Supplier',
      violations: [violation.id, 'private-import:Customer:backend/customer/x->backend/supplier/y'],
      reason: 'Customer still constructs the legacy supplier workflow directly.',
    }
    expect(reconcileBoundaryDebt([violation], [padded])).toEqual([
      expect.objectContaining({ kind: 'stale-debt' }),
    ])
  })
})
