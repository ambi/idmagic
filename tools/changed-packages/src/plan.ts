/**
 * Choose the verification layers a change needs, and for a rebase, the layers
 * both sides need. Each side was verified on its own tree, so after a rebase a
 * layer is rerun only where both sides' changes can reach it.
 */

import { type GoPackage, goImpact } from './changed-packages.ts'

export type Decision = { selected: boolean; reasons: string[] }
export type GoDecision = Decision & { all: boolean; packages: string[] }
export type Plan = { check: Decision; go: GoDecision; ui: Decision; tools: Decision }

/** Top-level areas some layer owns. A change elsewhere may affect every layer. */
const OWNED = [
  'backend/',
  'frontend/',
  'tools/',
  'docs/',
  'work-items/',
  'infra/',
  'spec/',
  'seed/',
  '.agents/',
  '.claude/',
]

function decision(selected: boolean, reasons: string[]): Decision {
  return { selected, reasons: selected ? reasons : ['no change reaches this layer'] }
}

/** The layers one change set needs. `check` reads the whole repository and is cheap. */
export function planChange(files: readonly string[], packages: readonly GoPackage[]): Plan {
  const unowned = files
    .filter((path) => !OWNED.some((prefix) => path.startsWith(prefix)))
    .map((path) => `${path} is outside every layer`)
  /** A layer that owns a prefix runs when a path under it, or a path no layer owns, changes. */
  const owning = (prefix: string): Decision => {
    const touched = files.find((path) => path.startsWith(prefix))
    if (unowned.length > 0) return decision(true, unowned)
    return decision(touched !== undefined, [`${touched} changed`])
  }
  const impact = goImpact(files, packages)
  return {
    check: decision(files.length > 0, [`${files.length} changed path(s)`]),
    go: {
      ...decision(
        impact.packages.length > 0,
        impact.all ? impact.reasons : ['changed packages and their reverse dependencies'],
      ),
      all: impact.all,
      packages: impact.packages,
    },
    ui: owning('frontend/'),
    tools: owning('tools/'),
  }
}

/**
 * The layers a rebase needs: what both the source's own change and the
 * incoming change can affect. A side that may reach every package leaves the
 * other side's packages.
 */
export function intersectPlans(source: Plan, incoming: Plan): Plan {
  const both = (a: Decision, b: Decision): Decision =>
    decision(a.selected && b.selected, ['both sides of the rebase reach this layer'])
  const packages =
    source.go.all && incoming.go.all
      ? source.go.packages
      : source.go.all
        ? incoming.go.packages
        : incoming.go.all
          ? source.go.packages
          : source.go.packages.filter((name) => incoming.go.packages.includes(name))
  return {
    check: both(source.check, incoming.check),
    go: {
      ...decision(packages.length > 0, ['packages both sides of the rebase reach']),
      all: source.go.all && incoming.go.all,
      packages,
    },
    ui: both(source.ui, incoming.ui),
    tools: both(source.tools, incoming.tools),
  }
}
