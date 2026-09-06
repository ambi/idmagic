import { describe, expect, it } from 'bun:test'
import { formatSeconds, formatTimingTable } from './timing.ts'

describe('formatSeconds', () => {
  /**
   * The gates this measures span three orders of magnitude, from a check that
   * finishes in milliseconds to a race-enabled test run of a minute. One
   * significant digit below ten seconds keeps the short ones distinguishable
   * without pretending the long ones are reproducible to the millisecond.
   */
  it('keeps sub-second tasks distinguishable', () => {
    expect(formatSeconds(745)).toBe('0.75')
    expect(formatSeconds(1_060)).toBe('1.06')
  })

  it('rounds tasks of ten seconds and longer to a whole second', () => {
    expect(formatSeconds(9_960)).toBe('9.96')
    expect(formatSeconds(56_700)).toBe('57')
    expect(formatSeconds(83_200)).toBe('83')
  })
})

describe('formatTimingTable', () => {
  const rows = [
    { task: 'lint-go', milliseconds: 83_000, ok: true },
    { task: 'check-links', milliseconds: 745, ok: false },
    { task: 'test-go-race', milliseconds: 57_000, ok: true },
  ]

  /** The slowest gate is the one the next change should be aimed at. */
  it('orders the rows by elapsed time, slowest first', () => {
    const lines = formatTimingTable(rows).split('\n')
    const tasks = lines.map((line) => line.split(' | ')[0])
    expect(tasks).toEqual([
      '| task',
      '| ---',
      '| lint-go',
      '| test-go-race',
      '| check-links',
      '| total',
      '',
    ])
  })

  it('marks which gates failed', () => {
    const table = formatTimingTable(rows)
    expect(table).toContain('| check-links | 0.75 | fail |')
    expect(table).toContain('| lint-go | 83 | ok |')
  })

  /**
   * The total is the serial cost, which is what a comparison across two
   * revisions can be read from. It is deliberately not the wall time of the
   * parallel suite, which varies with what else the machine is doing.
   */
  it('reports the serial total', () => {
    expect(formatTimingTable(rows)).toContain('| total | 141 |')
  })

  it('renders an empty run without inventing rows', () => {
    expect(formatTimingTable([])).toContain('| total | 0.00 |')
  })
})
