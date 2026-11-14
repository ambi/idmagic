/**
 * 設計文書の用語を、採用した表記へ固定する。
 *
 * 照合は部分文字列の一致であり、文脈を読まない。そのため規則に置くのは、一致した
 * 箇所が採らない表記としか読めない語に限る。複合語、旧い区分や文書の名前、表記の
 * 揺れがこれに当たる。「秘密」「容量」「訓練」のような一般語や、「列型」のように
 * 別の語（文字列型、配列型）の一部になる語は、正当な用法まで落とすので置かない。
 * 例外の一覧を積み増さないと保てない規則は、効果が確かでない規則として外す。
 *
 * `allow` は、採用した表記が採らない表記を含む場合（トポロジーとトポロジ）にだけ使う。
 * literal が覆う位置に現れた occurrence を通す。ファイル単位の免除は持たない。
 * 文書単位で外せるようにすると、その文書だけ用語が戻ったことにだれも気付かなくなる。
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
  { term: '実行時アーキテクチャ', adopt: '「ランタイムアーキテクチャ」' },
  { term: 'API 規則', adopt: '「API ガイドライン」' },
  { term: 'API規則', adopt: '「API ガイドライン」' },
  { term: '設計規則', adopt: '「設計ガイドライン」' },
  { term: '観測可能性', adopt: '「オブザーバビリティ」' },
  { term: '基盤設計', adopt: '「プラットフォーム設計」' },
  {
    term: 'トポロジ',
    adopt: '「トポロジー」',
    allow: [{ literal: 'トポロジー', reason: '採用した表記そのもの' }],
  },
  { term: '入場制御', adopt: '「アドミッションコントロール」' },
  { term: '参照運用プロファイル', adopt: '「想定ワークロード」' },
  { term: 'リファレンスワークロードプロファイル', adopt: '「想定ワークロード」' },
  { term: 'Planning assumption', adopt: '「仮定値」' },
  { term: 'Specification target', adopt: '「仕様目標」' },
  // requirement の訳語。「要求」は依頼やリクエストの意味の一般語と区別できないので、区分と文書の名前は「要件」にする。
  { term: '要求文書', adopt: '「要件文書」' },
  { term: '機能要求', adopt: '「機能要件」' },
  { term: '品質要求', adopt: '「品質要件」' },
  { term: 'システム要求', adopt: '「システム要件」' },
  { term: '話題の索引', adopt: '「設計領域の索引」' },
  { term: '構成算出規則', adopt: '「サイジング計算式」' },
  { term: '縮退順序', adopt: '「ロードシェディング順序」' },
  { term: '参照トポロジー', adopt: '「共通トポロジー」' },
  { term: '参照プロファイル', adopt: '「デプロイプロファイル」' },
  { term: '既定', adopt: '「デフォルト」' },
  { term: 'コンピュート', adopt: '「コンピューティング」' },
  // 単独の「行」「列」は CSV、表、ログにも使う一般語なので規則にできない。
  // データベースの意味にしか読めない複合語だけを固定する。
  { term: '非キー列', adopt: '「非キーカラム」' },
  { term: 'ランブック', adopt: '「運用手順書」' },
  { term: 'ドメイン設計文書', adopt: '「モジュール設計文書」' },
]

/** 用語を固定する文書のうち、リポジトリ root 直下にあるもの。 */
export const TERMINOLOGY_ROOT_DOCUMENTS: readonly string[] = [
  'AGENTS.md',
  'CONTRIBUTING.md',
  'README.md',
  'SECURITY.md',
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
