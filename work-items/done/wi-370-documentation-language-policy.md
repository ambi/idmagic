---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-08-14
depends_on: []
change_kind: docs
initial_context:
  specification: [spec/SPECIFICATION.md]
  typespec: []
  source: [AGENTS.md, README.md, infra/runbooks]
  tests: []
  stop_before_reading: [backend, frontend]
spec_impact:
  kind: none
  reason: "規範的な意味、REQ ID、TypeSpec シンボル、Strength 列は変更せず、散文の言語と表現を統一する。"
---

# 文書とコードコメントの言語を対象ごとに定め、日英混在を解消する

## Motivation
リポジトリの文章が日本語と英語で混在している。実測は次のとおり。

| 区分 | 実態 |
|---|---|
| `README.md` / `CONFIGURATION.md` / `DEVELOPMENT.md` / `*_FORMAT.md` / `AGENTS.md` / `spec/SPECIFICATION.md` | 英語 100% |
| `spec/contexts/*/SPECIFICATION.md` 21 件 | 混在 |
| `work-items/**` | 368/368 が日本語 |
| Go のコードコメント | 705 ファイル / 約 4,746 行が日本語 |

問題は「どちらの言語か」ではなく、**どちらの言語でもない状態**にある。`spec/contexts/authentication/SPECIFICATION.md` の `Overview` には日本語と英語でほぼ同じ内容を述べる段落があり、二重管理が発生している。`Standards` の `Statement` 列にも、一般語を英字のまま残して日本語の述語をつなぐ表現があり、日本語話者にも英語話者にも読みにくい。

さらに `AGENTS.md` の "Code comments must be written in English." は 705 ファイルで守られておらず、規則と実態のどちらかを変える必要がある。

オーナーの母語は日本語であり、設計文書を日本語で読めることが理解速度と設計品質に直結する。リポジトリは `github.com/ambi/idmagic` として公開済みだが、現在の保守者はオーナー 1 人である。英語の正本を維持する費用より、保守者が正確に読み書きできることを優先する。

## Scope
- **policy**:
  - `AGENTS.md` に `Language` 節を追加し、対象ごとの言語を表で定義する。言語を宣言する場所は `AGENTS.md` ただ一箇所とする。
  - コードコメントのルールを英語から日本語へ変更する。
  - ルートの `README.md` を日本語とし、英語版を併記しない。
  - `Formatting` 節を追加する。Markdown に桁数の上限を設けず、日本語の散文は 1 段落 1 行にする。段落内の改行は描画時に空白になるため、日本語の文字どうしの間で改行すると本来ない空白が入る。ソースファイルは 150 桁程度で折り返す。
- **spec**:
  - `spec/SPECIFICATION.md` を日本語化する。
  - `spec/contexts/*/SPECIFICATION.md` 21 件から日英重複段落を除去し、`Glossary` の `Definition` 列と `Standards` の `Statement` 列の日英混在を解消する。
- **docs**:
  - `frontend/` `infra/` `seed/` 配下の README と `infra/runbooks/*` を日本語化する。
  - 既存の英語版 `README.md` を日本語版で置き換え、`README.ja.md` は残さない。

## Out of Scope
- 既存の Go / TypeScript コメント約 4,746 行の書き換え。規則を変更するだけで、既存コメントは触らない。英語を主な作業言語とする保守体制へ変わった場合に、別の作業項目として一括英訳する。
- `DEVELOPMENT.md` / `SPECIFICATION_FORMAT.md` / `WORK_ITEM_FORMAT.md` / `AGENTS.md` 本体の日本語化。これらは英語で維持する。
- `DEVELOPMENT.md` / `SPECIFICATION_FORMAT.md` / `WORK_ITEM_FORMAT.md` への言語に関する記述の追加。これらは将来別リポジトリへ切り出して汎用化する予定であり、利用リポジトリごとに言語が異なりうる。言語について沈黙していること自体が「利用リポジトリが決めてよい」を意味する。
- `tools/**` の日本語化。汎用 tool として将来別リポジトリへ切り出せるよう、文書とコメントを含む全内容を英語で維持する。
- UI 文言。`*.i18n.ts` の辞書で `ja` / `en` を切り替える現在のローカライゼーションを維持し、どちらか一方だけにはしない。
- 言語を検査するツールの追加。規則の明文化だけで担保し、まれな違反は許容する。
- 英訳生成パイプライン（`just spec-translate` など）の実装。リポジトリは公開済みだが読み手がいないため、現時点では維持コストを払わない。スター、フォーク、Issue、PR のいずれかが付いた時点で起票し直す。

