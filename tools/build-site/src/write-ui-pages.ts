import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { dirname } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { modulePageDocument } from '../../workspace/src/document-layout.ts'
import { loadUiCatalog, resolveRoutes, routeDeclarations, validateUiCatalog } from './ui-pages.ts'
import { renderUiDocuments } from './ui-page-documents.ts'

const root = resolve(import.meta.dir, '../../..')
const snapshot = createWorkspaceSnapshot(root)
const declared = await loadUiCatalog(root)
const sources = new Map(
  await Promise.all(
    (await snapshot.files('frontend/src/routes'))
      .filter(
        (path) => path.endsWith('.tsx') && !path.split('/').some((part) => part.startsWith('-')),
      )
      .map(async (path) => [path, await snapshot.read(path)] as const),
  ),
)
const routes = routeDeclarations(await snapshot.read('frontend/src/routeTree.gen.ts'), sources)
const { catalog, findings: unresolved } = resolveRoutes(declared, routes)
const findings = [...unresolved, ...validateUiCatalog(catalog, routes)]
if (findings.length) throw new Error(findings.join('\n'))
const documents = renderUiDocuments(catalog)
const existing = [
  ...(snapshot.exists('docs/design/application/pages')
    ? await snapshot.files('docs/design/application/pages')
    : []),
  ...(await snapshot.files('docs/modules')).filter(
    (path) => modulePageDocument(path) !== undefined,
  ),
]
const stale = existing.filter((path) => path.endsWith('.md') && !documents.has(path))
for (const path of stale) {
  if (process.argv.includes('--check'))
    throw new Error(`${path}: UI の宣言に対応しない古い生成文書がある。`)
  if (!(await snapshot.read(path)).includes('TypeSpec の UI 宣言から'))
    throw new Error(`${path}: 手書きの文書を生成物として削除できない。`)
  await rm(resolve(root, path))
}
for (const [path, output] of documents) {
  const absolute = resolve(root, path)
  if (process.argv.includes('--check')) {
    if (!(await Bun.file(absolute).exists()) || (await Bun.file(absolute).text()) !== output)
      throw new Error(`${path}: UI ページの生成文書が古い。mise run generate-ui-pages を実行する。`)
  } else {
    await mkdir(dirname(absolute), { recursive: true })
    await Bun.write(absolute, output)
  }
}
// 単一文書だった旧出力だけを除く。ページの詳細は生成先の集合で追跡する。
if (!process.argv.includes('--check'))
  await rm(resolve(root, 'docs/design/application/pages.md'), { force: true })
console.log(
  `ok  ${catalog.pages.length} UI page(s), ${catalog.routeNotes.length} non-rendering route(s), ${routes.length} route(s)`,
)
