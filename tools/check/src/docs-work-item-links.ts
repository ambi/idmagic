/**
 * 現在状態を書く文書から、work item を参照させない。
 *
 * work item は一つの変更の計画と経緯の記録であり、完了すれば更新されない。
 * 現在状態の文書がそれを指すと、記録が古くなった時点で文書の記述も古くなる。
 * 依存は work item から文書へだけ向け、逆向きには持たない。
 *
 * 指摘するのは `wi-<番号>` という識別子と、`work-items/` を指す Markdown の
 * リンクである。ディレクトリ名を地の文で説明するだけの記述は、特定の記録に
 * 依存しないので通す。
 */

export type WorkItemLinkDocument = {
  file: string
  source: string
}

export type WorkItemLinkFinding = {
  file: string
  line: number
  column: number
  reference: string
}

/** 変更単位で書く文書と、手順の例として番号を使う文書は対象外にする。 */
const EXCLUDED_DIRECTORIES: readonly string[] = ['docs/releases/', 'docs/development/']

export function isCurrentStateDocument(path: string): boolean {
  if (!path.startsWith('docs/') || !path.endsWith('.md')) return false
  return !EXCLUDED_DIRECTORIES.some((directory) => path.startsWith(directory))
}

/** 英数字に続く `wi-` は別の識別子の一部なので、直前が英数字でない位置に限る。 */
const WORK_ITEM_ID = /(?<![A-Za-z0-9])wi-[0-9]+/g
const LINK_TARGET = /\]\(([^)]*)\)/g
const WORK_ITEMS_DIRECTORY = 'work-items/'

export function verifyNoWorkItemLinks(
  documents: readonly WorkItemLinkDocument[],
): WorkItemLinkFinding[] {
  const findings: WorkItemLinkFinding[] = []
  for (const document of documents) {
    for (const [index, line] of document.source.split('\n').entries()) {
      const matches: Array<{ column: number; reference: string }> = []
      for (const match of line.matchAll(WORK_ITEM_ID)) {
        matches.push({ column: match.index + 1, reference: match[0] })
      }
      for (const match of line.matchAll(LINK_TARGET)) {
        const offset = (match[1] ?? '').indexOf(WORK_ITEMS_DIRECTORY)
        if (offset === -1) continue
        // リンク先は `](` の 2 文字の後ろから始まる。
        matches.push({ column: match.index + 2 + offset + 1, reference: WORK_ITEMS_DIRECTORY })
      }
      matches.sort((left, right) => left.column - right.column)
      for (const { column, reference } of matches) {
        findings.push({ file: document.file, line: index + 1, column, reference })
      }
    }
  }
  return findings
}
