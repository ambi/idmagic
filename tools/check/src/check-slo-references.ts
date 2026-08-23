#!/usr/bin/env bun

import { existsSync } from 'node:fs'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { alertReferences, checkAlerts, declaredObjectives } from './slo-references.ts'

const root = resolve(import.meta.dir, '../../..')

const MONITORING_ASSETS = [
  'infra/docker/prometheus-rules.yml',
  'infra/k8s/monitoring/prometheus-rule.yaml',
]

const declared = declaredObjectives(await readFile(resolve(root, 'docs/capacity.md'), 'utf8'))
if (declared.size === 0) {
  console.error('fail  docs/capacity.md declares no SLO-* or CAP-* objective')
  process.exit(1)
}

const findings = []
for (const path of MONITORING_ASSETS) {
  const source = await readFile(resolve(root, path), 'utf8')
  findings.push(
    ...checkAlerts(path, alertReferences(source), declared, (relative) =>
      existsSync(resolve(root, relative)),
    ),
  )
}

for (const finding of findings) console.error(`fail  ${finding.path}: ${finding.message}`)
if (findings.length > 0) process.exit(1)
console.log(`ok  ${declared.size} objective(s), ${MONITORING_ASSETS.length} monitoring asset(s)`)
