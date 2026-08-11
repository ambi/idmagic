#!/usr/bin/env bun

import { mkdir, readFile, readdir, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { renderSpecificationSite, type SourceDocument } from './render.ts'

const root = resolve(import.meta.dir, '../../..')
const output = resolve(root, 'spec/generated/docs/index.html')
const openapiPath = resolve(root, 'spec/generated/openapi/idmagic.openapi.json')
const checkOnly = process.argv.includes('--check')

const paths = [
  'DEVELOPMENT.md',
  'SPECIFICATION_FORMAT.md',
  'WORK_ITEM_FORMAT.md',
  'spec/SPECIFICATION.md',
]
const contextRoot = resolve(root, 'spec/contexts')
for (const entry of await readdir(contextRoot, { withFileTypes: true })) {
  if (entry.isDirectory()) paths.push(`spec/contexts/${entry.name}/SPECIFICATION.md`)
}

const documents: SourceDocument[] = []
for (const path of paths.sort()) {
  documents.push({ path, source: await readFile(resolve(root, path), 'utf8') })
}
const openapi = JSON.parse(await readFile(openapiPath, 'utf8'))
const result = renderSpecificationSite({ documents, openapi, repositoryRoot: root, output })

if (!checkOnly) {
  await mkdir(dirname(output), { recursive: true })
  await writeFile(output, result.html, 'utf8')
  console.log(`wrote ${output}`)
}
console.log(
  `ok  ${documents.length} document(s), ${result.operations} operation(s), ${result.tags.length} API tag(s)`,
)
