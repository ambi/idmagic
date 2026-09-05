#!/usr/bin/env bun

import { Glob } from 'bun'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { parseScenarioDocument } from '../../check/src/gherkin-scenarios.ts'
import { errorTypesNamedByScenarios } from '../../check/src/security-controls.ts'
import { WORKSPACE_ROOT } from '../../workspace/src/workspace.ts'
import { legacyErrorTypes, missingFragments } from './fragment-audit.ts'

const oldRules = new Set<string>()
const newRules = new Set<string>()
const oldErrors = new Set<string>()
const newErrors = new Set<string>()
let alternatives = 0
let examples = 0
let files = 0
const absentFragments: string[] = []

for (const featurePath of new Glob('docs/**/scenarios.feature.md').scanSync({
  cwd: WORKSPACE_ROOT,
  onlyFiles: true,
})) {
  files += 1
  const legacyPath = featurePath.replace('scenarios.feature.md', 'scenarios.md')
  const shown = Bun.spawnSync(['git', 'show', `HEAD:${legacyPath}`], { cwd: WORKSPACE_ROOT })
  if (shown.exitCode !== 0) throw new Error(`cannot read ${legacyPath} from HEAD`)
  const oldSource = shown.stdout.toString()
  const newSource = await readFile(resolve(WORKSPACE_ROOT, featurePath), 'utf8')
  for (const id of oldSource.matchAll(/^### (REQ-[A-Z0-9-]+):/gm)) oldRules.add(id[1] ?? '')
  alternatives += [...oldSource.matchAll(/^  - ALT /gm)].length
  for (const type of legacyErrorTypes(oldSource)) oldErrors.add(type)
  for (const fragment of missingFragments(oldSource, newSource)) {
    absentFragments.push(`${legacyPath}: ${fragment}`)
  }
  const parsed = parseScenarioDocument(newSource)
  if (parsed.findings.length > 0) throw new Error(`${featurePath}: invalid migrated document`)
  for (const rule of parsed.rules) {
    newRules.add(rule.id)
    examples += rule.examples.length
  }
  for (const type of errorTypesNamedByScenarios(newSource)) newErrors.add(type)
}

const absentRules = [...oldRules].filter((id) => !newRules.has(id))
const addedRules = [...newRules].filter((id) => !oldRules.has(id))
const absentErrors = [...oldErrors].filter((id) => !newErrors.has(id))
const addedErrors = [...newErrors].filter((id) => !oldErrors.has(id))
if (
  absentFragments.length > 0 ||
  absentRules.length > 0 ||
  addedRules.length > 0 ||
  absentErrors.length > 0 ||
  addedErrors.length > 0
) {
  throw new Error(
    JSON.stringify(
      { missingFragments: absentFragments, absentRules, addedRules, absentErrors, addedErrors },
      null,
      2,
    ),
  )
}
console.log(
  `audit passed: ${files} files, ${oldRules.size} rules, ${alternatives} alternatives, ${examples} examples, ${oldErrors.size} error types`,
)
