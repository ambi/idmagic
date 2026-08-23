import { describe, expect, it } from 'bun:test'
import { alertReferences, checkAlerts, declaredObjectives } from './slo-references.ts'

const CAPACITY = [
  '| ID | Endpoint population | Target |',
  '| --- | --- | --- |',
  '| SLO-TOKEN-LATENCY | `/token` | p99 ≤ 300 ms |',
  '| CAP-TOKEN-THROUGHPUT | `/token` | 5,000 rps |',
  '',
  'アラートは SLO-TOKEN-LATENCY の予算を見る。',
].join('\n')

const BLOCK_STYLE = [
  '      - alert: TokenLatencyBudgetBurn',
  '        expr: something > 0.3',
  '        labels:',
  '          severity: page',
  '        annotations:',
  '          summary: "burning the error budget of SLO-TOKEN-LATENCY"',
  '          runbook_url: "docs/runbooks/token-endpoint-latency.md"',
].join('\n')

const FLOW_STYLE = [
  '        - alert: JobsFailureRatioBudgetBurn',
  '          labels: { severity: warn }',
  '          annotations: { summary: "dead-letter rate", runbook_url: "docs/runbooks/async-jobs.md" }',
].join('\n')

const exists = (path: string) => path.startsWith('docs/runbooks/')

describe('declaredObjectives', () => {
  it('collects ids that begin a table row', () => {
    expect(declaredObjectives(CAPACITY)).toEqual(
      new Set(['SLO-TOKEN-LATENCY', 'CAP-TOKEN-THROUGHPUT']),
    )
  })

  it('does not treat a mention in prose as a declaration', () => {
    expect(declaredObjectives('SLO-INVENTED-ID を参照する。')).toEqual(new Set())
  })
})

describe('alertReferences', () => {
  it('reads block-style annotations', () => {
    expect(alertReferences(BLOCK_STYLE)).toEqual([
      {
        alert: 'TokenLatencyBudgetBurn',
        severity: 'page',
        objectives: ['SLO-TOKEN-LATENCY'],
        runbook: 'docs/runbooks/token-endpoint-latency.md',
      },
    ])
  })

  it('reads flow-style annotations on one line', () => {
    expect(alertReferences(FLOW_STYLE)).toEqual([
      {
        alert: 'JobsFailureRatioBudgetBurn',
        severity: 'warn',
        objectives: [],
        runbook: 'docs/runbooks/async-jobs.md',
      },
    ])
  })
})

describe('checkAlerts', () => {
  const declared = declaredObjectives(CAPACITY)

  it('accepts an alert naming a declared objective with a runbook that exists', () => {
    expect(checkAlerts('rules.yml', alertReferences(BLOCK_STYLE), declared, exists)).toEqual([])
  })

  it('rejects an objective id nothing declares', () => {
    const alerts = alertReferences(BLOCK_STYLE.replace('SLO-TOKEN-LATENCY', 'SLO-GONE-LATENCY'))
    expect(checkAlerts('rules.yml', alerts, declared, exists)[0]?.message).toContain(
      'names SLO-GONE-LATENCY',
    )
  })

  it('rejects a page alert with no runbook', () => {
    const alerts = alertReferences(
      BLOCK_STYLE.split('\n')
        .filter((line) => !line.includes('runbook_url'))
        .join('\n'),
    )
    expect(checkAlerts('rules.yml', alerts, declared, exists)[0]?.message).toContain(
      'severity page and has no runbook_url',
    )
  })

  it('rejects a runbook path that does not resolve', () => {
    const alerts = alertReferences(BLOCK_STYLE.replace('docs/runbooks/', 'docs/gone/'))
    expect(checkAlerts('rules.yml', alerts, declared, exists)[0]?.message).toContain(
      'points at a missing runbook',
    )
  })

  it('does not require an objective id on an alert that names none', () => {
    expect(checkAlerts('rules.yml', alertReferences(FLOW_STYLE), declared, exists)).toEqual([])
  })
})
