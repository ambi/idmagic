#!/usr/bin/env bun

import { readdir } from 'node:fs/promises'
import { resolve } from 'node:path'
import { verifyCommandMap } from './command-map.ts'

const root = resolve(import.meta.dir, '../../..')
const workflowDirectory = resolve(root, '.github/workflows')

const workflows: Array<{ file: string; source: string }> = []
try {
  for (const entry of await readdir(workflowDirectory, { withFileTypes: true })) {
    if (!entry.isFile() || !/\.ya?ml$/.test(entry.name)) continue
    workflows.push({
      file: `.github/workflows/${entry.name}`,
      source: await Bun.file(resolve(workflowDirectory, entry.name)).text(),
    })
  }
} catch {
  // A repository without workflows has nothing to disagree with.
}

const justfile = await Bun.file(resolve(root, 'justfile')).text()
const findings = verifyCommandMap(justfile, workflows)

for (const finding of findings) {
  console.error(
    `${finding.file}: workflow calls a recipe the justfile does not define: ${finding.recipe}`,
  )
}
if (findings.length > 0) process.exit(1)
console.log(`ok  command map (${workflows.length} workflow file(s))`)
