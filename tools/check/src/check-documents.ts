import type { Dirent } from 'node:fs'
import {
  canonicalDocumentNames,
  CONTEXT_DOCUMENTS,
  FREELY_NAMED_DOCUMENT_DIRECTORIES,
  type DirectoryListing,
} from '../../workspace/src/document-layout.ts'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { verifyCanonicalDocumentSet } from './canonical-document-set.ts'
import {
  checkNormativeCoverage,
  citedNormativeIds,
  type DebtEntry,
  type DeclaredId,
  type SourceFile,
} from './normative-coverage.ts'
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
      if (!entry.isDirectory()) continue
      const child = `${directory}/${entry.name}`
      if (!FREELY_NAMED_DOCUMENT_DIRECTORIES.has(child)) pending.push(child)
    }
  }
  try {
    for (const entry of await snapshot.list('docs/contexts')) {
      if (!entry.isDirectory()) continue
      const directory = `docs/contexts/${entry.name}`
      listings.push({
        directory,
        files: (await snapshot.list(directory))
          .filter((item) => item.isFile())
          .map((item) => item.name),
      })
    }
  } catch {
    // Context がまだ無い最小 workspace も、残りの文書を検査する。
  }
  return listings
}

type DebtFile = { untested: DebtEntry[] }

async function readDebt(snapshot: WorkspaceSnapshot, path: string): Promise<DebtEntry[]> {
  if (!snapshot.exists(path)) return []
  const parsed = JSON.parse(await snapshot.read(path)) as DebtFile
  if (!Array.isArray(parsed.untested)) throw new Error(`${path}: untested must be an array`)
  return parsed.untested.map((entry) => {
    if (typeof entry?.id !== 'string' || typeof entry?.reason !== 'string') {
      throw new Error(`${path}: every untested entry needs an id and a reason`)
    }
    return entry
  })
}

/**
 * The work item names a ledger row's `blocked_by` may resolve to.
 *
 * `work-items/done/` counts: a decision that has already been taken is still
 * the record a blocked row points at, and dropping it the moment the item
 * completes would turn every settled pointer into a failure.
 *
 * A workspace with no `work-items/` returns undefined rather than an empty set,
 * which switches the existence rule off instead of failing every row. Minimal
 * fixtures have no records to resolve against.
 */
async function knownWorkItemNames(
  snapshot: WorkspaceSnapshot,
): Promise<ReadonlySet<string> | undefined> {
  try {
    const paths = await snapshot.files('work-items', EXCLUDED_DIRECTORIES)
    const names = paths
      .filter((path) => path.endsWith('.md'))
      .map((path) => (path.split('/').pop() ?? '').replace(/\.md$/, ''))
      .filter((name) => name.startsWith('wi-'))
    return names.length === 0 ? undefined : new Set(names)
  } catch {
    return undefined
  }
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
    .filter((listing) => listing.directory.startsWith('docs/contexts/'))
    .map((listing) => listing.directory.slice('docs/contexts/'.length))
  if (snapshot.exists('docs/architecture/logical.md')) {
    const classifications = verifySubdomainClassification(
      await snapshot.read('docs/architecture/logical.md'),
      contextDirectories,
    )
    failed ||= classifications.length > 0
    lines.push(
      ...classifications.map(
        (finding) => `fail  docs/architecture/logical.md:${finding.line}: ${finding.message}`,
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

  // path を一緒に運ぶのは、指摘が「どこが名指しているか」を言えるようにするため
  // である。読み手が rg で探し直すところから始めなくて済む。
  const sources: SourceFile[] = []
  for (const tree of PRODUCT_TREES) {
    try {
      const paths = await snapshot.files(tree, EXCLUDED_DIRECTORIES)
      sources.push(
        ...(await Promise.all(
          paths
            .filter((path) => TEST_FILE.test(path))
            .map(async (path) => ({ path, source: await snapshot.read(path) })),
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
  const examplesDebt = 'tools/check/example-coverage-debt.json'
  const coverage = [
    // 標準の側は台帳を持たない (wi-495)。宣言した行は、その id を名指すテストを
    // 持つか、検査に落ちるかのどちらかである。
    ...checkNormativeCoverage({ declared: standards, cited }),
    ...checkNormativeCoverage({
      declared: examples,
      cited,
      ledger: { entries: await readDebt(snapshot, examplesDebt), path: examplesDebt },
      knownWorkItems: await knownWorkItemNames(snapshot),
    }),
  ]
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
