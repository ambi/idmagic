#!/usr/bin/env bun

import { readFile } from 'node:fs/promises'
import { relative } from 'node:path'

const seen = new Map<string, string>()
let failed = false
for (const path of process.argv.slice(2)) {
  const source = await readFile(path, 'utf8')
  const rel = relative(process.cwd(), path)
  const ids = [...source.matchAll(/^### (REQ-[A-Z0-9-]+):/gm)].map((match) => match[1] ?? '')
  for (const id of ids) {
    const previous = seen.get(id)
    if (previous) {
      console.error(`${rel}: duplicate requirement id ${id}; first declared in ${previous}`)
      failed = true
    } else {
      seen.set(id, rel)
    }
  }
  if (source.includes('## State machines')) {
    const requiredHeader = '| From | Event | Guard | To | Effects |'
    const machineCount = [...source.matchAll(/^### [^\n]+$/gm)].length
    const tableCount = source.split(requiredHeader).length - 1
    if (tableCount === 0 || tableCount > machineCount) {
      console.error(`${rel}: every state-machine section must use the normative transition table`)
      failed = true
    }
  }
  console.log(`ok  ${rel} (${ids.length} requirement id(s))`)
}
if (failed) process.exit(1)
