/**
 * 設計文書の用語を、採用した表記へ固定する。
 *
 * 採らない表記は、どれも普通名詞としての読みが先に立つ日本語訳である。読み手が
 * その節を deployment、secret、runtime、capacity、observability、platform の
 * どれとして読むかを文脈から推定することになり、用語から概念へたどれない。
 *
 * 素朴な禁止語一覧にすると共起で正当な用法まで落ちるので、規則は残す共起を
 * `allow` として理由付きで持つ。`allow` は対象語を含む literal で、その literal が
 * 覆う位置に現れた occurrence だけを採用済みの用法として通す。免除はここにしか
 * 無く、ファイル単位の免除は持たない。文書単位で外せるようにすると、その文書
 * だけ用語が戻ったことにだれも気付かなくなる。
 */

export type TerminologyDocument = {
  file: string
  source: string
}

export type TerminologyFinding = {
  file: string
  line: number
  column: number
  term: string
  message: string
}

/** 採らない表記 1 件と、その代わりに使う表記。 */
export type TerminologyRule = {
  /** 採らない表記。 */
  term: string
  /** 代わりに使う表記。指摘文にそのまま載る。 */
  adopt: string
  /** この literal が覆う位置の occurrence は、別概念なので通す。 */
  allow?: readonly { readonly literal: string; readonly reason: string }[]
}

export const TERMINOLOGY_RULES: readonly TerminologyRule[] = [
  { term: '配備', adopt: '「デプロイ」（行為）または「デプロイメント」（ビュー名）' },
  {
    term: '秘密',
    adopt: '「シークレット」',
    allow: [
      { literal: '秘密鍵', reason: 'private key であってシークレットではない' },
      {
        literal: '秘密情報',
        reason: '復号できる形で保持する機微データ。起動時シークレットとは別の概念',
      },
    ],
  },
  { term: '実行時アーキテクチャ', adopt: '「ランタイムアーキテクチャ」' },
  { term: 'API 規則', adopt: '「API ガイドライン」' },
  { term: 'API規則', adopt: '「API ガイドライン」' },
  { term: '設計規則', adopt: '「設計ガイドライン」' },
  {
    term: '容量',
    adopt: '「キャパシティ」',
    allow: [
      { literal: '保存容量', reason: 'ストレージの量であり capacity planning ではない' },
      { literal: '空き容量', reason: 'ストレージの量であり capacity planning ではない' },
      { literal: '容量超過', reason: 'ストレージの量が上限を超えることであり capacity ではない' },
    ],
  },
  { term: '観測可能性', adopt: '「オブザーバビリティ」' },
  { term: '基盤設計', adopt: '「プラットフォーム設計」' },
  {
    term: 'トポロジ',
    adopt: '「トポロジー」',
    allow: [{ literal: 'トポロジー', reason: '採用した表記そのもの' }],
  },
  { term: '入場制御', adopt: '「アドミッションコントロール」' },
  { term: '参照運用プロファイル', adopt: '「リファレンスワークロードプロファイル」' },
  { term: '構成算出規則', adopt: '「サイジング計算式」' },
  { term: '縮退順序', adopt: '「ロードシェディング順序」' },
  {
    term: '訓練',
    adopt: '演習なら「ドリル」、人への教育なら「研修」。どちらを指すかを決めて書く',
  },
]

/** 用語を固定する文書のうち、リポジトリ root 直下にあるもの。 */
export const TERMINOLOGY_ROOT_DOCUMENTS: readonly string[] = [
  'AGENTS.md',
  'CONTRIBUTING.md',
  'DOCUMENTATION_GUIDE.md',
  'README.md',
  'SECURITY.md',
  'SPECIFICATION_FORMAT.md',
  'WORK_ITEM_FORMAT.md',
]

/** literal が覆う位置を、対象語の occurrence と同じ座標系で集める。 */
function allowedSpans(line: string, rule: TerminologyRule): Array<[number, number]> {
  const spans: Array<[number, number]> = []
  for (const { literal } of rule.allow ?? []) {
    let from = line.indexOf(literal)
    while (from !== -1) {
      spans.push([from, from + literal.length])
      from = line.indexOf(literal, from + 1)
    }
  }
  return spans
}

export function verifyTerminology(
  documents: readonly TerminologyDocument[],
  rules: readonly TerminologyRule[] = TERMINOLOGY_RULES,
): TerminologyFinding[] {
  const findings: TerminologyFinding[] = []
  for (const document of documents) {
    const lines = document.source.split('\n')
    for (const [index, line] of lines.entries()) {
      for (const rule of rules) {
        const spans = allowedSpans(line, rule)
        let at = line.indexOf(rule.term)
        while (at !== -1) {
          const end = at + rule.term.length
          const covered = spans.some(([from, to]) => from <= at && end <= to)
          if (!covered) {
            findings.push({
              file: document.file,
              line: index + 1,
              column: at + 1,
              term: rule.term,
              message: `「${rule.term}」は採らない表記。${rule.adopt} を使う。`,
            })
          }
          at = line.indexOf(rule.term, at + 1)
        }
      }
    }
  }
  return findings
}
