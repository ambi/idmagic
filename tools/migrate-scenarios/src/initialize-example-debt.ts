#!/usr/bin/env bun

import { Glob } from 'bun'
import { readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { parseScenarioDocument } from '../../check/src/gherkin-scenarios.ts'
import { WORKSPACE_ROOT } from '../../workspace/src/workspace.ts'

const glob = new Glob('docs/**/scenarios.feature.md')
const ids: string[] = []
for (const relativePath of glob.scanSync({ cwd: WORKSPACE_ROOT, onlyFiles: true })) {
  const source = await readFile(resolve(WORKSPACE_ROOT, relativePath), 'utf8')
  const parsed = parseScenarioDocument(source)
  if (parsed.findings.length > 0) throw new Error(`${relativePath}: scenario document is invalid`)
  ids.push(...parsed.rules.flatMap((rule) => rule.examples.map((example) => example.id)))
}
ids.sort()
const debt = {
  comment: [
    'Each entry is an example introduced by wi-491 whose exact test correspondence was not asserted without evidence.',
    'Remove an entry when a product test names the EX id. New examples are not admitted to this migration baseline.',
  ],
  untested: ids.map((id) => ({
    id,
    reason: '移行時点で既存テストの入力と観測がこの具体例へ一致することを個別に確認していないため',
  })),
}
await writeFile(
  resolve(WORKSPACE_ROOT, 'tools/check/example-coverage-debt.json'),
  `${JSON.stringify(debt, null, 2)}\n`,
)
await writeFile(
  resolve(WORKSPACE_ROOT, 'tools/check/example-coverage-debt-baseline.json'),
  `${JSON.stringify(
    {
      comment: [
        'Immutable admission set for the example coverage debt introduced by wi-491.',
        'Removing debt entries does not remove IDs here; no new ID may be added.',
      ],
      ids,
    },
    null,
    2,
  )}\n`,
)
console.log(`initialized example coverage debt with ${ids.length} entries`)
