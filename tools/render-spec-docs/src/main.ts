#!/usr/bin/env bun

import { compile, formatDiagnostic, NodeHost } from '@typespec/compiler'
import { Window } from 'happy-dom'
import { mkdir, readFile, readdir, rm, writeFile } from 'node:fs/promises'
import { basename, dirname, resolve } from 'node:path'
import { discoverGeneratedOpenApi } from '../../workspace/src/workspace.ts'
import { renderSpecificationSite, type SourceDocument } from './render.ts'
import { extractTypeSpecCatalog } from './typespec-catalog.ts'

const root = resolve(import.meta.dir, '../../..')
const outputDirectory = resolve(root, 'spec/generated/docs')
const typespecPath = resolve(root, 'spec/main.tsp')
const checkOnly = process.argv.includes('--check')
const openapiPath = await discoverGeneratedOpenApi(root)

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

const program = await compile(NodeHost, typespecPath, { noEmit: true })
if (program.hasError()) {
  throw new Error(program.diagnostics.map((diagnostic) => formatDiagnostic(diagnostic)).join('\n'))
}
const apiSchemas = new Set<string>(Object.keys(openapi.components?.schemas ?? {}))
const models = extractTypeSpecCatalog(program, apiSchemas)
const result = renderSpecificationSite({
  documents,
  openapi,
  repositoryRoot: root,
  outputDirectory,
  openapiFileName: basename(openapiPath),
  models,
})

const validationWindow = new Window()
Object.assign(globalThis, {
  window: validationWindow,
  document: validationWindow.document,
  DOMParser: validationWindow.DOMParser,
  HTMLElement: validationWindow.HTMLElement,
  SVGElement: validationWindow.SVGElement,
})
const { default: mermaid } = await import('mermaid')
mermaid.initialize({ startOnLoad: false, securityLevel: 'strict' })
for (const source of result.mermaidSources) await mermaid.parse(source)
await validationWindow.close()

const dependencyAssets = new Map<string, string>([
  ['mermaid.min.js', resolve(root, 'tools/node_modules/mermaid/dist/mermaid.min.js')],
  [
    'swagger-ui-bundle.js',
    resolve(root, 'tools/node_modules/swagger-ui-dist/swagger-ui-bundle.js'),
  ],
  ['swagger-ui.css', resolve(root, 'tools/node_modules/swagger-ui-dist/swagger-ui.css')],
  ['licenses/mermaid-LICENSE', resolve(root, 'tools/node_modules/mermaid/LICENSE')],
  ['licenses/swagger-ui-LICENSE', resolve(root, 'tools/node_modules/swagger-ui-dist/LICENSE')],
  ['licenses/swagger-ui-NOTICE', resolve(root, 'tools/node_modules/swagger-ui-dist/NOTICE')],
])

if (!checkOnly) {
  if (outputDirectory !== resolve(root, 'spec/generated/docs'))
    throw new Error(`refusing to replace unexpected output directory ${outputDirectory}`)
  await rm(outputDirectory, { recursive: true, force: true })
  for (const [path, content] of Object.entries(result.files)) {
    const output = resolve(outputDirectory, path)
    await mkdir(dirname(output), { recursive: true })
    await writeFile(output, content, 'utf8')
  }
  for (const [path, content] of Object.entries(result.assets)) {
    const output = resolve(outputDirectory, 'assets', path)
    await mkdir(dirname(output), { recursive: true })
    await writeFile(output, content, 'utf8')
  }
  for (const [path, source] of dependencyAssets) {
    const output = resolve(outputDirectory, 'assets', path)
    await mkdir(dirname(output), { recursive: true })
    await writeFile(output, await readFile(source))
  }
  console.log(`wrote ${Object.keys(result.files).length} page(s) to ${outputDirectory}`)
}
console.log(
  `ok  ${documents.length} document(s), ${result.operations} operation(s), ${result.tags.length} API tag(s), ${result.models} TypeSpec symbol(s)`,
)
