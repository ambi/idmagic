import { describe, expect, it } from 'bun:test'
import {
  type AdrSupersessionRecord,
  checkAdrSupersession,
  extractAdrIds,
} from './adr-supersession.ts'

const NS = '/repo/decisions'
function rec(id: string, extra: Partial<AdrSupersessionRecord> = {}): AdrSupersessionRecord {
  return {
    id,
    path: `${NS}/${id}.md`,
    supersedes: [],
    supersededBy: [],
    ...extra,
  }
}

describe('extractAdrIds', () => {
  it('normalizes a scalar id to adr-<NNN>', () => {
    expect(extractAdrIds('ADR-92')).toEqual(['adr-092'])
  })

  it('normalizes a list of ids', () => {
    expect(extractAdrIds(['ADR-068', 'ADR-070'])).toEqual(['adr-068', 'adr-070'])
  })

  it('pulls the id out of prose and drops the rest', () => {
    expect(extractAdrIds('ADR-046 (username / IP 条項)')).toEqual(['adr-046'])
  })

  it('de-duplicates and ignores non-string values', () => {
    expect(extractAdrIds(['ADR-1', 'ADR-001', 42])).toEqual(['adr-001'])
    expect(extractAdrIds(undefined)).toEqual([])
  })
})

describe('checkAdrSupersession', () => {
  it('accepts a bidirectionally consistent pair', () => {
    const findings = checkAdrSupersession([
      rec('adr-094', { status: 'superseded', supersededBy: ['adr-095'] }),
      rec('adr-095', { supersedes: ['adr-094'] }),
    ])
    expect(findings).toEqual([])
  })

  it('flags a one-sided supersedes (partner missing superseded_by)', () => {
    const findings = checkAdrSupersession([
      rec('adr-097', { supersedes: ['adr-096'] }),
      rec('adr-096'),
    ])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('does not list')
  })

  it('flags a reference to a non-existent ADR', () => {
    const findings = checkAdrSupersession([rec('adr-095', { supersedes: ['adr-999'] })])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain("unknown ADR 'adr-999'")
  })

  it('flags status superseded without a successor', () => {
    const findings = checkAdrSupersession([rec('adr-094', { status: 'superseded' })])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('no')
  })

  it('accepts a partial supersede kept as accepted', () => {
    const findings = checkAdrSupersession([
      rec('adr-046', { status: 'accepted', supersededBy: ['adr-104'] }),
      rec('adr-104', { status: 'accepted', supersedes: ['adr-046'] }),
    ])
    expect(findings).toEqual([])
  })
})
