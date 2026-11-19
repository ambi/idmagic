/**
 * 論理アーキテクチャへ生成する、モジュールの Context Map を作る。
 *
 * 節点はモジュールであり、辺の分類だけを DDD の Context Map から借りる。
 * コードの参照は境界検査から得て、関係の種類とラベルは人が表に宣言したものを使う。
 * 26 の節点を一枚に描くと読めないので、責務表の区分（H3）を節点にした全体図と、
 * 区分ごとの図の二段に分ける。
 */

import { isResponsibilityHeader, type ModuleDependency } from './boundary-fitness.ts'
import { tableRowCells } from './markdown-table.ts'

type ModuleGroup = { name: string; modules: string[] }

/** 関係の種類。順序は全体図で本数を並べる順でもある。 */
const RELATIONSHIP_KINDS = ['使う', '変換', '共同管理', '公開契約'] as const
type RelationshipKind = (typeof RELATIONSHIP_KINDS)[number]

/** 種類ごとの線。変換は太線、公開契約は点線、共同管理は双方向にする。 */
const ARROWS: Record<RelationshipKind, string> = {
  使う: '-->',
  変換: '==>',
  共同管理: '<-->',
  公開契約: '-.->',
}

const RELATIONSHIP_HEADER = '| 利用元 | 利用先 | 関係の種類 | 関係 |'

type Edge = ModuleDependency & { kind: RelationshipKind; label?: string; debt: boolean }

/** 責務表の行を、直前の H3 の区分ごとに集める。組み立て地点の行は図に含めない。 */
export function moduleGroups(markdown: string, composition: string): ModuleGroup[] {
  const groups: ModuleGroup[] = []
  let heading = ''
  let inTable = false
  for (const line of markdown.split(/\r?\n/)) {
    const h3 = line.match(/^### (.+)$/)?.[1]?.trim()
    if (h3 !== undefined) heading = h3
    if (isResponsibilityHeader(line)) {
      inTable = true
      continue
    }
    if (!line.trim().startsWith('|')) {
      inTable = false
      continue
    }
    if (!inTable) continue
    const name = line.match(/^\|\s*\[([^\]]+)\]/)?.[1]?.trim()
    if (!name || name === composition) continue
    let group = groups.find((one) => one.name === heading)
    if (!group) {
      group = { name: heading, modules: [] }
      groups.push(group)
    }
    group.modules.push(name)
  }
  return groups
}

/**
 * 全体図と区分ごとの図を、見出し付きの Mermaid の Markdown として返す。
 * 表に宣言のない import は、提供側の公開型をそのまま使う関係として描く。
 */
export function moduleContextMap(
  groups: readonly ModuleGroup[],
  dependencies: readonly ModuleDependency[],
  markdown = '',
  debt: readonly ModuleDependency[] = [],
): string {
  const edges = contextMapEdges(groups, dependencies, markdown, debt)
  const sections = [
    ['### 全体', overviewDiagram(groups, edges)],
    ...groups.map((group) => [`### ${group.name}`, groupDiagram(group, edges)]),
  ]
  return sections
    .map(([heading, lines]) => `${heading}\n\n\`\`\`mermaid\n${lines}\n\`\`\``)
    .join('\n\n')
}

