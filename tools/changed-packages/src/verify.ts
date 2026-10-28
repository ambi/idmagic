#!/usr/bin/env bun

/**
 * Run the verification layers a change needs.
 *
 *   verify.ts [--dry-run] changed [base]
 *       the working tree since it left base (main)
 *   verify.ts [--dry-run] integration <verified-source> [rebased]
 *       a rebased branch (HEAD when omitted): only what both sides reach
 *
 * For integration, S is the source commit verified before the rebase and R the
 * rebased result. The source's own change is merge-base(S, R)..S, and the
 * incoming change is S..R, plus anything uncommitted when R is HEAD. Each side
 * was verified on its own tree, so a layer reruns only where both sides reach it.
 *
 * Layers that do not run are reported with their reason and never counted as
 * passed; `.verification/verify-selection.json` records both. `--dry-run`
 * prints the selection without running or recording anything.
 */

import { mkdirSync, writeFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { gatesFor } from './gates.ts'
import { type Plan, intersectPlans, planChange } from './plan.ts'
import { changedBetween, changedSince, commit, goPackages, mergeBase } from './repository.ts'

const root = resolve(import.meta.dir, '../../..')
const args = process.argv.slice(2)
const dryRun = args.includes('--dry-run')
const [mode, revision, rebased] = args.filter((arg) => arg !== '--dry-run')
const packages = goPackages(root)

function select(): { plan: Plan; inputs: Record<string, string[]> } {
  if (mode === 'changed') {
    const files = changedSince(root, revision || 'main')
    return { plan: planChange(files, packages), inputs: { changed: files } }
  }
  if (mode === 'integration' && revision) {
    const source = commit(root, revision)
    const result = commit(root, rebased || 'HEAD')
    const own = changedBetween(root, mergeBase(root, source, result), source)
    const incoming = [
      ...new Set([
        ...changedBetween(root, source, result),
        ...(rebased ? [] : changedSince(root, 'HEAD')),
      ]),
    ]
    return {
      plan: intersectPlans(planChange(own, packages), planChange(incoming, packages)),
      inputs: { source: own, incoming },
    }
  }
  console.error(
    'usage: verify.ts [--dry-run] changed [base] | verify.ts [--dry-run] integration <verified-source> [rebased]',
  )
  process.exit(2)
}

const { plan, inputs } = select()
const gates = gatesFor(plan, packages)
if (dryRun) {
  for (const gate of gates)
    console.log(`run     ${gate.layer}: ${gate.command.slice(0, 4).join(' ')}`)
  for (const layer of Object.keys(plan) as (keyof Plan)[]) {
    if (!plan[layer].selected) console.log(`skip    ${layer}: ${plan[layer].reasons.join('; ')}`)
  }
  if (plan.go.selected) console.log(`go packages: ${plan.go.packages.length} of ${packages.length}`)
  process.exit(0)
}
// verify と同じく並列に走らせ、出力が混ざらないよう各検査の出力は終わったときにまとめて出す。
const results = await Promise.all(
  gates.map(async (gate) => {
    const started = performance.now()
    const child = Bun.spawn(gate.command, { cwd: root, stdout: 'pipe', stderr: 'pipe' })
    const [stdout, stderr, code] = await Promise.all([
      new Response(child.stdout).text(),
      new Response(child.stderr).text(),
      child.exited,
    ])
    console.log(`\n== ${gate.layer}: ${gate.command.join(' ')}\n${stdout}${stderr}`)
    return {
      ...gate,
      state: code === 0 ? 'passed' : 'failed',
      seconds: Math.round((performance.now() - started) / 100) / 10,
    }
  }),
)

const skipped = (Object.keys(plan) as (keyof Plan)[])
  .filter((layer) => !plan[layer].selected)
  .map((layer) => ({ layer, reasons: plan[layer].reasons }))
mkdirSync(join(root, '.verification'), { recursive: true })
writeFileSync(
  join(root, '.verification', 'verify-selection.json'),
  `${JSON.stringify({ schemaVersion: 1, mode, revision, inputs, plan, results, skipped }, null, 2)}\n`,
)

console.log('\nverification selection')
for (const result of results) {
  console.log(
    `  ${result.state.padEnd(7)} ${result.layer}: ${result.command.slice(0, 4).join(' ')} (${result.seconds}s)`,
  )
}
for (const layer of skipped) console.log(`  skipped ${layer.layer}: ${layer.reasons.join('; ')}`)
if (plan.go.selected && !plan.go.all)
  console.log(`  go packages: ${plan.go.packages.length} of ${packages.length}`)
if (results.some((result) => result.state === 'failed')) process.exit(1)
