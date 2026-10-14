const SUMMARY_SECTION = /^##\s+要約\s*$/

/** リンクの文字だけを残す。縮めた記録は後のファイルの移動に追従しないため、リンクを持たない。 */
function withoutLinks(text: string): string {
  return text.replace(/!?\[([^\]]*)\]\([^)]*\)/g, '$1')
}

/** `## 要約` の節の本文。節がないか空なら undefined。 */
export function summaryOf(text: string): string[] | undefined {
  const lines = text.split('\n')
  const start = lines.findIndex((line) => SUMMARY_SECTION.test(line))
  if (start < 0) return undefined
  const body: string[] = []
  for (const line of lines.slice(start + 1)) {
    if (/^#{1,2}\s/.test(line)) break
    body.push(line)
  }
  while (body[0]?.trim() === '') body.shift()
  while (body.at(-1)?.trim() === '') body.pop()
  return body.length > 0 ? body : undefined
}

/**
 * 記録を、状態、題名、`## 要約` だけに縮める。全文は作業中のコミットに残り、
 * `work-items/active/` のパスの履歴から読む。
 */
export function collapseRecord(text: string, status: string, activePath: string): string {
  const summary = summaryOf(text)
  if (!summary) throw new Error(`${activePath} has no ## 要約 section to keep`)
  const title = text.split('\n').find((line) => /^# /.test(line)) ?? '# (題名なし)'
  return [
    '---',
    `status: ${status}`,
    '---',
    '',
    title,
    '',
    ...summary.map(withoutLinks),
    '',
    `全文は \`git log -1 -- ${activePath}\` が示すコミットの親、またはそのコミットで読む。`,
    '',
  ].join('\n')
}
