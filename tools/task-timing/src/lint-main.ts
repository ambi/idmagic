#!/usr/bin/env bun

/**
 * Run the Go lint gate with per-analyzer timing on, and print the summary.
 *
 * The lint cache is cleared first: a cached run measures nothing, and a
 * partially cached one measures whichever packages happened to be stale. What
 * this reports is the cold cost, which is the shape of the gate itself.
 */

import { resolve } from 'node:path'
import { formatAnalyzerTable, summarizeAnalyzers } from './analyzers.ts'

const root = resolve(import.meta.dir, '../../..')
const limit = Number(process.argv[2] ?? '15')

const clean = Bun.spawn(['golangci-lint', 'cache', 'clean'], { cwd: root, stdout: 'pipe' })
if ((await clean.exited) !== 0) {
  process.stderr.write('time-lint-go: could not clear the lint cache\n')
  process.exit(1)
}

const started = Bun.nanoseconds()
const lint = Bun.spawn(['golangci-lint', 'run', '--timeout', '15m', './...'], {
  cwd: root,
  env: { ...process.env, GL_DEBUG: 'goanalysis/analyze' },
  stdout: 'pipe',
  stderr: 'pipe',
})
const [stdout, stderr, code] = await Promise.all([
  new Response(lint.stdout).text(),
  new Response(lint.stderr).text(),
  lint.exited,
])
const elapsed = (Bun.nanoseconds() - started) / 1_000_000_000

// The debug lines go to stderr and the findings to stdout; a run that reported
// findings still measured something, so the table is printed either way.
const rows = summarizeAnalyzers(`${stderr}\n${stdout}`)
process.stdout.write(`wall ${elapsed.toFixed(1)}s, ${code === 0 ? 'no findings' : 'findings'}\n\n`)
process.stdout.write(formatAnalyzerTable(rows, limit))
if (code !== 0) process.stderr.write(stdout)
process.exit(code)
