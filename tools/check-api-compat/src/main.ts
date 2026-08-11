#!/usr/bin/env bun
/**
 * check-api-compat — detect breaking changes between a frozen OpenAPI
 * release baseline and the currently generated OpenAPI document.
 *
 *   check-api-compat --baseline <file> --current <file>
 *
 * Exits 1 (and prints every finding) if any breaking change is detected,
 * 0 otherwise. Pure logic lives in `./compat.ts`; this is the shell.
 */

import { readFile } from 'node:fs/promises'
import { discoverGeneratedOpenApi, discoverOpenApiBaseline } from '../../workspace/src/workspace.ts'
import { compareOpenApi, type JsonSchema } from './compat.ts'

function parseArgs(argv: readonly string[]): { baseline: string; current: string } | undefined {
  const opts: { baseline?: string; current?: string } = {}
  for (let i = 0; i < argv.length; i++) {
    const flag = argv[i]
    const value = argv[i + 1]
    if ((flag === '--baseline' || flag === '--current') && value !== undefined) {
      opts[flag.slice(2) as 'baseline' | 'current'] = value
      i++
    }
  }
  return opts.baseline && opts.current
    ? { baseline: opts.baseline, current: opts.current }
    : undefined
}

const argv = process.argv.slice(2)
const args =
  argv.length === 0
    ? { baseline: await discoverOpenApiBaseline(), current: await discoverGeneratedOpenApi() }
    : parseArgs(argv)
if (!args) {
  process.stderr.write('Usage: check-api-compat --baseline <file> --current <file>\n')
  process.exit(2)
}

const [baselineRaw, currentRaw] = await Promise.all([
  readFile(args.baseline, 'utf8'),
  readFile(args.current, 'utf8'),
])
const baseline = JSON.parse(baselineRaw) as JsonSchema
const current = JSON.parse(currentRaw) as JsonSchema

const findings = compareOpenApi(baseline, current)
if (findings.length === 0) {
  process.stderr.write(`check-api-compat: no breaking changes vs ${args.baseline}\n`)
  process.exit(0)
}

process.stderr.write(
  `check-api-compat: ${findings.length} breaking change(s) vs ${args.baseline}\n`,
)
for (const finding of findings) {
  process.stderr.write(`  ${finding.operation}: ${finding.message}\n`)
}
process.stderr.write(
  '\nIf this break is intentional: version the path, add a new interface, and mark the old one\n' +
    'deprecated with its sunset policy instead of changing it in place. If the baseline\n' +
    `itself is simply out of date after a release, refresh ${args.baseline}.\n`,
)
process.exit(1)
