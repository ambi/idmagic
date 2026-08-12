#!/usr/bin/env bun

import { readFile } from 'node:fs/promises'
import { basename, relative } from 'node:path'
import { validateSpecification } from './specification-doc.ts'

const seen = new Map<string, string>()
const supersessions: Array<{ where: string; target: string }> = []
let failed = false

for (const path of process.argv.slice(2)) {
  const rel = relative(process.cwd(), path)
  if (basename(path) !== 'SPECIFICATION.md') {
    console.error(`${rel}: canonical document must be named SPECIFICATION.md`)
    failed = true
    continue
  }
  const source = await readFile(path, 'utf8')
  const result = validateSpecification(source)
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

for (const supersession of supersessions) {
  if (!seen.has(supersession.target)) {
    console.error(`${supersession.where}: superseding ${supersession.target} does not exist`)
    failed = true
  }
}

if (failed) process.exit(1)
