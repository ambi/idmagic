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
  { term: '参照運用プロファイル', adopt: '「想定ワークロード」' },
  { term: 'リファレンスワークロードプロファイル', adopt: '「想定ワークロード」' },
  { term: 'Planning assumption', adopt: '「仮定値」' },
  { term: 'Specification target', adopt: '「仕様目標」' },
  {
    term: '本書',
    adopt: '「この文書」',
    allow: [{ literal: '「本書は」', reason: '採らない書き出しとして引用している' }],
  },
  { term: '構成算出規則', adopt: '「サイジング計算式」' },
  { term: '縮退順序', adopt: '「ロードシェディング順序」' },
  {
    term: '訓練',
    adopt:
      '障害を起こして対処を確かめるなら「障害試験」、バックアップから戻して確かめるなら「復元試験」、人への教育なら「研修」。どれを指すかを決めて書く',
  },
  { term: 'ドリル', adopt: '障害を起こすなら「障害試験」、バックアップから戻すなら「復元試験」' },
  {
    term: '演習',
    adopt:
      '障害を起こすなら「障害試験」、バックアップから戻すなら「復元試験」、人への教育なら「研修」',
  },
  { term: '参照トポロジー', adopt: '「共通トポロジー」' },
  { term: '参照プロファイル', adopt: '「デプロイプロファイル」' },
  { term: '既定', adopt: '「デフォルト」' },
  { term: 'コンピュート', adopt: '「コンピューティング」' },
  {
    term: '資材',
    adopt: '「構成ファイル」。Kubernetes に限るなら「マニフェスト」',
  },
  // 単独の「行」「列」は CSV、表、ログにも使う一般語なので規則にできない。
  // データベースの意味にしか読めない複合語だけを固定する。
  { term: '非キー列', adopt: '「非キーカラム」' },
  {
    term: '列型',
    adopt: '「カラム型」',
    allow: [{ literal: '配列型', reason: 'array 型であってカラムの型ではない' }],
  },
  { term: 'ランブック', adopt: '「運用手順書」' },
  {
    term: 'runbook',
    adopt: '「運用手順書」',
    allow: [
      { literal: 'runbooks/', reason: '運用手順書を置くディレクトリのパス' },
      { literal: 'runbook_url', reason: 'Prometheus のアラートが持つアノテーション名' },
    ],
  },
  { term: 'Runbook', adopt: '「運用手順書」' },
  { term: 'ドメイン設計文書', adopt: '「モジュール設計」' },
  {
    term: 'コンテキスト',
    adopt: '設計と仕様の単位は「モジュール」',
    allow: [
      { literal: '認証コンテキスト', reason: '認証済み主体の情報であり、設計の単位ではない' },
      { literal: '実行コンテキスト', reason: '処理の実行の文脈であり、設計の単位ではない' },
      {
        literal: 'リクエストコンテキスト',
        reason: 'リクエストの情報を運ぶ文脈であり、設計の単位ではない',
      },
      { literal: 'セキュリティコンテキスト', reason: '認証規格の情報であり、設計の単位ではない' },
      { literal: 'プラグインのコンテキスト', reason: 'ビルドプラグインの実行環境である' },
      { literal: '`LocaleProvider` のコンテキスト', reason: 'React の Context である' },
      { literal: 'ルーターのコンテキスト', reason: '描画に必要な React の Context である' },
      { literal: 'アカウントコンテキスト', reason: 'ブラウザー初期化 API の情報である' },
      { literal: 'パスワードリセットコンテキスト', reason: 'ブラウザー初期化 API の情報である' },
      { literal: 'コンテキストが返る', reason: 'ブラウザー初期化 API の応答である' },
      { literal: 'コンテキストの節約', reason: 'エージェントが読み込む情報量を指す' },
    ],
  },
  // Bounded Context、Context Map、Context 間もこの規則で落ちる。Context Map は
  // 置き換え先のない廃止した概念なので、採用語の指摘文で示す。
  {
    term: 'Context',
    adopt: '「モジュール」。Context Map は廃止したので書かない',
    allow: [
      { literal: 'context.Context', reason: 'Go の標準ライブラリの型であり、設計の単位ではない' },
      { literal: 'System Context', reason: 'C4 model のビュー名であり、設計の単位ではない' },
      // 標準の行の文面を変えると、振る舞いを固定する証拠が要る。語の差し替えだけでは
      // その証拠を作れないので、行に触れる次の作業が改名する。
      { literal: 'ApiTokens Context', reason: '標準の行 RFC7644-BEARER-AUTHORIZATION の文面' },
      { literal: 'OAuth2 Context', reason: '標準の行 GDPR-CONSENT-WITHDRAWAL の文面' },
      { literal: 'Audit Context', reason: '標準の行 GDPR-PROCESSING-RECORDS の文面' },
    ],
  },
  {
    term: '公開言語',
    adopt: 'import の可否なら「公開パッケージ」、型、意味、拒否、作用の約束なら「公開契約」',
  },
  {
    term: 'Published Language',
    adopt: 'import の可否なら「公開パッケージ」、型、意味、拒否、作用の約束なら「公開契約」',
  },
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
