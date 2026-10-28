import { describe, expect, it } from 'bun:test'
import { changedGoPackages, goImpact, parseGoList, type GoPackage } from './changed-packages.ts'

const PACKAGES: GoPackage[] = [
  { importPath: 'app/domain', dir: 'backend/domain', deps: [] },
  { importPath: 'app/usecases', dir: 'backend/usecases', deps: ['app/domain'] },
  { importPath: 'app/handlers', dir: 'backend/handlers', deps: ['app/usecases', 'app/domain'] },
  { importPath: 'app/handlers/inner', dir: 'backend/handlers/inner', deps: [] },
  { importPath: 'app/unrelated', dir: 'backend/unrelated', deps: [] },
]
const ALL = PACKAGES.map((one) => one.importPath).sort()

describe('changedGoPackages', () => {
  it('selects the package a changed file belongs to', () => {
    expect(changedGoPackages(['backend/usecases/start.go'], PACKAGES)).toContain('app/usecases')
  })

  /**
   * The narrow gate covers everything the change can break, which is the
   * reverse dependencies — the packages whose test binaries compile the
   * changed code in.
   */
  it('selects every package that depends on a changed one, directly or not', () => {
    expect(changedGoPackages(['backend/domain/user.go'], PACKAGES)).toEqual([
      'app/domain',
      'app/handlers',
      'app/usecases',
    ])
  })

  it('leaves out packages the change cannot reach', () => {
    expect(changedGoPackages(['backend/domain/user.go'], PACKAGES)).not.toContain('app/unrelated')
  })

  /** testdata and go:embed files are inputs of the nearest package above them. */
  it('gives a non-Go file to the deepest package directory that contains it', () => {
    expect(changedGoPackages(['backend/domain/testdata/case.json'], PACKAGES)).toEqual([
      'app/domain',
      'app/handlers',
      'app/usecases',
    ])
    expect(changedGoPackages(['backend/handlers/inner/defaults/a.yaml'], PACKAGES)).toEqual([
      'app/handlers/inner',
    ])
  })

  /** A deleted package's importers change in the same diff, so its files add nothing. */
  it('ignores a backend file no package claims', () => {
    expect(changedGoPackages(['backend/gone/user.go'], PACKAGES)).toEqual([])
  })

  it('ignores paths that no Go test reads', () => {
    expect(
      changedGoPackages(
        [
          'docs/modules/demo/features/run.md',
          'work-items/active/wi-1-x.md',
          'tools/check/src/x.ts',
          '.agents/skills/a/SKILL.md',
          'frontend/src/App.tsx',
          'frontend/tests/e2e/a.spec.ts',
        ],
        PACKAGES,
      ),
    ).toEqual([])
  })

  /**
   * Tests read files outside their package (the schema, deployment assets,
   * frontend gateway configuration). Listing each one would miss the next
   * test that starts reading a file, so any such path selects everything.
   */
  it('selects every package for an input outside the packages it cannot attribute', () => {
    for (const path of [
      'go.mod',
      'mise.toml',
      'infra/schema/postgres.sql',
      'frontend/Caddyfile',
      'spec/modules/demo/main.tsp',
    ]) {
      expect(changedGoPackages([path], PACKAGES)).toEqual(ALL)
    }
  })

  it('reports nothing for an unchanged tree', () => {
    expect(changedGoPackages([], PACKAGES)).toEqual([])
  })
})

describe('goImpact', () => {
  it('names the unattributable path that selects every package', () => {
    expect(goImpact(['docs/a.md', 'infra/k8s/base/api.yaml'], PACKAGES)).toEqual({
      all: true,
      packages: ALL,
      reasons: ['infra/k8s/base/api.yaml is outside every package and may be read by tests'],
    })
  })
})

describe('parseGoList', () => {
  /**
   * `go list -test` lists the package, its test variant, its external test
   * package, and the test binary. They share one directory; the binary's
   * dependencies include what only the tests import, transitively.
   */
  it('merges the test variants into one package with every test dependency', () => {
    const output = [
      'app/domain\t/repo/backend/domain\tfmt',
      'app/usecases\t/repo/backend/usecases\tapp/domain',
      'app/usecases.test\t/repo/backend/usecases\tapp/domain,app/fixtures,app/fixtures/deep',
      'app/usecases [app/usecases.test]\t/repo/backend/usecases\tapp/domain,testing',
      'app/usecases_test [app/usecases.test]\t/repo/backend/usecases\tapp/usecases [app/usecases.test]',
      '',
    ].join('\n')
    expect(parseGoList(output, '/repo')).toEqual([
      { importPath: 'app/domain', dir: 'backend/domain', deps: ['fmt'] },
      {
        importPath: 'app/usecases',
        dir: 'backend/usecases',
        deps: ['app/domain', 'app/fixtures', 'app/fixtures/deep', 'app/usecases', 'testing'],
      },
    ])
  })
})
