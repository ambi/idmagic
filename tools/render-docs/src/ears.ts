/**
 * 機能仕様の要件文を、日本語の EARS の節に分ける。生成サイトは、受け入れ例の Gherkin と同じ
 * 見た目の名札で節を区切り、条件と応答の対応を一目で読めるようにする。
 *
 * 要件文の書式は仕様フォーマットの EARS 形式が定め、検査（`verifyEarsStatements`）が確かめる。
 * ここは表示のための読み取りであり、読み取れない文は分けずに原文のまま表示させる。
 */

/** 節の種類。前置きの四つと、主体が返す応答である。 */
export type EarsKind = 'where' | 'while' | 'when' | 'if' | 'then'

/** 節の名札。EARS の英語のキーワードを、受け入れ例の Gherkin と同じく英語のまま示す。 */
const EARS_LABELS = new Map<EarsKind, string>([
  ['where', 'Where'],
  ['while', 'While'],
  ['when', 'When'],
  ['if', 'If'],
  ['then', 'Then'],
])

/**
 * 表示で取り除く範囲と、その位置から始まる節。`from` から `to` までの文字（前置きの区切りの語と、
 * 主体の「〈主体〉は、」）を消し、そこから `kind` の節を始める。最初の要素は文頭から始まる。
 */
export type EarsCut = { from: number; to: number; kind: EarsKind }

const MARKER_KINDS = new Map<string, EarsKind>([
  ['では', 'where'],
  ['の間', 'while'],
  ['間', 'while'],
  ['とき', 'when'],
  ['場合', 'if'],
])

/** 生成サイトが要件文の節を区切る印。名札は受け入れ例の Gherkin のキーワードと同じ見た目にする。 */
function clauseOpening(kind: EarsKind): string {
  return `<span class="ears-clause"><span class="scenario-keyword ears-${kind}">${EARS_LABELS.get(kind)}</span> `
}

/** インラインの要素のうち、ここで読み書きする部分。 */
export type InlineToken = { type: string; content: string }

/**
 * 箇条の本文のインラインの要素を、EARS の節ごとに区切った並びに置き換える。
 * 区切りの語と主体はテキストの要素の中にあるので、テキストだけを連ねて節を読み、その位置で
 * テキストを分けて印を差し込む。読み取れなければ undefined を返し、原文を残させる。
 */
export function earsChildren<T extends InlineToken>(
  children: readonly T[],
  responder: string,
  make: (type: 'text' | 'html_inline', content: string) => T,
): T[] | undefined {
  const plain = children
    .filter((child) => child.type === 'text')
    .map((child) => child.content)
    .join('')
  const cuts = earsCuts(plain, responder)
  const [first] = cuts ?? []
  if (!cuts || !first) return undefined
  const out: T[] = [make('html_inline', clauseOpening(first.kind))]
  let skipUntil = first.to
  let next = 1
  let offset = 0
  for (const child of children) {
    if (child.type !== 'text') {
      // 取り除く範囲の途中にテキスト以外の要素があれば、区切りを読み違えている。
      if (offset < skipUntil) return undefined
      out.push(child)
      continue
    }
    const start = offset
    const end = offset + child.content.length
    let cursor = Math.min(Math.max(start, skipUntil), end)
    for (let cut = cuts[next]; cut !== undefined && cut.from < end; cut = cuts[next]) {
      if (cut.from < cursor) return undefined
      if (cut.from > cursor) out.push(make('text', plain.slice(cursor, cut.from)))
      out.push(make('html_inline', `</span>${clauseOpening(cut.kind)}`))
      skipUntil = cut.to
      next += 1
      cursor = Math.min(skipUntil, end)
    }
    if (cursor < end) out.push(make('text', plain.slice(cursor, end)))
    offset = end
  }
  if (next < cuts.length) return undefined
  out.push(make('html_inline', '</span>'))
  return out
}

/** 前置きの一節。検査と同じく、区切りの語と読点で終わる。 */
const PREAMBLE_CLAUSE = /(.*?)(では|の間|(?<=[うくすつぬむるぐぶい])間|とき|場合)、/y

function escapeRegExp(text: string): string {
  return text.replaceAll(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * 要件文を節に分ける。`statement` は箇条書きの記号を除いた本文で、コード片を含まない文字列として
 * 渡す。主体がちょうど一度だけ現れ、前置きがすべて区切りの語で終わる文だけを分ける。
 */
export function earsCuts(statement: string, responder: string): EarsCut[] | undefined {
  // 読点を消費せずに数え、主体が続けて二度現れる文も二度と数える。
  const marks = [
    ...statement.matchAll(new RegExp(`(?:^|(?<=、))${escapeRegExp(responder)} ?は、`, 'g')),
  ]
  const [mark] = marks
  if (mark === undefined || marks.length !== 1) return undefined
  const markStart = mark.index ?? 0
  const responseStart = markStart + mark[0].length
  if (markStart === 0) return [{ from: 0, to: responseStart, kind: 'then' }]

  // 前置きは、最後の節の区切りの読点までである。
  const preamble = statement.slice(0, markStart)
  const clauses: Array<{ kind: EarsKind; markerStart: number; markerEnd: number }> = []
  PREAMBLE_CLAUSE.lastIndex = 0
  while (PREAMBLE_CLAUSE.lastIndex < preamble.length) {
    const start = PREAMBLE_CLAUSE.lastIndex
    const clause = PREAMBLE_CLAUSE.exec(preamble)
    if (clause === null) return undefined
    const [, content = '', marker = ''] = clause
    const kind = MARKER_KINDS.get(marker)
    if (kind === undefined || content === '') return undefined
    const markerStart = start + content.length
    clauses.push({ kind, markerStart, markerEnd: PREAMBLE_CLAUSE.lastIndex })
  }
  const [first] = clauses
  if (first === undefined) return undefined
  return [
    { from: 0, to: 0, kind: first.kind },
    ...clauses.map((clause, index) => {
      const next = clauses[index + 1]
      return next === undefined
        ? { from: clause.markerStart, to: responseStart, kind: 'then' as const }
        : { from: clause.markerStart, to: clause.markerEnd, kind: next.kind }
    }),
  ]
}
