/**
 * モジュール設計文書の入口に置く、モジュール間の依存の図を作る。
 *
 * コードの参照は境界検査から得る。図のラベルは人が記したモデルの関係として保持し、
 * コードの参照がない関係は破線で描く。モジュールは責務表の区分（H3）でまとめる。
 */

import { isResponsibilityHeader, type ModuleDependency } from './boundary-fitness.ts'

export const MODULE_DEPENDENCY_DIAGRAM_START = '<!-- generated:module-dependencies:start -->'
export const MODULE_DEPENDENCY_DIAGRAM_END = '<!-- generated:module-dependencies:end -->'

type ModuleGroup = { name: string; modules: string[] }

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

/** 区分ごとの部分グラフと、モジュール間の辺からなる Mermaid の図。 */
export function moduleDependencyDiagram(
  groups: readonly ModuleGroup[],
  dependencies: readonly ModuleDependency[],
  markdown = '',
): string {
  const key = ({ sourceModule, targetModule }: ModuleDependency) =>
    `${sourceModule}:${targetModule}`
  const edges = new Map(dependencies.map((edge) => [key(edge), edge]))
  const imports = new Set(edges.keys())
  const modules = new Set(groups.flatMap((group) => group.modules))
  const labels = new Map<string, string>()
  const source =
    markdown.split(MODULE_DEPENDENCY_DIAGRAM_START)[1]?.split(MODULE_DEPENDENCY_DIAGRAM_END)[0] ??
    ''
  for (const [, sourceModule, label, targetModule] of source.matchAll(
    /^[ \t]+(\w+) (?:-->|-\.->)\|"([^"|\r\n]+)"\| (\w+)[ \t]*$/gm,
  )) {
    if (!modules.has(sourceModule!) || !modules.has(targetModule!)) {
      throw new Error(`Unknown module in relationship: ${sourceModule} -> ${targetModule}`)
    }
    const edge = { sourceModule: sourceModule!, targetModule: targetModule! }
    if (labels.has(key(edge))) throw new Error(`Duplicate module relationship: ${key(edge)}`)
    labels.set(key(edge), label!)
    edges.set(key(edge), edge)
  }
  const lines = ['```mermaid', 'flowchart LR']
  for (const [index, group] of groups.entries()) {
    lines.push(`  subgraph group${index}["${group.name}"]`)
    for (const module of group.modules) lines.push(`    ${module}`)
    lines.push('  end')
  }
  for (const edge of edges.values()) {
    const arrow = imports.has(key(edge)) ? '-->' : '-.->'
    const label = labels.get(key(edge))
    lines.push(`  ${edge.sourceModule} ${arrow}${label ? `|"${label}"|` : ''} ${edge.targetModule}`)
  }
  lines.push('```')
  return lines.join('\n')
}

/** 印の間を図で置き換えた文書。印がなければ undefined を返す。 */
export function withModuleDependencyDiagram(markdown: string, diagram: string): string | undefined {
  const start = markdown.indexOf(MODULE_DEPENDENCY_DIAGRAM_START)
  const end = markdown.indexOf(MODULE_DEPENDENCY_DIAGRAM_END)
  if (start < 0 || end < start) return undefined
  return (
    markdown.slice(0, start + MODULE_DEPENDENCY_DIAGRAM_START.length) +
    `\n\n${diagram}\n\n` +
    markdown.slice(end)
  )
}
