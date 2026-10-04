import { posix } from 'node:path'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { markdownAnchors } from './markdown-links.ts'
import { repositoryGoFiles } from './repository-inputs.ts'
import type { CheckOutcome } from './runner.ts'
import {
  codeSlices,
  goDeclarations,
  type FeatureSliceDebt,
  type Finding,
  verifyEarsStatements,
  verifyFeatureSliceSpecifications,
  verifyRuleFields,
  verifySpecificationOutline,
  verifySpecificationRuleFields,
} from './specification-rules.ts'
import { documentKind } from './specification-doc.ts'

/** 仕様のディレクトリと対応しないコードのディレクトリの基準。導入時点の一覧で、減る方向にしか動かさない。 */
const FEATURE_SLICE_DEBT = 'tools/check/feature-slice-debt.json'

/** Context をまたぐ要件を宣言する文書。 */
const SYSTEM_SCENARIOS = 'docs/domain/scenarios.feature.md'

/**
 * 要件文の EARS の構文を確かめる Context。要件文を書き直した Context から加え、
 * 減らす方向には動かさない。
 */
const EARS_CONTEXTS = new Set<string>([
  'tenancy',
  'identity-management',
  'claim-mapping',
  'api-tokens',
  'ws-federation',
  'data-keys',
  'audit',
  'sourcing',
  'saml',
  'seeding',
  'authorization',
  'workloadidentity',
  'sharedsignals',
  'signing-keys',
  'application',
  'identity-governance',
  'jobs',
  'provisioning',
  'system',
  'authentication',
  'oauth2',
])

export async function checkSpecificationRules(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const domainFiles = await snapshot.files('docs/domain', [])
  // 規則を宣言する文書。システムの `scenarios.feature.md` と、機能仕様である。
  const specificationPaths = new Set(
    domainFiles.filter((path) => documentKind(path) === 'specification'),
  )
  const scenarioPaths = domainFiles
    .filter((path) => path === SYSTEM_SCENARIOS || specificationPaths.has(path))
    .sort()
  const declarations = goDeclarations(await repositoryGoFiles(snapshot))
  const anchors = new Map<string, Set<string>>()
  const resolveLink = (from: string, target: string): boolean => {
    const [file = '', anchor] = target.split('#').map(decoded)
    if (file === undefined) return false
    const path = file === '' ? from : posix.normalize(posix.join(posix.dirname(from), file))
    if (!snapshot.exists(path)) return false
    if (anchor === undefined) return !target.includes('#')
    return anchors.get(path)?.has(anchor) ?? false
  }
  for (const path of await markdownTargets(snapshot, scenarioPaths)) {
    anchors.set(path, markdownAnchors(await snapshot.read(path)))
  }

  const featureContexts = new Set(
    domainFiles.flatMap(
      (path) => path.match(/^docs\/domain\/([^/]+)\/design\/README\.md$/)?.[1] ?? [],
    ),
  )
  const parents = new Set(domainFiles.map((path) => posix.dirname(posix.dirname(path))))
  const inFeatureContext = (path: string) => featureContexts.has(path.split('/')[2] ?? '')

  const findings: Finding[] = []
  for (const path of scenarioPaths) {
    const source = await snapshot.read(path)
    findings.push(...verifyRuleFields(path, source, declarations, resolveLink))
    if (specificationPaths.has(path) && inFeatureContext(path)) {
      findings.push(...verifySpecificationRuleFields(path, source))
      const context = path.split('/')[2] ?? ''
      if (EARS_CONTEXTS.has(context)) {
        findings.push(...verifyEarsStatements(path, source, await responder(snapshot, context)))
      }
      const directory = posix.dirname(path)
      if (path.endsWith('/README.md') && !parents.has(directory)) {
        findings.push(...verifySpecificationOutline(path, source))
      }
    }
  }

  const debt: FeatureSliceDebt = snapshot.exists(FEATURE_SLICE_DEBT)
    ? JSON.parse(await snapshot.read(FEATURE_SLICE_DEBT))
    : { contextAliases: {}, unmappedSlices: [] }
  const backendDirectories = (await snapshot.files('backend', ['vendor', 'dist', 'build'])).map(
    (path) => posix.dirname(path),
  )
  const specifications = new Set(
    domainFiles.flatMap((path) => {
      const directory = posix.dirname(path)
      if (/^docs\/domain\/[^/]+\/[^/]+$/.test(directory)) return [directory]
      // 機能群の一段下も機能スライスの仕様になる。内部設計の段は機能ではない。
      return inFeatureContext(path) &&
        /^docs\/domain\/[^/]+\/[^/]+\/[^/]+$/.test(directory) &&
        directory.split('/')[3] !== 'design'
        ? [directory]
        : []
    }),
  )
  findings.push(
    ...verifyFeatureSliceSpecifications(
      codeSlices([...new Set(backendDirectories)]),
      specifications,
      debt,
    ),
  )

  return {
    ok: findings.length === 0,
    lines:
      findings.length > 0
        ? findings.map((finding) => `${finding.path}:${finding.line}: ${finding.message}`)
        : [`ok  specification rules (${scenarioPaths.length} scenario document(s))`],
  }
}

/** 要件文の主体。Context の `README.md` の H1 を使う。 */
async function responder(snapshot: WorkspaceSnapshot, context: string): Promise<string> {
  const readme = await snapshot.read(`docs/domain/${context}/README.md`)
  return readme.match(/^# (.+)$/m)?.[1]?.trim() ?? context
}

function decoded(value: string): string | undefined {
  try {
    return decodeURIComponent(value)
  } catch {
    return undefined
  }
}

/** 上位の要件のリンクが指し得る文書。アンカーを読むのはリンク先の Markdown だけでよい。 */
async function markdownTargets(snapshot: WorkspaceSnapshot, from: string[]): Promise<string[]> {
  const targets = new Set<string>()
  for (const path of from) {
    const source = await snapshot.read(path)
    for (const match of source.matchAll(/\]\(([^)#\s]+\.md)(?:#[^)\s]*)?\)/g)) {
      const file = decoded(match[1] ?? '')
      if (file === undefined) continue
      const target = posix.normalize(posix.join(posix.dirname(path), file))
      if (snapshot.exists(target)) targets.add(target)
    }
  }
  return [...targets]
}
