import { posix } from 'node:path'
import MarkdownIt from 'markdown-it'
import { markdownAnchors } from '../../check/src/markdown-links.ts'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'

export type ContextSeed = {
  kind: string
  targets: readonly string[]
  concerns: readonly string[]
  /** `spec_impact: none` の変更では false。契約の concern がない限り TypeSpec を候補にしない。 */
  specification?: boolean
}

export type ContextCandidate = {
  location: string
  origin: string
  reason: string
  basis: 'explicit' | 'inferred'
}

export type ContextPlan = {
  reading: ContextCandidate[]
  tasks: ContextCandidate[]
  warnings: string[]
}

export async function deriveContext(
  seed: ContextSeed,
  workspace: WorkspaceSnapshot,
): Promise<ContextPlan> {
  const plan: ContextPlan = { reading: [], tasks: [], warnings: [] }
  const addReading = async (candidate: ContextCandidate) => {
    const [path, anchor] = candidate.location.split('#')
    if (!path || !workspace.exists(path)) {
      plan.warnings.push(`Missing candidate: ${candidate.location} (origin: ${candidate.origin})`)
      return
    }
    if (anchor && !markdownAnchors(await workspace.read(path)).has(decodeURIComponent(anchor))) {
      plan.warnings.push(`Missing section: ${candidate.location} (origin: ${candidate.origin})`)
      return
    }
    if (!plan.reading.some((one) => one.location === candidate.location))
      plan.reading.push(candidate)
  }

  if (workspace.exists('AGENTS.md')) {
    for (const row of markdownRows(await workspace.read('AGENTS.md'))) {
      const scope = row.cells
        .map((cell) => cell.content)
        .join(' ')
        .match(/<!-- context: (.*?) -->/)
      if (!scope || !matchesScope(scope[1]!, seed)) continue
      for (const cell of row.cells) {
        for (const link of cell.links) {
          const location = localLink('AGENTS.md', link)
          if (location) {
            await addReading({
              location,
              origin: `AGENTS.md:${row.line}`,
              reason: row.cells[0]?.text ?? '',
              basis: 'explicit',
            })
          }
        }
      }
    }
  } else {
    plan.warnings.push('No AGENTS.md routing table; only normative references can be resolved.')
  }

  const miseSource = await workspace.read('mise.toml')
  const config = Bun.TOML.parse(miseSource) as { tasks?: Record<string, { description?: string }> }
  let task = ''
  for (const [index, line] of miseSource.split('\n').entries()) {
    const header = line.match(/^\[tasks\.([\w:-]+)\]$/)
    if (line.startsWith('[')) task = header?.[1] ?? ''
    const scope = line.match(/^# context: (.*)$/)
    if (!task || !scope || !matchesScope(scope[1]!, seed)) continue
    plan.tasks.push({
      location: task,
      origin: `mise.toml:${index + 1}`,
      reason: config.tasks?.[task]?.description ?? task,
      basis: 'explicit',
    })
  }

  // モジュール名の別名を新しい台帳へ写さず、既存の責務表のリンクと接頭辞を使う。
  const architecture = 'docs/design/architecture/logical.md'
  if (!workspace.exists(architecture)) return plan
  for (const row of markdownRows(await workspace.read(architecture))) {
    const link = row.cells[0]?.links[0]
    const entry = link ? localLink(architecture, link) : undefined
    if (!entry?.startsWith('docs/modules/')) continue
    const moduleDirectory = posix.dirname(entry)
    const specification = `spec/modules/${posix.basename(moduleDirectory)}`
    const prefixes = row.cells[3]?.text.split(',').map((one) => one.trim()) ?? []
    if (
      !seed.targets.some(
        (target) =>
          within(target, moduleDirectory) ||
          within(target, specification) ||
          prefixes.some((prefix) => within(target, prefix)),
      )
    )
      continue

    const origin = `${architecture}:${row.line}`
    const addModule = (location: string, reason: string) =>
      addReading({ location, origin, reason, basis: 'inferred' })
    await addModule(
      `${entry}#公開する契約`,
      'The responsibility table assigns this target to the module; read its public contract.',
    )
    if (workspace.exists(`${moduleDirectory}/quality.md`)) {
      await addModule(
        `${moduleDirectory}/quality.md`,
        'Quality requirements allocated to the target module.',
      )
    }
    // 設計領域の索引と設計の記録は、モジュールの README の最後の節「設計」が並べる。
    const index = entry
    const source = workspace.exists(index) ? await workspace.read(index) : ''
    const designLine = source.split('\n').indexOf('## 設計') + 1
    if (designLine > 0) {
      await addModule(
        `${index}#設計`,
        'Shared design and known risks are linked by this module index; expand only the relevant links.',
      )
      if (seed.concerns.includes('effects')) {
        const designRows = markdownRows(source).filter((row) => row.line > designLine)
        for (const designRow of designRows) {
          const description = designRow.cells.map((cell) => cell.text).join(' ')
          if (!/確定|通知|監査|イベント|トランザクション/.test(description)) continue
          for (const cell of designRow.cells) {
            for (const designLink of cell.links) {
              const location = localLink(index, designLink)
              if (location)
                await addReading({
                  location,
                  origin: `${index}:${designRow.line}`,
                  reason: description,
                  basis: 'inferred',
                })
            }
          }
        }
      }
    }
    if (
      seed.concerns.includes('boundary') &&
      workspace.exists(`${moduleDirectory}/design/risks.md`)
    ) {
      await addModule(
        `${moduleDirectory}/design/risks.md`,
        'Known module risks may constrain a boundary change.',
      )
    }
    if (
      workspace.exists(specification) &&
      (seed.specification !== false || seed.concerns.includes('contract'))
    ) {
      for (const file of await workspace.files(specification)) {
        if (file.endsWith('.tsp'))
          await addModule(
            file,
            'TypeSpec candidate inferred from the canonical module directory; not proof of symbol ownership.',
          )
      }
    }
  }
  return plan
}

type MarkdownCell = { content: string; text: string; links: string[] }
type MarkdownRow = { cells: MarkdownCell[]; line: number }

const markdown = new MarkdownIt({ html: true })

/** Markdown の構造に従うので、コード例の中の表やリンクを候補に数えない。 */
function markdownRows(source: string): MarkdownRow[] {
  const rows: MarkdownRow[] = []
  let row: MarkdownRow | undefined
  for (const token of markdown.parse(source, {})) {
    if (token.type === 'tr_open') row = { cells: [], line: (token.map?.[0] ?? 0) + 1 }
    if (token.type === 'tr_close' && row) {
      rows.push(row)
      row = undefined
    }
    if (token.type !== 'inline' || !row) continue
    const children = token.children ?? []
    row.cells.push({
      content: token.content,
      text: children
        .filter((child) => child.type === 'text' || child.type === 'code_inline')
        .map((child) => child.content)
        .join('')
        .trim(),
      links: children
        .filter((child) => child.type === 'link_open')
        .map((child) => String(child.attrGet('href') ?? '')),
    })
  }
  return rows
}

function within(path: string, prefix: string): boolean {
  return path === prefix || path.startsWith(`${prefix}/`)
}

function localLink(origin: string, target: string): string | undefined {
  if (/^[a-z][a-z\d+.-]*:/i.test(target) || target.startsWith('/')) return undefined
  const [path, encodedAnchor] = target.split('#')
  const anchor = encodedAnchor ? decodeURIComponent(encodedAnchor) : undefined
  const resolved = path ? posix.normalize(posix.join(posix.dirname(origin), path)) : origin
  if (resolved === '..' || resolved.startsWith('../')) return undefined
  return `${resolved}${anchor ? `#${anchor}` : ''}`
}

/** 値内は OR、指定したキー間は AND。条件の内容は一次情報側が所有する。 */
function matchesScope(scope: string, seed: ContextSeed): boolean {
  const selectors = scope.trim().split(/ +/)
  return selectors.every((selector) => {
    const [key, value] = selector.split('=')
    if (!value || !['kinds', 'targets', 'concerns'].includes(key ?? ''))
      throw new Error(`Invalid context selector: ${selector}`)
    const alternatives = value.split(',')
    if (alternatives.includes('*')) return true
    if (key === 'kinds') return alternatives.includes(seed.kind)
    if (key === 'concerns') return alternatives.some((one) => seed.concerns.includes(one))
    return alternatives.some((prefix) => seed.targets.some((target) => within(target, prefix)))
  })
}

export type ReadingSize = { bytes: number; children: { anchor: string; bytes: number }[] }

/**
 * 候補を開いたときに読む量を測る。
 * 節の候補は最初の子見出しまでを本体とし、子節は本体と別に名前と量を並べる。
 * 子節は変更に当てはまるものだけを読むので、合計には本体だけを数える。
 */
export async function measureCandidates(
  plan: ContextPlan,
  workspace: WorkspaceSnapshot,
): Promise<Map<string, ReadingSize>> {
  const sizes = new Map<string, ReadingSize>()
  for (const candidate of plan.reading) {
    const [path, anchor] = candidate.location.split('#')
    const source = await workspace.read(path!)
    if (!anchor) {
      sizes.set(candidate.location, { bytes: Buffer.byteLength(source), children: [] })
      continue
    }
    const lines = source.split('\n')
    const headings = markdown
      .parse(source, {})
      .filter((token) => token.type === 'heading_open')
      .map((token) => ({
        level: Number(token.tag.slice(1)),
        line: token.map![0],
        anchor: [...markdownAnchors(lines[token.map![0]]!)][0] ?? '',
      }))
    const index = headings.findIndex((heading) => markdownAnchors(lines[heading.line]!).has(anchor))
    const heading = headings[index]!
    const following = headings.slice(index + 1)
    const end = following.find((one) => one.level <= heading.level)?.line ?? lines.length
    const children = following.filter((one) => one.line < end && one.level === heading.level + 1)
    const span = (from: number, to: number) => Buffer.byteLength(lines.slice(from, to).join('\n'))
    sizes.set(candidate.location, {
      bytes: span(heading.line, children[0]?.line ?? end),
      children: children.map((child, position) => ({
        anchor: child.anchor,
        bytes: span(child.line, children[position + 1]?.line ?? end),
      })),
    })
  }
  return sizes
}

const kilobytes = (bytes: number) => `${Math.max(1, Math.round(bytes / 1024))} KB`

export function renderContext(plan: ContextPlan, sizes?: Map<string, ReadingSize>): string[] {
  const lines = [
    '## Task context candidates',
    '',
    'Explicit references and inferred candidates are reading aids, not evidence of implementation or verification.',
    '',
  ]
  for (const candidate of plan.reading) {
    const size = sizes?.get(candidate.location)
    lines.push(
      `- ${candidate.location} [${candidate.basis}]${size ? ` ${kilobytes(size.bytes)}` : ''}; origin: ${candidate.origin}; reason: ${candidate.reason}`,
    )
    for (const child of size?.children ?? [])
      lines.push(`  - #${child.anchor} ${kilobytes(child.bytes)}`)
  }
  if (sizes) {
    const total = [...sizes.values()].reduce((sum, size) => sum + size.bytes, 0)
    lines.push(
      '',
      `Lead sections total ${kilobytes(total)}. Open a child section only when it applies to this change.`,
    )
  }
  lines.push('', '## Verification task candidates', '')
  for (const candidate of plan.tasks) {
    lines.push(
      `- mise run ${candidate.location}; origin: ${candidate.origin}; reason: ${candidate.reason}`,
    )
  }
  lines.push(
    '',
    ...plan.warnings.map((one) => `Warning: ${one}`),
    '',
    'Expand with --include history or --include neighbors; inferred neighbors may be unrelated.',
    '',
  )
  return lines
}

export async function neighborCandidates(
  plan: ContextPlan,
  workspace: WorkspaceSnapshot,
): Promise<ContextCandidate[]> {
  const architecture = 'docs/design/architecture/logical.md'
  if (!workspace.exists(architecture)) return []
  const rows = markdownRows(await workspace.read(architecture))
  const neighbors = new Map<string, ContextCandidate>()
  const owners = new Set(plan.reading.map((one) => one.location.split('#')[0]))
  for (const contract of plan.reading.filter((one) =>
    one.location.endsWith('/README.md#公開する契約'),
  )) {
    const file = contract.location.split('#')[0]!
    const source = await workspace.read(file)
    for (const row of rows) {
      const name = row.cells[0]?.text
      const link = row.cells[0]?.links[0]
      const location = link ? localLink(architecture, link) : undefined
      if (!name || !location || owners.has(location) || !source.includes(`\`${name}\``)) continue
      neighbors.set(location, {
        location,
        origin: contract.location,
        reason: `The target module names ${name}; this mention may not describe the changed dependency.`,
        basis: 'inferred',
      })
    }
  }
  return [...neighbors.values()]
}
