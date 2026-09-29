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
} from './specification-rules.ts'

/** 対応のない機能スライスの基準。導入時点の一覧で、減る方向にしか動かさない。 */
const FEATURE_NODE_DEBT = 'tools/check/feature-node-debt.json'

export async function checkSpecificationRules(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const domainFiles = await snapshot.files('docs/domain', [])
  // システム、Context、機能ノードの三段にある `scenarios.feature.md`。
  const scenarioPaths = domainFiles
    .filter((path) => /^docs\/domain\/(?:[^/]+\/){0,2}scenarios\.feature\.md$/.test(path))
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

  const findings: Finding[] = []
  for (const path of scenarioPaths) {
    const source = await snapshot.read(path)
    findings.push(...verifyRuleFields(path, source, declarations, resolveLink))
    findings.push(...verifySectionOrder(path, source))
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
      return /^docs\/domain\/[^/]+\/[^/]+$/.test(node) ? [node] : []
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

/** 上位の規則のリンクが指し得る文書。アンカーを読むのはリンク先の Markdown だけでよい。 */
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
