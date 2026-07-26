/**
 * Loading and merging of Architecture ledgers (`architecture.yaml`).
 *
 * ADR-143 splits the second layer into a prose design record (`ARCHITECTURE.md`)
 * and a machine-checked ledger (`architecture.yaml`). A ledger may sit next to
 * the code it describes, so the workspace holds several of them: a module is
 * declared by the nearest ancestor ledger, and the root ledger is the fallback.
 * `contexts`, `runtime_units` and `complexity` are cross-cutting and therefore
 * belong to the root ledger alone.
 *
 * `mergeArchitectureLedgers` rebases every ledger-relative `modules[].path` onto
 * the workspace root and returns one document, so the existing cross-checks
 * (`verifyArchitecture`, `evaluateArchitectureWorkspace`) keep operating on a
 * single graph exactly as they did when the ledger lived in one file.
 */

import { normalize, sep } from 'node:path'

export type ArchitectureLedgerFile = {
  /** Workspace-relative path of the ledger, e.g. `backend/jobs/architecture.yaml`. */
  path: string
  doc: unknown
}

export type ArchitectureLedgerFinding = {
  path: string
  message: string
}

export type MergedArchitectureLedger = {
  doc: Record<string, unknown>
  findings: ArchitectureLedgerFinding[]
}

/** Fields which describe the workspace as a whole, not one ledger's subtree. */
const ROOT_ONLY_FIELDS = ['contexts', 'runtime_units', 'complexity'] as const

function slash(path: string): string {
  return path.split(sep).join('/')
}

function dict(value: unknown): Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {}
}

/** Directory holding a ledger, workspace-relative; `.` for the root ledger. */
export function ledgerDirectory(ledgerPath: string): string {
  const normalized = slash(ledgerPath).replace(/^\.\//, '')
  const cut = normalized.lastIndexOf('/')
  return cut < 0 ? '.' : normalized.slice(0, cut)
}

/** Rebase a ledger-relative path onto the workspace root. */
function rebase(dir: string, path: string): string {
  const joined = dir === '.' ? path : `${dir}/${path}`
  return slash(normalize(joined)).replace(/^\.\//, '').replace(/\/$/, '')
}

function isInside(dir: string, path: string): boolean {
  if (dir === '.') return !path.startsWith('..')
  return path === dir || path.startsWith(`${dir}/`)
}

/**
 * Merge every ledger into one workspace-rooted Architecture document.
 *
 * Ledgers are processed in path order so that findings are stable regardless of
 * filesystem traversal order.
 */
export function mergeArchitectureLedgers(
  ledgers: readonly ArchitectureLedgerFile[],
): MergedArchitectureLedger {
  const findings: ArchitectureLedgerFinding[] = []
  const merged: Record<string, unknown> = {}
  const modules: Record<string, unknown> = {}
  const declaredBy = new Map<string, string>()

  const ordered = [...ledgers].sort((left, right) => left.path.localeCompare(right.path))
  const rootLedger = ordered.find((ledger) => ledgerDirectory(ledger.path) === '.')
  if (!rootLedger) {
    findings.push({
      path: 'architecture.yaml',
      message:
        'architecture: no root ledger found; the workspace root must declare architecture.yaml',
    })
  }

  for (const ledger of ordered) {
    const doc = dict(ledger.doc)
    const dir = ledgerDirectory(ledger.path)
    const isRoot = dir === '.'

    if (isRoot) {
      for (const [key, value] of Object.entries(doc)) {
        if (key === 'modules') continue
        merged[key] = value
      }
    } else {
      for (const field of ROOT_ONLY_FIELDS) {
        if (doc[field] === undefined) continue
        findings.push({
          path: ledger.path,
          message: `architecture: '${field}' is cross-cutting and belongs to the root ledger, not '${ledger.path}'`,
        })
      }
    }

    for (const [id, value] of Object.entries(dict(doc.modules))) {
      const previous = declaredBy.get(id)
      if (previous !== undefined) {
        findings.push({
          path: ledger.path,
          message: `architecture: module '${id}' is already declared by '${previous}'`,
        })
        continue
      }
      declaredBy.set(id, ledger.path)

      const record = dict(value)
      const path = record.path
      if (typeof path !== 'string' || path.length === 0) {
        // The schema reports the missing path; keep the entry so the graph stays complete.
        modules[id] = { ...record }
        continue
      }
      const rebased = rebase(dir, path)
      if (!isInside(dir, rebased)) {
        findings.push({
          path: ledger.path,
          message: `architecture: module '${id}' path '${path}' resolves to '${rebased}', outside the ledger directory '${dir}'`,
        })
      }
      modules[id] = { ...record, path: rebased }
    }
  }

  merged.modules = modules
  return { doc: merged, findings }
}
