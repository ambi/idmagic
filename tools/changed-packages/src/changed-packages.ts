/**
 * Select the Go packages a change can break.
 *
 * The decision is handed to the dependency graph `go list -test` computes: a
 * package is selected when its test binary compiles in a package the change
 * touched. A changed file belongs to the deepest package directory that holds
 * it, so testdata and go:embed files count as inputs of their package.
 *
 * Tests also read files outside their package (the database schema,
 * deployment assets, the frontend gateway configuration). Listing each one
 * would miss the next test that starts reading a file, so a path that no
 * package claims selects every package, unless it lies under a prefix no Go
 * test reads. `docs/` is one: `test-document-reads` rejects Go tests that name
 * it.
 */

import { relative } from 'node:path'

export type GoPackage = {
  importPath: string
  /** Repository-relative directory of the package. */
  dir: string
  /** Every package the package's test binary compiles in, test-only imports included. */
  deps: readonly string[]
}

export type GoImpact = {
  /** Whether the change may reach every package. */
  all: boolean
  packages: string[]
  /** Why `all` is set; empty when it is not. */
  reasons: string[]
}

/** Paths no Go test reads. Changes under them select no package. */
const NOT_GO_INPUTS = [
  'docs/',
  'work-items/',
  'tools/',
  '.agents/',
  '.claude/',
  'frontend/src/',
  'frontend/tests/',
]

/** Strip the `[pkg.test]` variant suffix `go list -test` adds. */
function base(importPath: string): string {
  return importPath.replace(/ \[.*\]$/, '')
}

/** The package a `go list -test` entry belongs to: itself, its test binary, or its external tests. */
function owner(importPath: string): string {
  return base(importPath)
    .replace(/\.test$/, '')
    .replace(/_test$/, '')
}

/**
 * Read `go list -test -f '{{.ImportPath}}\t{{.Dir}}\t{{join .Deps ","}}'`.
 * The package, its test variant, its external test package, and its test
 * binary are merged into one entry whose dependencies are their union.
 */
export function parseGoList(output: string, root: string): GoPackage[] {
  const packages = new Map<string, { dir: string; deps: Set<string> }>()
  for (const line of output.split('\n')) {
    if (line.trim() === '') continue
    const [importPath = '', dir = '', deps = ''] = line.split('\t')
    const name = owner(importPath)
    const entry = packages.get(name) ?? {
      dir: relative(root, dir).replaceAll('\\', '/'),
      deps: new Set<string>(),
    }
    for (const dep of deps.split(',')) {
      const trimmed = base(dep.trim())
      if (trimmed !== '') entry.deps.add(trimmed)
    }
    packages.set(name, entry)
  }
  return [...packages].map(([importPath, entry]) => ({
    importPath,
    dir: entry.dir,
    deps: [...entry.deps].sort(),
  }))
}

/** The deepest package directory that holds the path, if any. */
function claimingPackage(path: string, packages: readonly GoPackage[]): GoPackage | undefined {
  let found: GoPackage | undefined
  for (const one of packages) {
    if (path.startsWith(`${one.dir}/`) && (!found || one.dir.length > found.dir.length)) found = one
  }
  return found
}

export function goImpact(
  changedFiles: readonly string[],
  packages: readonly GoPackage[],
): GoImpact {
  const sorted = packages.map((one) => one.importPath).sort()
  const reasons: string[] = []
  const changed = new Set<string>()
  for (const path of changedFiles) {
    if (NOT_GO_INPUTS.some((prefix) => path.startsWith(prefix))) continue
    const claimed = claimingPackage(path, packages)
    if (claimed) changed.add(claimed.importPath)
    // A deleted package's importers change in the same diff, so its files add nothing.
    else if (!path.startsWith('backend/'))
      reasons.push(`${path} is outside every package and may be read by tests`)
  }
  if (reasons.length > 0) return { all: true, packages: sorted, reasons }
  const selected = packages
    .filter((one) => changed.has(one.importPath) || one.deps.some((dep) => changed.has(dep)))
    .map((one) => one.importPath)
  return { all: false, packages: [...new Set(selected)].sort(), reasons: [] }
}

export function changedGoPackages(
  changedFiles: readonly string[],
  packages: readonly GoPackage[],
): string[] {
  return goImpact(changedFiles, packages).packages
}
