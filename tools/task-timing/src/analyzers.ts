/**
 * Break one lint run down by analyzer.
 *
 * `mise run time-verify` says which gate costs the most; when that answer is
 * `lint-go`, the next question is which of its ~260 analyzers is responsible,
 * and the gate itself will not say. golangci-lint will, under
 * `GL_DEBUG=goanalysis/analyze` — but it says it in one line per analyzer per
 * package, which for this repository is over a hundred thousand lines. Reading
 * that is the cost this summary removes.
 *
 * The totals exceed the wall time, because the analyzers run in parallel. They
 * are here to be compared with each other, not with a clock: an analyzer
 * holding a third of the total is a decision to make, whatever the absolute
 * number reads on one machine.
 */

export type AnalyzerRow = {
  analyzer: string
  seconds: number
  packages: number
}

const TIMING = /metalinter: ([^:]+): analyzed package \\?"[^"]*\\?" in ([0-9.]+)(ns|µs|ms|s)\b/
const UNIT_SECONDS: Record<string, number> = { ns: 1e-9, µs: 1e-6, ms: 1e-3, s: 1 }

export function summarizeAnalyzers(debugOutput: string): AnalyzerRow[] {
  const seconds = new Map<string, number>()
  const packages = new Map<string, number>()
  for (const line of debugOutput.split('\n')) {
    const match = TIMING.exec(line)
    if (!match) continue
    const [, analyzer = '', amount = '0', unit = 's'] = match
    seconds.set(analyzer, (seconds.get(analyzer) ?? 0) + Number(amount) * (UNIT_SECONDS[unit] ?? 0))
    packages.set(analyzer, (packages.get(analyzer) ?? 0) + 1)
  }
  return [...seconds.entries()]
    .map(([analyzer, total]) => ({
      analyzer,
      seconds: total,
      packages: packages.get(analyzer) ?? 0,
    }))
    .sort((left, right) => right.seconds - left.seconds)
}

export function formatAnalyzerTable(rows: readonly AnalyzerRow[], limit: number): string {
  const total = rows.reduce((sum, row) => sum + row.seconds, 0)
  const share = (value: number) => (total === 0 ? 0 : Math.round((value / total) * 100))
  return [
    '| analyzer | seconds | packages | ms/package | share |',
    '| --- | --- | --- | --- | --- |',
    ...rows
      .slice(0, limit)
      .map(
        (row) =>
          `| ${row.analyzer} | ${row.seconds.toFixed(2)} | ${row.packages} | ` +
          `${((row.seconds / row.packages) * 1000).toFixed(2)} | ${share(row.seconds)}% |`,
      ),
    `| total (${rows.length} analyzers) | ${total.toFixed(2)} | | | 100% |`,
    '',
  ].join('\n')
}
