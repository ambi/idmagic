/**
 * Present the elapsed time of every gate in a verification suite.
 *
 * The point of the table is a comparison: the same suite on the same machine
 * before and after a change, so that a claim about a gate getting faster is an
 * observation rather than an impression. Everything here is therefore ordered
 * by cost and totalled serially — the parallel wall time is what the machine's
 * other work decides, and it cannot be compared across two runs.
 */

export type TimingRow = {
  task: string
  milliseconds: number
  ok: boolean
}

/** Seconds at a precision the measurement can actually support. */
export function formatSeconds(milliseconds: number): string {
  const seconds = milliseconds / 1000
  // Rounded off the integer millisecond rather than through toFixed, whose
  // binary representation turns an exact half — 745 ms — downwards.
  return seconds >= 10
    ? String(Math.round(seconds))
    : (Math.round(milliseconds / 10) / 100).toFixed(2)
}

export function formatTimingTable(rows: readonly TimingRow[]): string {
  const ordered = [...rows].sort((left, right) => right.milliseconds - left.milliseconds)
  const total = rows.reduce((sum, row) => sum + row.milliseconds, 0)
  return [
    '| task | seconds | status |',
    '| --- | --- | --- |',
    ...ordered.map(
      (row) => `| ${row.task} | ${formatSeconds(row.milliseconds)} | ${row.ok ? 'ok' : 'fail'} |`,
    ),
    `| total | ${formatSeconds(total)} | ${rows.every((row) => row.ok) ? 'ok' : 'fail'} |`,
    '',
  ].join('\n')
}
