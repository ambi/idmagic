#!/usr/bin/env bun

import { resolve } from 'node:path'
import { agentGuidanceFiles, verifyAgentGuidance } from './agent-guidance.ts'

const root = resolve(import.meta.dir, '../../..')
const documents = await Promise.all(
  agentGuidanceFiles.map(async (file) => ({
    file,
    source: await Bun.file(resolve(root, file)).text(),
  })),
)
const findings = verifyAgentGuidance(documents)

for (const finding of findings) {
  console.error(`${finding.file}: ${finding.message}`)
}
if (findings.length > 0) process.exit(1)
console.log(`ok  agent guidance (${documents.length} skill file(s))`)
