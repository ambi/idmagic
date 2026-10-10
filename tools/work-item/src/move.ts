import { mkdir, rename, writeFile } from 'node:fs/promises'
import { basename, dirname, posix } from 'node:path'
import MarkdownIt, { type Env, type Token } from 'markdown-it'
import { parseFrontmatterAndMarkdown } from '../../check/src/work-item-markdown.ts'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'

type Destination = { start: number; end: number; value: string }

const nativeLinkRules = ['link', 'image'].map((name) => {
  const parser = new MarkdownIt()
  parser.inline.ruler.enableOnly(name)
  return { name, rule: parser.inline.ruler.getRules('')[0]! }
})

/** Markdown の構文解析に委ね、コードとリンクを同じ文字列だからと一括置換しない。 */
export function relocateMarkdownLinks(
  source: string,
  from: string,
  to: string,
  moves: ReadonlyMap<string, string>,
): string {
  const markdown = new MarkdownIt()
  markdown.helpers = { ...markdown.helpers }
  markdown.core.ruler.disable('strip_references')
  const environment: Env = {}
  const tokens = markdown.parse(source, environment)
  const lines = source.split('\n')
  const offsets: number[] = []
  let offset = 0
  for (const line of lines) {
    offsets.push(offset)
    offset += line.length + 1
  }
  const edits = new Map<number, { end: number; value: string }>()
  const cursors = new Map<number, number>()
  let map: [number, number] = [0, lines.length]
  const relocate = (value: string): string | undefined => {
    if (/^(?:[a-z][a-z0-9+.-]*:|\/|#)/i.test(value)) return undefined
    const split = value.search(/[?#]/)
    const path = split < 0 ? value : value.slice(0, split)
    if (!path) return undefined
    let decoded: string
    try {
      decoded = decodeURIComponent(path)
    } catch {
      return undefined
    }
    const target = posix.normalize(posix.join(posix.dirname(from), decoded))
    const movedTarget = moves.get(target) ?? target
    const relative = posix.relative(posix.dirname(to), movedTarget)
    if (posix.dirname(from) === posix.dirname(to) && target === movedTarget) return undefined
    // 共通文書は active と done のどちらからも同じ相対位置になる。
    if (posix.normalize(decoded) === relative && target === movedTarget) return undefined
    return (
      relative.split('/').map(encodeURIComponent).join('/') + (split < 0 ? '' : value.slice(split))
    )
  }
  const recordEdit = (raw: string, value: string, range: [number, number], column?: number) => {
    const replacement = relocate(value)
    if (replacement === undefined) return
    for (let line = range[0]; line < range[1]; line++) {
      const start = column ?? lines[line]!.indexOf(raw, cursors.get(line) ?? 0)
      if (start < 0) continue
      if (column === undefined) cursors.set(line, start + raw.length)
      const angled = raw.startsWith('<')
      // 解析器はエスケープを外した値を返すので、書き戻すときにバックスラッシュも戻す。
      const rendered = angled
        ? `<${replacement.replace(/[\\<>]/g, '\\$&')}>`
        : replacement.replace(/[\\()]/g, '\\$&')
      edits.set(offsets[line]! + start, {
        end: offsets[line]! + start + raw.length,
        value: rendered,
      })
      return
    }
    throw new Error(`cannot locate Markdown destination in ${from}: ${raw}`)
  }
  for (const token of tokens) {
    if (token.map) map = token.map
    if (token.type === 'reference_definition') {
      const block = lines.slice(map[0], map[1]).join('\n')
      const marker = block.indexOf(']:')
      if (marker < 0) continue
      const start = marker + 2 + (block.slice(marker + 2).match(/^\s*/)?.[0].length ?? 0)
      const result = markdown.helpers.parseLinkDestination(block, start, block.length)
      if (result.ok) recordEdit(block.slice(start, result.pos), result.str, map)
    }
    if (token.type !== 'inline') continue
    const destinations: Destination[] = []
    for (const { name, rule } of nativeLinkRules) {
      markdown.inline.ruler.at(name, (state, silent) => {
        const start = destinations.length
        const tokenStart = state.tokens.length
        const accepted = rule(state, silent)
        const targets = new Set(
          state.tokens.slice(tokenStart).flatMap((child) => {
            const attribute =
              child.type === 'link_open'
                ? child.attrGet('href')
                : child.type === 'image'
                  ? child.attrGet('src')
                  : null
            return attribute === null ? [] : [String(attribute)]
          }),
        )
        const parsed = destinations.splice(start)
        if (accepted && !silent) {
          destinations.push(
            ...parsed.filter((one) => targets.has(markdown.normalizeLink(one.value))),
          )
        }
        return accepted
      })
    }
    const parse = markdown.helpers.parseLinkDestination
    markdown.helpers.parseLinkDestination = (text, start, max) => {
      const result = parse(text, start, max)
      if (result.ok && text === token.content) {
        destinations.push({ start, end: result.pos, value: result.str })
      }
      return result
    }
    const children: Token[] = []
    try {
      markdown.inline.parse(token.content, markdown, environment, children)
    } finally {
      markdown.helpers.parseLinkDestination = parse
    }
    const contentLines = token.content.split('\n')
    const columns = contentLines.map((content, index) => {
      const line = map[0] + index
      const column = lines[line]?.indexOf(content, cursors.get(line) ?? 0) ?? -1
      if (column >= 0) cursors.set(line, column + content.length)
      return column
    })
    for (const destination of new Map(destinations.map((one) => [one.start, one])).values()) {
      if (relocate(destination.value) === undefined) continue
      const preceding = token.content.slice(0, destination.start)
      const lineIndex = preceding.split('\n').length - 1
      const column = columns[lineIndex]!
      if (column < 0) throw new Error(`cannot locate Markdown inline content in ${from}`)
      recordEdit(
        token.content.slice(destination.start, destination.end),
        destination.value,
        [map[0] + lineIndex, map[0] + lineIndex + 1],
        column + preceding.length - (preceding.lastIndexOf('\n') + 1),
      )
    }
  }
  let result = source
  for (const [start, edit] of [...edits].sort(([left], [right]) => right - left)) {
    result = result.slice(0, start) + edit.value + result.slice(edit.end)
  }
  return result
}

/** 移動と参照更新を一つの操作として行う。計画をすべて作ってから書き込む。 */
async function relocateWorkItems(root: string, moves: ReadonlyMap<string, string>): Promise<void> {
  const snapshot = createWorkspaceSnapshot(root)
  for (const [from, to] of moves) {
    if (!snapshot.exists(from)) throw new Error(`source does not exist: ${from}`)
    if (snapshot.exists(to)) throw new Error(`destination already exists: ${to}`)
  }
  const writes: Array<{ path: string; source: string }> = []
  for (const path of await snapshot.files('', [
    '.git',
    '.worktrees',
    'node_modules',
    'site',
    'generated',
    'vendor',
    'dist',
    'build',
  ])) {
    if (!path.endsWith('.md')) continue
    const source = await snapshot.read(path)
    const target = moves.get(path) ?? path
    const result = relocateMarkdownLinks(source, path, target, moves)
    if (result !== source) writes.push({ path: target, source: result })
  }
  for (const [from, to] of moves) {
    await mkdir(dirname(snapshot.path(to)), { recursive: true })
    await rename(snapshot.path(from), snapshot.path(to))
  }
  for (const write of writes) await writeFile(snapshot.path(write.path), write.source)
}

export async function moveWorkItem(root: string, id: string): Promise<void> {
  const snapshot = createWorkspaceSnapshot(root)
  const stem = basename(id, '.md')
  const matches = (await snapshot.files('work-items/active')).filter((path) => {
    const name = basename(path, '.md')
    return path.endsWith('.md') && (name === stem || name.startsWith(`${stem}-`))
  })
  if (matches.length !== 1)
    throw new Error(`expected one active work item matching ${id}, found ${matches.length}`)
  const from = matches[0]!
  const record = parseFrontmatterAndMarkdown(from, await snapshot.read(from))
  if (record.status !== 'completed' && record.status !== 'cancelled') {
    throw new Error(`set status to completed or cancelled before moving ${from}`)
  }
  await relocateWorkItems(root, new Map([[from, `work-items/done/${basename(from)}`]]))
}
