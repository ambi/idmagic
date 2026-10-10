#!/usr/bin/env bun

import { compile, formatDiagnostic, NodeHost } from '@typespec/compiler'
import { Window } from 'happy-dom'
import { mkdir, readFile, readdir, rm, writeFile } from 'node:fs/promises'
import { basename, dirname, resolve } from 'node:path'
import {
  allowsDocument,
  describeDocumentSet,
  type DirectoryListing,
  documentAllowance,
  SYSTEM_DOCUMENT_DIRECTORIES,
} from '../../workspace/src/document-layout.ts'
import { createWorkspaceSnapshot, discoverGeneratedOpenApi } from '../../workspace/src/workspace.ts'
import { findEnglishProse } from './prose-language.ts'
import { renderDocumentationSite, type SourceDocument } from './render.ts'
import { extractTypeSpecCatalog } from './typespec-catalog.ts'

const root = resolve(import.meta.dir, '../../..')
const outputDirectory = resolve(root, 'site')
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

/** Procedures form an open set; README stays first and the remaining subjects sort by file name. */
async function procedureDocuments(directory: string): Promise<string[]> {
  const entries = await readdir(resolve(root, directory), { withFileTypes: true })
  return entries
    .filter((entry) => entry.isFile() && entry.name.endsWith('.md'))
    .map((entry) => entry.name)
    .sort((left, right) => {
      if (left === 'README.md') return -1
      if (right === 'README.md') return 1
      return left.localeCompare(right)
    })
    .map((name) => `${directory}/${name}`)
}

const paths: string[] = []
for (const { directory, names } of SYSTEM_DOCUMENT_DIRECTORIES) {
  paths.push(...(await canonicalDocuments(directory, names)))
}
paths.push(...(await procedureDocuments('docs/development')))
paths.push(...(await procedureDocuments('docs/runbooks')))
const moduleRoot = resolve(root, 'docs/modules')
const moduleDirectories = (await readdir(moduleRoot, { withFileTypes: true }))
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name)
  .sort()
/**
 * モジュールの文書。モジュールの直下、内部設計、機能群と機能スライスの順に、各段では
 * `README.md` を先に置いて集める。置いてよいかは検査と同じ判定（`documentAllowance`）で決める。
 */
async function moduleDocuments(module: string): Promise<string[]> {
  const listings: DirectoryListing[] = []
  const pending = [module]
  while (pending.length > 0) {
    const directory = pending.shift() as string
    const entries = await readdir(resolve(root, directory), { withFileTypes: true })
    listings.push({
      directory,
      files: entries.filter((entry) => entry.isFile()).map((entry) => entry.name),
    })
    for (const entry of entries) {
      if (entry.isDirectory() && !entry.name.startsWith('.'))
        pending.push(`${directory}/${entry.name}`)
    }
  }
  const view = describeDocumentSet(listings)
  const fixed = (directory: string) => documentAllowance(directory, view).names
  const ranked = (directory: string, name: string) => {
    const index = fixed(directory).indexOf(name)
    return index < 0 ? Number.MAX_SAFE_INTEGER : index
  }
  const designFirst = (directory: string) => (directory === `${module}/design` ? 0 : 1)
  return listings
    .sort(
      (left, right) =>
        Number(left.directory !== module) - Number(right.directory !== module) ||
        designFirst(left.directory) - designFirst(right.directory) ||
        left.directory.localeCompare(right.directory),
    )
    .flatMap(({ directory, files }) => {
      const allowance = documentAllowance(directory, view)
      return files
        .filter((name) => allowsDocument(allowance, name))
        .sort(
          (left, right) =>
            ranked(directory, left) - ranked(directory, right) || left.localeCompare(right),
        )
        .map((name) => `${directory}/${name}`)
    })
}

for (const name of moduleDirectories) {
  paths.push(...(await moduleDocuments(`docs/modules/${name}`)))
}

// The order the canonical layout defines is the order the site lists, so the
// paths are not re-sorted into the alphabet here.
const documents: SourceDocument[] = []
for (const path of paths) {
  documents.push({ path, source: await readFile(resolve(root, path), 'utf8') })
}
const openapi = JSON.parse(await readFile(openapiPath, 'utf8'))

const program = await compile(NodeHost, typespecPath, { noEmit: true })
if (program.hasError()) {
  throw new Error(program.diagnostics.map((diagnostic) => formatDiagnostic(diagnostic)).join('\n'))
}
const apiSchemas = new Set<string>(Object.keys(openapi.components?.schemas ?? {}))
const catalog = extractTypeSpecCatalog(program, apiSchemas, root)
const snapshot = createWorkspaceSnapshot(root)
const { moduleAliases } = JSON.parse(await snapshot.read('tools/check/feature-slice-debt.json'))
const sourcePaths = [
  ...(await snapshot.files('backend')),
  ...(await snapshot.files('spec/modules')),
].sort()
const result = renderDocumentationSite({
  documents,
  openapi,
  repositoryRoot: root,
  outputDirectory,
  openapiFileName: basename(openapiPath),
  models: catalog.symbols,
  moduleTags: catalog.moduleTags,
  sourcePaths,
  moduleAliases,
})

// 文章の言語の規則に反する英文は、書き出す前に止める。生成物だけを直しても元の文書に英語が残るので、
// 報告から TypeSpec または Markdown の原稿へ戻って直す。
const englishProse = findEnglishProse(result.files)
if (englishProse.length > 0) {
  for (const { page, text } of englishProse) console.error(`${page}: ${text}`)
  throw new Error(
    `${englishProse.length} English sentence(s) remain in the rendered site; translate them at their source or add a classified entry to ALLOWED_ENGLISH`,
  )
}

const validationWindow = new Window()
Object.assign(globalThis, {
  window: validationWindow,
  document: validationWindow.document,
  DOMParser: validationWindow.DOMParser,
  HTMLElement: validationWindow.HTMLElement,
  SVGElement: validationWindow.SVGElement,
})
const { default: mermaid } = await import('mermaid')
mermaid.initialize({
  startOnLoad: false,
  securityLevel: 'strict',
  layout: 'dagre',
  look: 'classic',
})
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
  if (outputDirectory !== resolve(root, 'site'))
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
  const publishedOpenapi = resolve(outputDirectory, 'openapi', basename(openapiPath))
  await mkdir(dirname(publishedOpenapi), { recursive: true })
  await writeFile(publishedOpenapi, await readFile(openapiPath))
  console.log(`wrote ${Object.keys(result.files).length} page(s) to ${outputDirectory}`)
}
console.log(
  `ok  ${documents.length} document(s), ${result.operations} operation(s), ${result.tags.length} API tag(s), ${result.models} TypeSpec symbol(s)`,
)
