/**
 * 論理アーキテクチャへ生成する、モジュール間の依存の行列を作る。
 *
 * コードの参照は境界検査から得る。ラベルは人が表に記したモデルの関係として保持し、
 * コードの参照に現れない関係と負債を記号で区別し、責務表の区分（H3）の順に並べる。
 */

import { isResponsibilityHeader, type ModuleDependency } from './boundary-fitness.ts'
import { tableRowCells } from './markdown-table.ts'

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

/** 行が利用元、列が利用先の Markdown の行列。列の番号は行の番号に対応する。 */
export function moduleDependencyMatrix(
  groups: readonly ModuleGroup[],
  dependencies: readonly ModuleDependency[],
  markdown = '',
  debt: readonly ModuleDependency[] = [],
): string {
  const key = ({ sourceModule, targetModule }: ModuleDependency) =>
    `${sourceModule}:${targetModule}`
  const imports = new Set(dependencies.map(key))
  const edges = new Set(imports)
  const debts = new Set(debt.map(key))
  const modules = new Set(groups.flatMap((group) => group.modules))
  const labels = new Map<string, string>()
  let inRelationships = false
  for (const line of markdown.split(/\r?\n/)) {
    if (line.trim() === '| 利用元 | 利用先 | 関係 |') {
      inRelationships = true
      continue
    }
    if (!line.trim().startsWith('|')) {
      inRelationships = false
      continue
    }
    if (!inRelationships) continue
    const [sourceModule, targetModule, label] = tableRowCells(line)
    if (sourceModule && /^:?-+:?$/.test(sourceModule)) continue
    if (!sourceModule || !targetModule || !label) {
      throw new Error('Module relationship needs a source, target, and label')
    }
    if (!modules.has(sourceModule) || !modules.has(targetModule)) {
      throw new Error(`Unknown module in relationship: ${sourceModule} -> ${targetModule}`)
    }
    const edge = { sourceModule, targetModule }
    if (labels.has(key(edge))) throw new Error(`Duplicate module relationship: ${key(edge)}`)
    labels.set(key(edge), label)
    edges.add(key(edge))
  }
  const names = [...modules]
  const lines = [
    `| 利用元＼利用先 | ${names.map((_, index) => index + 1).join(' | ')} |`,
    `| --- | ${names.map(() => '---').join(' | ')} |`,
  ]
  for (const [index, sourceModule] of names.entries()) {
    const cells = names.map((targetModule) => {
      const edgeKey = key({ sourceModule, targetModule })
      if (!edges.has(edgeKey)) return '—'
      const isDebt = imports.has(edgeKey) && debts.has(edgeKey)
      const symbol = isDebt ? '!' : imports.has(edgeKey) ? '●' : '◇'
      const kind = isDebt ? '境界の負債' : imports.has(edgeKey) ? 'import' : 'HTTP またはポート'
      const relationship = labels.get(edgeKey)
      const title = `${sourceModule} → ${targetModule}：${kind}${relationship ? `、${relationship}` : ''}`
      return `[${symbol}](../../modules/README.md#モジュールの間の依存 "${linkTitle(title)}")`
    })
    lines.push(`| ${index + 1} ${sourceModule} | ${cells.join(' | ')} |`)
  }
  return lines.join('\n')
}

function linkTitle(label: string): string {
  return label
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('|', '&#124;')
}
