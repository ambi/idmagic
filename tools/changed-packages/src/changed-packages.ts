/**
 * Select the Go packages a working-tree change can break.
 *
 * The verification ladder asks for the narrowest gate that can still fail on
 * what was just changed, and then for the wide gate at the end. Deciding which
 * packages are narrow enough is the part people get wrong — it is easier to
 * run everything than to work out what is related — so the decision is handed
 * to the dependency graph `go list` already computes. What the narrow gate
 * runs is the changed packages together with everything that compiles them in,
 * under the same `-race` build the final gate uses, which is what lets the
 * final gate answer from the test cache instead of running them again.
 */

import { relative } from 'node:path'

export type GoPackage = {
  importPath: string
  /** Repository-relative directory of the package. */
  dir: string
  /** Every package this one needs, its test-only imports included. */
  deps: readonly string[]
}

/**
 * Inputs that change what every package compiles to. The dependency graph does
 * not record them, so a change to one selects the whole module.
 */
const MODULE_WIDE = new Set(['go.mod', 'go.sum'])

/**
 * Read the tab-separated package listing `go list` is asked for. A line is
 * `<import path>\t<directory>\t<deps>\t<test imports>\t<external test
 * imports>`, each import list comma-separated. The tabular form is used rather
 * than `-json` because `go list` emits a stream of concatenated objects, which
 * is not a JSON document and cannot be parsed as one.
 */
export function parseGoList(output: string, root: string): GoPackage[] {
  return output
    .split('\n')
    .map((line) => line.trimEnd())
    .filter((line) => line !== '')
    .map((line) => {
      const [importPath = '', dir = '', ...imports] = line.split('\t')
      const deps = new Set(
        imports
          .flatMap((group) => group.split(','))
          .map((entry) => entry.trim())
          .filter((entry) => entry !== ''),
      )
      return {
        importPath,
        dir: relative(root, dir).replaceAll('\\', '/'),
        deps: [...deps],
      }
    })
}

export function changedGoPackages(
  changedFiles: readonly string[],
  packages: readonly GoPackage[],
): string[] {
  const sorted = () => packages.map((one) => one.importPath).sort()
  if (changedFiles.some((path) => MODULE_WIDE.has(path))) return sorted()

  const directories = new Set(
    changedFiles
      .filter((path) => path.endsWith('.go'))
      .map((path) => path.slice(0, Math.max(0, path.lastIndexOf('/')))),
  )
  if (directories.size === 0) return []

  const changed = new Set(
    packages.filter((one) => directories.has(one.dir)).map((one) => one.importPath),
  )
  if (changed.size === 0) return []

  const selected = packages
    .filter((one) => changed.has(one.importPath) || one.deps.some((dep) => changed.has(dep)))
    .map((one) => one.importPath)
  return [...new Set(selected)].sort()
}
