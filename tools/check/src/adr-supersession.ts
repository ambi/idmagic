/**
 * Supersession-hygiene guard for ADRs (ADR_FORMAT.md §「廃止・置換（supersede）」).
 *
 * When a later ADR replaces an earlier decision, both sides declare the
 * relationship in frontmatter and the two directions must agree:
 *
 *   new ADR:  supersedes:    [ADR-NNN]
 *   old ADR:  superseded_by: [ADR-NNN]
 *
 * This module is pure. `extractAdrIds` normalizes a raw frontmatter value
 * (scalar, list, or prose such as `ADR-046 (…条項)`) into canonical `adr-<NNN>`
 * ids; `checkAdrSupersession` verifies referential existence, bidirectional
 * agreement, and that a `superseded` status names a successor.
 */

import { basename } from 'node:path'
import type { IdFinding } from './record-ids.ts'

export type AdrSupersessionRecord = {
  /** Canonical id, `adr-<NNN>` (see record-ids.ts). */
  id: string
  path: string
  status?: string
  supersedes: string[]
  supersededBy: string[]
}

const ADR_ID_RE = /ADR-(\d{1,4})/gi

/** Pull every `ADR-NNN` token out of a raw frontmatter value and normalize each
 *  to `adr-<NNN>` (zero-padded to 3). Accepts a scalar string, an array of
 *  strings, or free prose; anything else yields no ids. Order-preserving and
 *  de-duplicated. */
export function extractAdrIds(value: unknown): string[] {
  const parts: string[] = []
  if (typeof value === 'string') parts.push(value)
  else if (Array.isArray(value))
    for (const item of value) if (typeof item === 'string') parts.push(item)
  const ids: string[] = []
  for (const part of parts) {
    for (const m of part.matchAll(ADR_ID_RE)) {
      const id = `adr-${(m[1] ?? '').padStart(3, '0')}`
      if (!ids.includes(id)) ids.push(id)
    }
  }
  return ids
}

/** Verify supersession frontmatter across a set of ADRs. */
export function checkAdrSupersession(records: AdrSupersessionRecord[]): IdFinding[] {
  const byId = new Map<string, AdrSupersessionRecord>()
  for (const record of records) byId.set(record.id, record)

  const findings: IdFinding[] = []
  for (const record of records) {
    const label = basename(record.path)

    for (const target of record.supersedes) {
      const other = byId.get(target)
      if (!other) {
        findings.push({ path: record.path, message: `supersedes unknown ADR '${target}'` })
        continue
      }
      if (!other.supersededBy.includes(record.id)) {
        findings.push({
          path: record.path,
          message: `supersedes '${target}' but ${basename(other.path)} does not list 'superseded_by: [${record.id}]'`,
        })
      }
    }

    for (const target of record.supersededBy) {
      const other = byId.get(target)
      if (!other) {
        findings.push({ path: record.path, message: `superseded_by unknown ADR '${target}'` })
        continue
      }
      if (!other.supersedes.includes(record.id)) {
        findings.push({
          path: record.path,
          message: `superseded_by '${target}' but ${basename(other.path)} does not list 'supersedes: [${record.id}]'`,
        })
      }
    }

    if (record.status === 'superseded' && record.supersededBy.length === 0) {
      findings.push({
        path: record.path,
        message: `${label} has 'status: superseded' but no 'superseded_by'`,
      })
    }
  }
  return findings
}
