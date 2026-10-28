import { expect, it } from 'bun:test'
import { resolve } from 'node:path'
import MarkdownIt from 'markdown-it'
import { markdownAnchors } from '../../check/src/markdown-links.ts'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { deriveContext } from '../src/context.ts'

const root = resolve(import.meta.dir, '../../..')
const markdown = new MarkdownIt({ html: true })

/** 同じ必要文書を全文で読む場合の上限と、候補の節だけを読む場合の上限を比較する。 */
it('measures representative reading spans without treating fewer candidates as coverage', async () => {
  const snapshot = createWorkspaceSnapshot(root)
  const measurements = []
  for (const seed of [
    { kind: 'feature', targets: ['backend/idmanagement/user'], concerns: [] },
    { kind: 'refactor', targets: ['backend/idmanagement/user'], concerns: ['boundary'] },
    { kind: 'docs', targets: ['docs/development/writing-language.md'], concerns: [] },
    { kind: 'tooling', targets: ['tools/brief'], concerns: [] },
  ]) {
    const plan = await deriveContext(seed, snapshot)
    expect(plan.warnings).toEqual([])
    const selected = new Map<string, Set<number>>()
    for (const candidate of plan.reading) {
      const [file, anchor] = candidate.location.split('#')
      if (!file?.endsWith('.md')) continue
      const source = await snapshot.read(file)
      const lines = source.split('\n')
      let start = 0
      let end = lines.length
      if (anchor) {
        const headings = markdown.parse(source, {}).filter((token) => token.type === 'heading_open')
        const index = headings.findIndex((token) =>
          markdownAnchors(lines[token.map![0]]!).has(anchor),
        )
        expect(index).toBeGreaterThanOrEqual(0)
        const heading = headings[index]!
        start = heading.map![0]
        end = headings.slice(index + 1).find((token) => token.tag <= heading.tag)?.map?.[0] ?? end
      }
      const visited = selected.get(file) ?? new Set<number>()
      for (let line = start; line < end; line++) visited.add(line)
      selected.set(file, visited)
    }
    let wholeDocumentLines = 0
    for (const file of selected.keys()) {
      wholeDocumentLines += (await snapshot.read(file)).split('\n').length
    }
    const afterSelectedLines = [...selected.values()].reduce((sum, lines) => sum + lines.size, 0)
    expect(afterSelectedLines).toBeLessThan(wholeDocumentLines)
    measurements.push({
      kind: seed.kind,
      documents: selected.size,
      wholeDocumentLines,
      afterSelectedLines,
      candidates: plan.reading.length,
      tasks: plan.tasks.length,
    })
  }
  console.info(
    'Reading-span comparison (whole-document envelope, not elapsed time or observed agent consumption):',
    JSON.stringify(measurements),
  )
})
