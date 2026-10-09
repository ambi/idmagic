#!/usr/bin/env bun

/**
 * Git の履歴から、同じコミットで変わったモジュールの組と回数を出力する。
 * 合否を判定しないレポートであり、`mise run verify` には含めない。
 *
 * 使い方: main.ts [--revision <rev>] [--limit <n>] [--exclude-at <modules>]
 */

import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { parseLogicalArchitecture } from '../../check/src/boundary-fitness.ts'
import { changeCoupling, pathClassifier, renderCouplingReport } from './coupling.ts'
import { readFirstParentChanges } from './git-history.ts'

const root = resolve(import.meta.dir, '../../..')

function option(name: string, fallback: string): string {
  const index = process.argv.indexOf(name)
  return index === -1 ? fallback : (process.argv[index + 1] ?? fallback)
}

function positiveInteger(name: string, value: string): number {
  const parsed = Number(value)
  if (!Number.isInteger(parsed) || parsed < 1) {
    console.error(`${name} must be a positive integer: ${value}`)
    process.exit(2)
  }
  return parsed
}

const revision = option('--revision', 'HEAD')
const limit = positiveInteger('--limit', option('--limit', '500'))
const excludeAt = positiveInteger('--exclude-at', option('--exclude-at', '6'))

const architecture = parseLogicalArchitecture(
  await readFile(resolve(root, 'docs/design/architecture/logical.md'), 'utf8'),
)
const sqlc = Bun.YAML.parse(await readFile(resolve(root, 'sqlc.yaml'), 'utf8')) as {
  sql?: { queries?: string; gen?: { go?: { out?: string } } }[]
}
const queryDirectories = new Set(
  (sqlc.sql ?? []).flatMap((entry) => (entry.queries ? [entry.queries.replace(/\/+$/, '')] : [])),
)
const generatedDirectories = new Set(
  (sqlc.sql ?? []).flatMap((entry) =>
    entry.gen?.go?.out ? [entry.gen.go.out.replace(/\/+$/, '')] : [],
  ),
)

const history = readFirstParentChanges(root, revision, limit)
const report = changeCoupling(
  history.commits,
  pathClassifier({ architecture, queryDirectories, generatedDirectories }),
  excludeAt,
)
console.log(renderCouplingReport(report, { revision: history.revision, limit }))
