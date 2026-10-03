import { posix } from 'node:path'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { markdownAnchors } from './markdown-links.ts'
import { repositoryGoFiles } from './repository-inputs.ts'
import type { CheckOutcome } from './runner.ts'
import {
  featureSlices,
  goDeclarations,
  type FeatureNodeDebt,
  type Finding,
  verifyFeatureNodes,
  verifyRuleFields,
  verifySectionOrder,
  verifySpecificationOutline,
  verifySpecificationRuleFields,
} from './specification-rules.ts'
import { documentKind } from './specification-doc.ts'

/** 対応のない機能スライスの基準。導入時点の一覧で、減る方向にしか動かさない。 */
const FEATURE_NODE_DEBT = 'tools/check/feature-node-debt.json'

export async function checkSpecificationRules(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const domainFiles = await snapshot.files('docs/domain', [])
  // 規則を宣言する文書。システム、Context、機能ノードの三段にある `scenarios.feature.md` と、
  // 新しい形式の機能仕様である。
  const specificationPaths = new Set(
    domainFiles.filter((path) => documentKind(path) === 'specification'),
  )
  const scenarioPaths = domainFiles
    .filter(
      (path) =>
        /^docs\/domain\/(?:[^/]+\/){0,2}scenarios\.feature\.md$/.test(path) ||
        specificationPaths.has(path),
    )
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
    findings.push(...verifySectionOrder(path, source))
    if (specificationPaths.has(path) && inFeatureContext(path)) {
      findings.push(...verifySpecificationRuleFields(path, source))
      const directory = posix.dirname(path)
      if (path.endsWith('/README.md') && !parents.has(directory)) {
        findings.push(...verifySpecificationOutline(path, source))
      }
    }
  }

  const debt: FeatureNodeDebt = snapshot.exists(FEATURE_NODE_DEBT)
    ? JSON.parse(await snapshot.read(FEATURE_NODE_DEBT))
    : { contextAliases: {}, unmappedSlices: [] }
  const backendDirectories = (await snapshot.files('backend', ['vendor', 'dist', 'build'])).map(
    (path) => posix.dirname(path),
  )
  const nodes = new Set(
    domainFiles.flatMap((path) => {
      const node = posix.dirname(path)
      if (/^docs\/domain\/[^/]+\/[^/]+$/.test(node)) return [node]
      // 新しい形式では、機能群の一段下も機能ノードになる。内部設計の段は機能ではない。
      return inFeatureContext(path) &&
        /^docs\/domain\/[^/]+\/[^/]+\/[^/]+$/.test(node) &&
        node.split('/')[3] !== 'design'
        ? [node]
        : []
    }),
  )
  findings.push(...verifyFeatureNodes(featureSlices([...new Set(backendDirectories)]), nodes, debt))

  return {
    ok: findings.length === 0,
    lines:
      findings.length > 0
        ? findings.map((finding) => `${finding.path}:${finding.line}: ${finding.message}`)
        : [`ok  specification rules (${scenarioPaths.length} scenario document(s))`],
  }
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