function contextMapEdges(
  groups: readonly ModuleGroup[],
  dependencies: readonly ModuleDependency[],
  markdown: string,
  debt: readonly ModuleDependency[],
): Edge[] {
  const key = ({ sourceModule, targetModule }: ModuleDependency) =>
    `${sourceModule}:${targetModule}`
  const imports = new Set(dependencies.map(key))
  const debts = new Set(debt.map(key))
  const modules = new Set(groups.flatMap((group) => group.modules))
  const declared = new Map<string, Edge>()
  let inRelationships = false
  for (const line of markdown.split(/\r?\n/)) {
    if (line.trim() === RELATIONSHIP_HEADER) {
      inRelationships = true
      continue
    }
    if (!line.trim().startsWith('|')) {
      inRelationships = false
      continue
    }
    if (!inRelationships) continue
    const [sourceModule, targetModule, kind, label] = tableRowCells(line)
    if (sourceModule && /^:?-+:?$/.test(sourceModule)) continue
    if (!sourceModule || !targetModule || !kind || !label) {
      throw new Error('Module relationship needs a source, target, kind, and label')
    }
    if (!modules.has(sourceModule) || !modules.has(targetModule)) {
      throw new Error(`Unknown module in relationship: ${sourceModule} -> ${targetModule}`)
    }
    if (!(RELATIONSHIP_KINDS as readonly string[]).includes(kind)) {
      throw new Error(
        `Unknown relationship kind ${kind} for ${sourceModule} -> ${targetModule}; use one of ${RELATIONSHIP_KINDS.join(', ')}`,
      )
    }
    const edgeKey = key({ sourceModule, targetModule })
    if (declared.has(edgeKey)) throw new Error(`Duplicate module relationship: ${edgeKey}`)
    if (kind === '公開契約' && imports.has(edgeKey)) {
      throw new Error(
        `${sourceModule} -> ${targetModule} is declared as a public contract but imports Go packages`,
      )
    }
    declared.set(edgeKey, {
      sourceModule,
      targetModule,
      kind: kind as RelationshipKind,
      label,
      debt: imports.has(edgeKey) && debts.has(edgeKey),
    })
  }
  for (const dependency of dependencies) {
    const edgeKey = key(dependency)
    if (declared.has(edgeKey)) continue
    if (!modules.has(dependency.sourceModule) || !modules.has(dependency.targetModule)) continue
    declared.set(edgeKey, { ...dependency, kind: '使う', debt: debts.has(edgeKey) })
  }
  return [...declared.values()]
}

function overviewDiagram(groups: readonly ModuleGroup[], edges: readonly Edge[]): string {
  const groupOf = new Map<string, number>()
  groups.forEach((group, index) => {
    for (const module of group.modules) groupOf.set(module, index + 1)
  })
  const lines = [
    'flowchart LR',
    ...groups.map(
      (group, index) =>
        `  g${index + 1}["${mermaidText(group.name)}<br/>${group.modules.length} モジュール"]`,
    ),
  ]
  const counts = new Map<string, Map<RelationshipKind, number>>()
  for (const edge of edges) {
    const from = groupOf.get(edge.sourceModule)
    const to = groupOf.get(edge.targetModule)
    if (from === undefined || to === undefined || from === to) continue
    const pair = `g${from} g${to}`
    const kinds = counts.get(pair) ?? new Map<RelationshipKind, number>()
    kinds.set(edge.kind, (kinds.get(edge.kind) ?? 0) + 1)
    counts.set(pair, kinds)
  }
  for (const [pair, kinds] of counts) {
    const [from, to] = pair.split(' ')
    const label = RELATIONSHIP_KINDS.filter((kind) => kinds.has(kind))
      .map((kind) => `${kind} ${kinds.get(kind)}`)
      .join('、')
    lines.push(`  ${from} -->|"${label}"| ${to}`)
  }
  return lines.join('\n')
}

function groupDiagram(group: ModuleGroup, edges: readonly Edge[]): string {
  const inside = new Set(group.modules)
  const relevant = edges.filter(
    (edge) => inside.has(edge.sourceModule) || inside.has(edge.targetModule),
  )
  const outside = [
    ...new Set(
      relevant
        .flatMap((edge) => [edge.sourceModule, edge.targetModule])
        .filter((module) => !inside.has(module)),
    ),
  ]
  const lines = ['flowchart LR', `  subgraph group["${mermaidText(group.name)}"]`]
  for (const module of group.modules) lines.push(`    ${nodeId(module)}["${mermaidText(module)}"]`)
  lines.push('  end')
  for (const module of outside)
    lines.push(`  ${nodeId(module)}["${mermaidText(module)}"]:::outside`)
  for (const edge of relevant) {
    const text = `${edge.kind}${edge.label ? `：${edge.label}` : ''}${edge.debt ? '（境界の負債）' : ''}`
    lines.push(
      `  ${nodeId(edge.sourceModule)} ${ARROWS[edge.kind]}|"${mermaidText(text)}"| ${nodeId(edge.targetModule)}`,
    )
  }
  lines.push('  classDef outside stroke-dasharray:4 3,opacity:0.7')
  return lines.join('\n')
}

function nodeId(module: string): string {
  return `m_${module.replace(/[^A-Za-z0-9]/g, '_')}`
}

function mermaidText(text: string): string {
  return text.replaceAll('"', '#quot;').replaceAll('<', '#lt;').replaceAll('>', '#gt;')
}
