import { describe, expect, it } from 'bun:test'
import { changedGoPackages, parseGoList, type GoPackage } from './changed-packages.ts'

const PACKAGES: GoPackage[] = [
  { importPath: 'app/domain', dir: 'backend/domain', deps: [] },
  { importPath: 'app/usecases', dir: 'backend/usecases', deps: ['app/domain'] },
  { importPath: 'app/handlers', dir: 'backend/handlers', deps: ['app/usecases', 'app/domain'] },
  { importPath: 'app/unrelated', dir: 'backend/unrelated', deps: [] },
]

describe('changedGoPackages', () => {
  it('selects the package a changed file belongs to', () => {
    expect(changedGoPackages(['backend/usecases/start.go'], PACKAGES)).toContain('app/usecases')
  })

  /**
   * The narrow gate exists so the wide one can return from cache. That only
   * holds if the narrow gate covers everything the change can break, which is
   * the reverse dependencies — the packages that compile the changed code in.
   * Selecting only the changed package would leave the wide gate to discover
   * the break, which is the round trip this is meant to remove.
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

  it('ignores files that are not Go source', () => {
    expect(changedGoPackages(['backend/domain/README.md'], PACKAGES)).toEqual([])
  })

  /** A deleted file, or one in a directory that builds no package, selects nothing. */
  it('ignores a changed file no package claims', () => {
    expect(changedGoPackages(['backend/gone/user.go'], PACKAGES)).toEqual([])
  })

  /**
   * A module-wide input changes what every package compiles to, and the
   * dependency graph does not record it. Selecting nothing there would report
   * a green narrow gate for a change that touched everything.
   */
  it('selects every package when a module-wide input changes', () => {
    expect(changedGoPackages(['go.mod'], PACKAGES)).toEqual([
      'app/domain',
      'app/handlers',
      'app/unrelated',
      'app/usecases',
    ])
  })

  it('reports nothing for an unchanged tree', () => {
    expect(changedGoPackages([], PACKAGES)).toEqual([])
  })
})

describe('parseGoList', () => {
  it('reads the import path, directory, and dependencies of each package', () => {
    const output = [
      'app/domain\t/repo/backend/domain\tfmt\t\t',
      'app/usecases\t/repo/backend/usecases\tapp/domain\ttesting\t',
      '',
    ].join('\n')
    expect(parseGoList(output, '/repo')).toEqual([
      { importPath: 'app/domain', dir: 'backend/domain', deps: ['fmt'] },
      { importPath: 'app/usecases', dir: 'backend/usecases', deps: ['app/domain', 'testing'] },
    ])
  })

  /**
   * A package only its tests import is still a reverse dependency: breaking it
   * breaks that package's test run, which is exactly what the narrow gate is
   * asked to catch.
   */
  it('counts a test-only import as a dependency', () => {
    expect(parseGoList('app/handlers\t/repo/h\t\t\tapp/fixtures', '/repo')[0]?.deps).toEqual([
      'app/fixtures',
    ])
  })
})
