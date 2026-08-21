#!/usr/bin/env bun

import { readFile } from 'node:fs/promises'
import { dirname, relative } from 'node:path'
import { WORKSPACE_ROOT } from '../../workspace/src/workspace.ts'
import { validateDocument } from './specification-doc.ts'

const seen = new Map<string, string>()
const supersessions: Array<{ where: string; target: string }> = []
const layouts = new Map<string, Set<'split' | 'single'>>()
let failed = false

for (const path of process.argv.slice(2)) {
  const rel = relative(process.cwd(), path)
  // The file name carries the grammar, so validation needs the path as the
  // repository sees it rather than as it looks from the tools directory.
  const canonical = relative(WORKSPACE_ROOT, path).replaceAll('\\', '/')
  const directory = dirname(canonical)
  const layout = layouts.get(directory) ?? new Set<'split' | 'single'>()
  layout.add(canonical.endsWith('/SPECIFICATION.md') ? 'single' : 'split')
  layouts.set(directory, layout)

  const source = await readFile(path, 'utf8')
  const result = validateDocument(canonical, source)
  for (const finding of result.findings) {
    console.error(`${rel}:${finding.line}: ${finding.message}`)
    failed = true
  }
  for (const scenario of result.scenarioIds) {
    const previous = seen.get(scenario.id)
    if (previous) {
      console.error(
        `${rel}:${scenario.line}: duplicate ${scenario.id}; first declared in ${previous}`,
      )
      failed = true
    } else {
      seen.set(scenario.id, `${rel}:${scenario.line}`)
    }
    if (scenario.supersededBy) {
      supersessions.push({ where: `${rel}:${scenario.line}`, target: scenario.supersededBy })
    }
  }
  console.log(`ok  ${rel} (${result.scenarioIds.length} normative scenario id(s))`)
}

// A directory that holds both layouts has two sources of truth for the same
// content, which is the one state the migration must never come to rest in.
for (const [directory, layout] of layouts) {
  if (layout.size > 1) {
    console.error(
      `${directory}: SPECIFICATION.md cannot coexist with the split canonical documents`,
    )
    failed = true
  }
}

for (const supersession of supersessions) {
  if (!seen.has(supersession.target)) {
    console.error(`${supersession.where}: superseding ${supersession.target} does not exist`)
    failed = true
  }
}

if (failed) process.exit(1)
