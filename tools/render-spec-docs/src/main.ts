#!/usr/bin/env bun

import { compile, formatDiagnostic, NodeHost } from '@typespec/compiler'
import { Window } from 'happy-dom'
import { mkdir, readFile, readdir, rm, writeFile } from 'node:fs/promises'
import { basename, dirname, relative, resolve } from 'node:path'
import { CONTEXT_DOCUMENTS, ROOT_DOCUMENTS } from '../../check/src/specification-doc.ts'
import { discoverGeneratedOpenApi } from '../../workspace/src/workspace.ts'
import { renderSpecificationSite, type ScenarioTrace, type SourceDocument } from './render.ts'
import { extractTypeSpecCatalog } from './typespec-catalog.ts'

const root = resolve(import.meta.dir, '../../..')
const outputDirectory = resolve(root, 'spec/generated/docs')
const typespecPath = resolve(root, 'spec/main.tsp')
const checkOnly = process.argv.includes('--check')
const openapiPath = await discoverGeneratedOpenApi(root)

/**
 * Only the names the canonical layout defines are read, so an unrelated
 * Markdown file beside them is not rendered as specification.
 */
async function canonicalDocuments(directory: string, names: readonly string[]): Promise<string[]> {
  const entries = await readdir(resolve(root, directory), { withFileTypes: true })
  const files = new Set(entries.filter((entry) => entry.isFile()).map((entry) => entry.name))
  return names.filter((name) => files.has(name)).map((name) => `${directory}/${name}`)
}

const paths = ['DEVELOPMENT.md', 'SPECIFICATION_FORMAT.md', 'WORK_ITEM_FORMAT.md']
paths.push(...(await canonicalDocuments('spec', ROOT_DOCUMENTS)))
const contextRoot = resolve(root, 'spec/contexts')
const contextDirectories = (await readdir(contextRoot, { withFileTypes: true }))
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name)
  .sort()
for (const name of contextDirectories) {
  paths.push(...(await canonicalDocuments(`spec/contexts/${name}`, CONTEXT_DOCUMENTS)))
}

// The order the canonical layout defines is the order the site lists, so the
// paths are not re-sorted into the alphabet here.
const documents: SourceDocument[] = []
for (const path of paths) {
  documents.push({ path, source: await readFile(resolve(root, path), 'utf8') })
}
const openapi = JSON.parse(await readFile(openapiPath, 'utf8'))

/**
 * Collect what names a scenario outside the specification. The search runs over
 * the working tree rather than an authored index, so the traceability view can
 * never drift from what the repository actually says.
 */
const SCENARIO_IDENTIFIER = /REQ-[A-Z0-9]+-[0-9]+/g
const SKIPPED_DIRECTORIES = new Set([
  '.git',
  'node_modules',
  'vendor',
  'dist',
  'build',
  'generated',
  'coverage',
])
const TEXT_EXTENSIONS = /\.(?:go|ts|tsx|js|jsx|sql|sh|py|rb|java|kt|rs|md|yaml|yml|json|tsp|toml)$/

async function collectTraces(): Promise<ScenarioTrace[]> {
  const sources = new Map<string, Set<string>>()
  const workItems = new Map<string, Set<string>>()
  const walk = async (directory: string): Promise<void> => {
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      if (entry.name.startsWith('.') || SKIPPED_DIRECTORIES.has(entry.name)) continue
      const absolute = resolve(directory, entry.name)
      const path = relative(root, absolute)
      if (entry.isDirectory()) {
        await walk(absolute)
        continue
      }
      // The specification declares the scenarios; only what points at them counts.
      if (path.startsWith('spec/') || !TEXT_EXTENSIONS.test(path)) continue
      const target = path.startsWith('work-items/') ? workItems : sources
      for (const match of (await readFile(absolute, 'utf8')).matchAll(SCENARIO_IDENTIFIER)) {
        const paths = target.get(match[0]) ?? new Set<string>()
        paths.add(path)
        target.set(match[0], paths)
      }
    }
  }
  await walk(root)
  const ids = new Set([...sources.keys(), ...workItems.keys()])
  return [...ids].map((id) => ({
    id,
    sources: [...(sources.get(id) ?? [])].sort(),
    workItems: [...(workItems.get(id) ?? [])].sort(),
  }))
}

const traces = await collectTraces()

const program = await compile(NodeHost, typespecPath, { noEmit: true })
if (program.hasError()) {
  throw new Error(program.diagnostics.map((diagnostic) => formatDiagnostic(diagnostic)).join('\n'))
}
const apiSchemas = new Set<string>(Object.keys(openapi.components?.schemas ?? {}))
const catalog = extractTypeSpecCatalog(program, apiSchemas, root)
const result = renderSpecificationSite({
  documents,
  openapi,
  repositoryRoot: root,
  outputDirectory,
  openapiFileName: basename(openapiPath),
  models: catalog.symbols,
  contextTags: catalog.contextTags,
  traces,
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
