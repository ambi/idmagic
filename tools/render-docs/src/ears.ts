/**
 * 機能仕様の要件文を、日本語の EARS の節に分ける。生成サイトは、EARS のキーワードを受け入れ例の
 * Gherkin と同じ見た目の名札で示し、条件と応答の対応を一目で読めるようにする。
 *
 * EARS（Alistair Mavin）の型は次のとおりで、節の順序は常に同じである。
 * - 常時：The 〈system name〉 shall 〈system response〉
 * - 状態：While 〈precondition〉, the 〈system name〉 shall …
 * - 契機：When 〈trigger〉, the 〈system name〉 shall …
 * - 任意の機能：Where 〈feature is included〉, the 〈system name〉 shall …
 * - 望まない振る舞い：If 〈trigger〉, then the 〈system name〉 shall …
 *
 * 要件文の書式は仕様フォーマットの EARS 形式が定め、検査（`verifyEarsStatements`）が確かめる。
 * ここは表示のための読み取りであり、読み取れない文は分けずに原文のまま表示させる。
 */

/** EARS のキーワード。応答の節は Shall で、望まない振る舞いの型では Then も付く。 */
export type EarsKeyword = 'where' | 'while' | 'when' | 'if' | 'then' | 'shall'

/** 名札。EARS のキーワードを、受け入れ例の Gherkin と同じく英語のまま示す。 */
const EARS_LABELS = new Map<EarsKeyword, string>([
  ['where', 'Where'],
  ['while', 'While'],
  ['when', 'When'],
  ['if', 'If'],
  ['then', 'Then'],
  ['shall', 'Shall'],
])

/**
 * 表示で取り除く範囲と、その位置から始まる節。`from` から `to` までの文字（前置きの区切りの語と
 * 読点）を消し、そこから `keywords` の名札を付けた節を始める。最初の要素は文頭から始まる。
 * EARS は要件に主体を一つ求めるので、応答の節は主体の「〈主体〉は、」から始めて残す。
 */
export type EarsCut = { from: number; to: number; keywords: EarsKeyword[] }

const MARKER_KEYWORDS = new Map<string, EarsKeyword>([
  ['では', 'where'],
  ['の間', 'while'],
  ['間', 'while'],
  ['とき', 'when'],
  ['場合', 'if'],
])

/** 生成サイトが要件文の節を区切る印。名札は受け入れ例の Gherkin のキーワードと同じ見た目にする。 */
function clauseOpening(keywords: readonly EarsKeyword[]): string {
  const labels = keywords
    .map(
      (keyword) =>
        `<span class="scenario-keyword ears-${keyword}">${EARS_LABELS.get(keyword)}</span> `,
    )
    .join('')
  return `<span class="ears-clause">${labels}`
}

/** インラインの要素のうち、ここで読み書きする部分。 */
export type InlineToken = { type: string; content: string }

/**
 * 箇条の本文のインラインの要素を、EARS の節ごとに区切った並びに置き換える。
 * 区切りの語はテキストの要素の中にあるので、テキストだけを連ねて節を読み、その位置で
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
  const out: T[] = [make('html_inline', clauseOpening(first.keywords))]
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
      out.push(make('html_inline', `</span>${clauseOpening(cut.keywords)}`))
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
  if (markStart === 0) return [{ from: 0, to: 0, keywords: ['shall'] }]

  // 前置きは、最後の節の区切りの読点までである。
  const preamble = statement.slice(0, markStart)
  const clauses: Array<{ keyword: EarsKeyword; markerStart: number; markerEnd: number }> = []
  PREAMBLE_CLAUSE.lastIndex = 0
  while (PREAMBLE_CLAUSE.lastIndex < preamble.length) {
    const start = PREAMBLE_CLAUSE.lastIndex
    const clause = PREAMBLE_CLAUSE.exec(preamble)
    if (clause === null) return undefined
    const [, content = '', marker = ''] = clause
    const keyword = MARKER_KEYWORDS.get(marker)
    if (keyword === undefined || content === '') return undefined
    const markerStart = start + content.length
    clauses.push({ keyword, markerStart, markerEnd: PREAMBLE_CLAUSE.lastIndex })
  }
  const [first] = clauses
  if (first === undefined) return undefined
  // 望まない振る舞いの型は、If と対になる Then を応答の前に置く。
  const response: EarsKeyword[] = clauses.some((clause) => clause.keyword === 'if')
    ? ['then', 'shall']
    : ['shall']
  return [
    { from: 0, to: 0, keywords: [first.keyword] },
    ...clauses.map((clause, index) => {
      const next = clauses[index + 1]
      return {
        from: clause.markerStart,
        to: clause.markerEnd,
        keywords: next === undefined ? response : [next.keyword],
      }
    }),
  ]
}