## Design

### Language by Target

正本は `AGENTS.md` の `Language` 節に置く対象別の表とする。層や区分の名前は導入しない。名前を付けても対象を一覧するのと情報量が変わらず、どの層に属するかを解釈する手間だけが増える。

未掲載のものを判断するための一文だけを添える。リポジトリの外に出るか実行時に現れるものは翻訳で差し替えられないので英語、それ以外は日本語とする。

英語で固定する側が翻訳で代替できないのは、エラーメッセージや識別子が実行時に現れ、翻訳処理を通せないためである。逆に設計文書の散文は、`REQ-<CONTEXT>-NNN` と TypeSpec シンボルという言語に依存しない骨格が別に存在するため、散文だけを日本語にしても参照が壊れない。

### Terminology

自然な技術文書の日本語を優先する。日本語またはカタカナの表記が定着した一般語は英字で残さない。英字を残すのは、識別子、値、パス、コマンド、製品名、プロトコル名、略語など正確な綴り自体が意味を持つもの、定着した日本語表記がないもの、および原語で参照した方が意味の明確な名前付き技術パターンに限る。コードや設定上の正確なトークンはバッククォートで囲み、説明上の一般語やパターン名と区別する。

`OTPを生成・検証する` のような英語の名詞と日本語の述語の組み合わせは、OTP のような標準的な略語であれば許容する。一方、`verifyする` のように通常の日本語で表せる動詞を英字のまま接続したり、翻訳を機械的に対応させるためだけに一般語を英字で残したりしない。

複数の表記が妥当な場合は文書ごとに選ばず、リポジトリ全体で統一する。ただし、単語ごとの置換表で決めず、文中で担う意味から表記を選ぶ。異なる概念を同じ訳語に潰さず、名前付き技術パターンを指す語と一般語を区別する。実際の識別子、ディレクトリ、パッケージを指す場合は、正確な名前をバッククォートで囲む。

### Role of Code Comments

コメントは同じ位置に日本語と英語の両方を置けないため、英訳は生成物ではなく一方向の移行になる。文書のように「日本語を正本にして英語を生成する」という逃げ道がない。

それでも日本語を選ぶ根拠は、リポジトリが公開済みであることを認めた上での次の判断による。

- コメントは「なぜ」を書く場所であり、母語でない言語で書くと精度が落ちる。曖昧な英語コメントは誰の役にも立たない。
- コメントはコードと一緒に読まれ、その識別子、型、関数名は英語のまま残る。読み手が得る情報はコメントだけではない。
- 現在の保守者はオーナー 1 人であり、現段階では保守速度を将来の貢献者の入口より優先する。

これは基準からの帰結ではなく判断である。英語を主な作業言語とする保守体制へ変わった場合は、約 4,746 行のコメントを一括して英訳する。

### Policy Location

宣言は `AGENTS.md` の `Language` 節ただ一箇所に置き、設定ファイルは新設しない。

- `AGENTS.md` は利用リポジトリ側に属し、方法論文書とライフサイクルが分かれている。`DEVELOPMENT.md` / `SPECIFICATION_FORMAT.md` / `WORK_ITEM_FORMAT.md` を別リポジトリへ切り出しても残る。
- エージェントが毎セッション読む唯一のファイルであり、ツールを追加せずルールだけで担保する方針と噛み合う。
- このリポジトリは定義一覧のための登録ファイルを置かないことを設計原則にしている（`AGENTS.md` の "Expect repository tools to discover the standard layout without a registry file"）。`language: ja` のような設定ファイルの新設はこの原則に反する。

方法論文書側には言語について何も書かない。沈黙が「利用リポジトリが決めてよい」を意味する。

