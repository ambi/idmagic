import type { Dirent } from 'node:fs'
import {
  canonicalDocumentNames,
  CONTEXT_DOCUMENTS,
  FREELY_NAMED_DOCUMENT_DIRECTORIES,
  type DirectoryListing,
} from '../../workspace/src/document-layout.ts'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { verifyCanonicalDocumentSet } from './canonical-document-set.ts'
import { checkNormativeCoverage, citedNormativeIds, type DeclaredId } from './normative-coverage.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'
import { validateDocument } from './specification-doc.ts'
import { verifySubdomainClassification } from './subdomain-classification.ts'

const PRODUCT_TREES = ['backend', 'frontend']
const TEST_FILE = /(?:_test\.go|\.(?:test|spec)\.tsx?)$/
const EXCLUDED_DIRECTORIES = ['node_modules', 'vendor', 'dist', 'build', 'generated']

async function canonicalDirectories(snapshot: WorkspaceSnapshot): Promise<DirectoryListing[]> {
  const listings: DirectoryListing[] = []
  const pending = ['docs']
  while (pending.length > 0) {
    const directory = pending.shift()!
    let entries: Dirent[] = []
    try {
      entries = await snapshot.list(directory)
    } catch {
      entries = []
    }
    listings.push({
      directory,
      files: entries.filter((entry) => entry.isFile()).map((entry) => entry.name),
    })
    for (const entry of entries) {
      // 道具の作業跡は文書の置き場所ではないので、隠しディレクトリはたどらない。
      if (!entry.isDirectory() || entry.name.startsWith('.')) continue
      const child = `${directory}/${entry.name}`
      if (!FREELY_NAMED_DOCUMENT_DIRECTORIES.has(child)) pending.push(child)
    }
  }
  return listings
}

export async function checkDocuments(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
): Promise<CheckOutcome> {
  const listings = await canonicalDirectories(snapshot)
  const lines = verifyCanonicalDocumentSet(listings).map(
    (finding) => `fail  ${finding.path}: ${finding.message}`,
  )
  let failed = lines.length > 0
  const contextDirectories = listings
    .filter((listing) => listing.directory.startsWith('docs/domain/'))
    .map((listing) => listing.directory.slice('docs/domain/'.length))
  if (snapshot.exists('docs/design/architecture/logical.md')) {
    const classifications = verifySubdomainClassification(
      await snapshot.read('docs/design/architecture/logical.md'),
      contextDirectories,
    )
    failed ||= classifications.length > 0
    lines.push(
      ...classifications.map(
        (finding) =>
          `fail  docs/design/architecture/logical.md:${finding.line}: ${finding.message}`,
      ),
    )
  }

  const paths = listings
    .flatMap((listing) => {
      const allowed = new Set(canonicalDocumentNames(listing.directory) ?? CONTEXT_DOCUMENTS)
      return listing.files
        .filter((name) => allowed.has(name))
        .map((name) => `${listing.directory}/${name}`)
    })
    .sort()
  const seen = new Map<string, string>()
  const supersessions: Array<{ where: string; target: string }> = []
  const scenarios: DeclaredId[] = []
  const examples: DeclaredId[] = []
  const standards: DeclaredId[] = []
  for (const path of paths) {
    const result = validateDocument(path, await snapshot.read(path))
    failed ||= result.findings.length > 0
    lines.push(...result.findings.map((finding) => `${path}:${finding.line}: ${finding.message}`))
    for (const declaration of [...result.scenarioIds, ...result.exampleIds]) {
      const previous = seen.get(declaration.id)
      if (previous) {
        failed = true
        lines.push(
          `${path}:${declaration.line}: duplicate ${declaration.id}; first declared in ${previous}`,
        )
      } else {
        seen.set(declaration.id, `${path}:${declaration.line}`)
      }
    }
    for (const scenario of result.scenarioIds) {
      if (scenario.supersededBy) {
        supersessions.push({ where: `${path}:${scenario.line}`, target: scenario.supersededBy })
      } else {
        scenarios.push({ id: scenario.id, path: `${path}:${scenario.line}` })
      }
    }
    examples.push(
      ...result.exampleIds.map((example) => ({
        id: example.id,
        path: `${path}:${example.line}`,
      })),
    )
    standards.push(
      ...result.standardIds.map((standard) => ({
        id: standard.id,
        path: `${path}:${standard.line}`,
      })),
    )
    if (options.verbose && result.findings.length === 0) {
      lines.push(`ok  ${path} (${result.scenarioIds.length} normative scenario id(s))`)
    }
  }
  for (const supersession of supersessions) {
    if (!seen.has(supersession.target)) {
      failed = true
      lines.push(`${supersession.where}: superseding ${supersession.target} does not exist`)
    }
  }

  const sources: string[] = []
  for (const tree of PRODUCT_TREES) {
    try {
      const paths = await snapshot.files(tree, EXCLUDED_DIRECTORIES)
      sources.push(
        ...(await Promise.all(
          paths.filter((path) => TEST_FILE.test(path)).map((path) => snapshot.read(path)),
        )),
      )
    } catch {
      // 製品 tree を持たない最小 fixture では、引用集合を空とする。
    }
  }
  const cited = citedNormativeIds(
    sources,
    [...standards, ...examples].map((declaration) => declaration.id),
  )
  const coverage = checkNormativeCoverage({ declared: [...standards, ...examples], cited })
  failed ||= coverage.length > 0
  lines.push(...coverage.map((finding) => `${finding.path}: ${finding.message}`))
  if (failed) return { ok: false, lines }
  return {
    ok: true,
    lines: [
      ...lines,
      ...(options.verbose ? [] : [`ok  ${paths.length} canonical document(s)`]),
      `ok  normative coverage (${standards.length} standard(s), ${scenarios.length} rule(s), ` +
        `${examples.length} example(s), ${cited.size} id(s) named by a test)`,
    ],
  }
}
