import { describe, expect, it } from 'bun:test'
import { formatAnalyzerTable, summarizeAnalyzers } from './analyzers.ts'

/**
 * `GL_DEBUG=goanalysis/analyze` writes one line per analyzer per package, at
 * whatever unit the duration happens to fall in. A parser that reads only
 * milliseconds would silently drop the seconds-long lines, which are the only
 * ones that matter.
 */
const DEBUG_OUTPUT = [
  'level=debug msg="[goanalysis/analyze] go/analysis: metalinter: goimports: analyzed package \\"domain\\" in 1.5s"',
  'level=debug msg="[goanalysis/analyze] go/analysis: metalinter: goimports: analyzed package \\"kernel\\" in 500ms"',
  'level=debug msg="[goanalysis/analyze] go/analysis: metalinter: revive: analyzed package \\"domain\\" in 120ms"',
  'level=debug msg="[goanalysis/analyze] go/analysis: metalinter: inspect: analyzed package \\"domain\\" in 250µs"',
  'level=debug msg="[goanalysis/analyze] go/analysis: metalinter: inspect: analyzed package \\"kernel\\" in 750ns"',
  'level=info msg="[runner] linters took 1m37.7s"',
  '',
].join('\n')

describe('summarizeAnalyzers', () => {
  const rows = summarizeAnalyzers(DEBUG_OUTPUT)

  it('sums every duration unit the debug output uses', () => {
    expect(rows.find((row) => row.analyzer === 'goimports')?.seconds).toBeCloseTo(2.0, 6)
    expect(rows.find((row) => row.analyzer === 'inspect')?.seconds).toBeCloseTo(0.00025075, 8)
  })

  it('counts the packages each analyzer ran on', () => {
    expect(rows.find((row) => row.analyzer === 'goimports')?.packages).toBe(2)
    expect(rows.find((row) => row.analyzer === 'revive')?.packages).toBe(1)
  })

  /** The dominant analyzer is the whole reason to look at this output. */
  it('orders the analyzers by total time, most expensive first', () => {
    expect(rows.map((row) => row.analyzer)).toEqual(['goimports', 'revive', 'inspect'])
  })

  it('ignores lines that are not per-package analyzer timings', () => {
    expect(rows.some((row) => row.analyzer.includes('runner'))).toBe(false)
  })

  it('reports nothing for output that carries no timings', () => {
    expect(summarizeAnalyzers('level=info msg="0 issues."\n')).toEqual([])
  })
})

describe('formatAnalyzerTable', () => {
  const table = formatAnalyzerTable(summarizeAnalyzers(DEBUG_OUTPUT), 2)

  /**
   * The share is what makes the table actionable: an analyzer at 39% is worth
   * a decision, and one at 3% is not, whatever its absolute number says on
   * this particular machine.
   */
  it('gives each analyzer its share of the total', () => {
    expect(table).toContain('| goimports | 2.00 | 2 | 1000.00 | 94% |')
  })

  it('keeps only the rows asked for and totals all of them', () => {
    expect(table).toContain('| revive | 0.12 | 1 | 120.00 | 6% |')
    expect(table).not.toContain('| inspect |')
    expect(table).toContain('| total (3 analyzers) | 2.12 |')
  })

  it('renders an empty run without dividing by zero', () => {
    expect(formatAnalyzerTable([], 5)).toContain('| total (0 analyzers) | 0.00 |')
  })
})