切り出しの際には次の区別が必要になる。言語の**選択**（散文は日本語）は利用リポジトリに属する。一方、識別子、見出し、表ヘッダー、frontmatter キーを ASCII とすることや、同じ内容を二言語で併記しないことなど、言語に**依存しない**書式制約は、汎用化した方法論文書へ移す候補になる。本作業項目では前者だけを扱い、後者は `AGENTS.md` に置いたままとする。

### Root README

ルートの `README.md` も日本語の正本とし、英語版は設けない。`README.md` と `README.ja.md` に同じ内容を置く二重管理は、本作業項目の方針と矛盾するためである。公開リポジトリに英語の入口がなくなる代償は受け入れ、英語の読み手が現れた時点で翻訳方法を別途検討する。

## Plan
- 言語方針を先に確定させる。文面が決まらないうちに 8,000 行規模の書き換えを始めない。
- `spec/contexts/*` は Context 単位で作業する。言語検査を追加しないため、途中状態で `just check` が失敗する順序制約はない。
- 規範文の書き換えでは REQ ID、`Strength` 列、`Adoption` 列、RFC の参照 URL、TypeSpec シンボルに触れない。変更するのは `Statement` と `Definition` の散文だけとする。
- `spec/SPECIFICATION.md` は全文が英語なので和訳になる。`Reading order` の手順番号と参照パスは維持する。

## Tasks
- [x] T001 [Policy] `AGENTS.md` に `Language` 節を追加し、コードコメントを日本語に変更する。
- [x] T002 [Spec] `spec/SPECIFICATION.md` を日本語化する。
- [x] T003 [Spec] `spec/contexts/*/SPECIFICATION.md` 21 件の日英重複と不自然な混在を解消する。
- [x] T004 [Docs] `tools/**` を除くサブディレクトリの README と `infra/runbooks/*` を日本語化する。
- [x] T005 [Docs] ルートの `README.md` を日本語版で置き換え、英語版と `README.ja.md` を廃止する。
- [x] T006 [Verify] `just check-work-items`、`just check-spec`、`just spec-render`、`just verify` を通す。

## Verification
- `just check-work-items`
- `just check-spec`
- `just spec-render`
- `just verify`

## Risk Notes
- **規範文の意味がずれる**。`Standards` と `Scenarios` の書き換えで MUST / SHOULD の強度や条件が変質する可能性がある。緩和として、`Strength` 列と `Adoption` 列は構造化データなので触らず、参照する RFC の URL も維持する。散文が原典と食い違ったら原典を優先する。
- **保守体制が英語中心に変わる**。その場合は、既存の日本語コメント約 4,746 行を別の作業項目で英訳する必要がある。
- **公開リポジトリの英語での可読性が下がる**。仕様書だけでなくルートの `README.md` も日本語にするため、英語話者向けの入口はなくなる。潜在的な読み手は黙って離脱し、後退自体を観測できない点が本質的な弱さである。この代償を承知の上で、読み手がいない現状ではメンテナーの速度と文書の一貫性を優先する。

## Completion

- **Completed At**: 2026-08-15
- **Summary**: 文書種別ごとの言語方針を `AGENTS.md` に定め、ルート README、21 件のコンテキスト仕様書、ルート仕様書、サブディレクトリの README、運用手順書を自然な技術文書の日本語へ統一した。正確な原語表記が必要な識別子、製品名、プロトコル名、名前付き技術パターンは保持し、再利用可能な `tools/**` は英語のまま維持した。
- **Verification Results**:
  - `git diff --check` - passed。
  - `just check-work-items` - passed（369 ファイル、369 件の依存記録）。
  - `just check-spec` - passed（24 文書、316 操作、17 API タグ、762 TypeSpec シンボル）。
  - `just check-api-compat` - passed（ベースラインに対する破壊的変更なし）。
  - `just check-boundaries` - passed。
  - `just spec-render` - passed（790 ページを生成）。
  - `just spec-diff` - completed（日本語化した規範シナリオの文章差分を確認）。
  - `just verify` - passed（check、Go/UI/ツールのテスト、ビルド、型検査、lint、format check、API compatibility）。
