import { readdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { changedPaths } from '../../check/src/work-item-changes.ts'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'

const SUMMARY = /^- \*\*(?:要約|Summary)\*\*:\s*(.*)$/
const MOTIVATION = /^#{1,2}\s+(?:動機|Motivation)\s*$/

/** リンクの文字だけを残す。縮めた記録は後のファイルの移動に追従しないため、リンクを持たない。 */
function withoutLinks(text: string): string {
  return text.replace(/!?\[([^\]]*)\]\([^)]*\)/g, '$1')
}

function summaryOf(lines: readonly string[]): string[] {
  const start = lines.findIndex((line) => SUMMARY.test(line))
  if (start >= 0) {
    const collected = [lines[start]!.match(SUMMARY)![1]!]
    for (const line of lines.slice(start + 1)) {
      if (/^- \*\*/.test(line) || /^#/.test(line)) break
      collected.push(line.trim())
    }
    const summary = collected.filter((line) => line !== '')
    if (summary.length > 0) return summary
  }
  const motivation = lines.findIndex((line) => MOTIVATION.test(line))
  if (motivation < 0) return []
  const paragraph: string[] = []
  for (const line of lines.slice(motivation + 1)) {
    if (line.trim() === '') {
      if (paragraph.length > 0) break
      continue
    }
    if (/^#/.test(line)) break
    paragraph.push(line.trim())
  }
  return paragraph
}

/**
 * 完了記録を、状態、全文のあるコミット、題名、完了節の要約だけに縮める。
 * 要約がない記録は動機の最初の段落を使う。縮めた記録はそのまま返す。
 */
export function collapseRecord(text: string, revision: string): string {
  const frontmatter = text.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n/)
  if (frontmatter && /^archived_from:/m.test(frontmatter[1]!)) return text
  const status = frontmatter?.[1]?.match(/^status:\s*(\S+)/m)?.[1] ?? 'completed'
  const lines = text.slice(frontmatter?.[0].length ?? 0).split('\n')
  const title = lines.find((line) => /^# /.test(line)) ?? '# (題名なし)'
  return [
    '---',
    `status: ${status}`,
    `archived_from: ${revision}`,
    '---',
    '',
    title,
    '',
    ...summaryOf(lines).map(withoutLinks),
    '',
    `全文は \`git show ${revision}:<このファイルのパス>\` で読む。`,
    '',
  ].join('\n')
}

/** パスごとに、そのパスを最後に変えたコミット。`git log` を一回だけ読む。 */
function lastRevisions(root: string): Map<string, string> {
  const result = Bun.spawnSync(
    ['git', 'log', '--format=%H', '--name-only', 'main', '--', 'work-items/done'],
    { cwd: root },
  )
  const revisions = new Map<string, string>()
  // Git の外や main のない workspace では全文の所在を示せないので、何も縮めない。
  if (result.exitCode !== 0) return revisions
  let revision = ''
  for (const line of result.stdout.toString().split('\n')) {
    if (/^[0-9a-f]{40}$/.test(line)) revision = line
    else if (line && !revisions.has(line)) revisions.set(line, revision)
  }
  return revisions
}

/**
 * main に取り込み済みで、作業中の変更に含まれない完了記録を縮める。
 * 完了させる変更の中の記録は全文で検証を受けるため、取り込まれるまで縮めない。
 */
export async function archiveWorkItems(root: string): Promise<string[]> {
  const changed = changedPaths(createWorkspaceSnapshot(root), 'work-items') ?? new Set()
  const revisions = lastRevisions(root)
  const archived: string[] = []
  for (const name of (await readdir(resolve(root, 'work-items/done'))).sort()) {
    const path = `work-items/done/${name}`
    const revision = revisions.get(path)
    if (!name.endsWith('.md') || changed.has(path) || !revision) continue
    const text = await readFile(resolve(root, path), 'utf8')
    const collapsed = collapseRecord(text, revision)
    if (collapsed === text) continue
    await writeFile(resolve(root, path), collapsed)
    archived.push(path)
  }
  return archived
}
