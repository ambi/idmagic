---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p1
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "現在状態の文書と、それを表示する生成サイトの情報設計を改善する。プロダクトの外部契約、TypeSpec、要求 ID の意味は変更しない。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 既存の開発者向け文書と生成サイトの構成を改善する変更であり、利用者向けのリリース、移行、非推奨、削除の告知は生じない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/render-spec-docs/src/render.ts
    - docs/development/process-metrics.md
    - docs/development/testing.md
    - docs/design/data/database.md
    - docs/design/observability/logging.md
    - docs/README.md
    - docs/standards.md
    - infra/schema/postgres.sql
    - DOCUMENTATION_GUIDE.md
  tests:
    - tools/render-spec-docs/src/render.test.ts
  stop_before_reading:
    - backend
    - frontend
---

# システム文書を、所属と目的が読める構成へ改める

## Motivation

生成仕様サイトのサイドバーでは、「方法論」と「開発」が控えめなグループ見出しである一方、その配下の「仕様書式」と「開発文書」は通常のリンクとして強く見える。
親子関係を逆に読ませる表示である。

「開発プロセスの計測」には `Measurement`、`Planning assumption`、`Evidence classes` が日本語の説明なしに残っている。
この文書は英語の用語を定義する場所ではなく、計測を採用しない現在の判断を説明する場所なので、意味が読める日本語へ改める必要がある。

「テスト方針」は水準とダブルを定めているが、プロパティベーステスト、ファジング、変異テスト、特性化テストをいつ選ぶかと、どの故障を検出するかを読者が見つけられない。
仕様先行ワークフローには前三者の説明があるが、方針の入口から到達できない。

システム文書の「全体規範」と Context の「採用規範」は、外部標準への準拠範囲を表す `standards.md` の実態より広く聞こえる。
標準仕様とその採用範囲であることを、見出し、リンク、用語で明確にする。

「データベース設計」は方針だけで、現行スキーマの全体像と主な関係を読む入口がない。
「ログ設計」も出力形式と禁止する値に留まり、アプリケーションとミドルウェアがどの事象を、どの属性と重大度で記録し、どの信号へ分けるかを定めていない。

Context のサイドバーでは所有者名の除去が日本語の助詞を残し、「の用語集」のように表示される。
さらに Context の API 一覧は見出し「操作」と列「概要」を使い、OpenAPI の `operationId` を概要の代替として出すため、`ListApiTokens` のような内部識別子が説明欄に現れる。

Context の索引では「DataKeys の内部設計」のような日本語題名に、`DataKeys Scenarios` のような英語題名が混在している。
題名と索引リンクを同じ命名規則へそろえる必要がある。

初回のサイドバー修正では、`参照` を常時展開する実装を残した。
また、分類見出しを小さく、配下リンクを左端に表示し、開発文書の子を子として扱わなかった。
このため、トップページでは「参照」だけが展開され、親子関係も逆に読める。

追加確認では、ER 図が現行 PostgreSQL の全テーブルを示さず、データベース設計ページのサイドバーで四段目以降のインデントが失われていた。
API リファレンスは `blob:` URL を基準に OpenAPI の参照を解決しようとしてエンドポイント詳細を開くと失敗し、モデルカタログには説明未記入のプロパティが多数残っていた。
主要な開発文書の題名と見出しにも英語が残っていた。

最終確認では、サイドバーの先頭が設計文書ではなくフォーマットになっていた。
文書サイトの主な入口はシステム全体と各 Context の設計なので、設計文書を最初に示す必要がある。

設計文書を先頭へ移した後も、プロダクト概要、用語集、全体の標準仕様、構造、システム横断シナリオは設計文書の直下に置かれていた。
これらは個別 Context の同種文書に対応するシステム全体の文書なので、コンテキスト文書の中で同じ階層に置く必要がある。

## Scope

- 生成仕様サイトのサイドバーで、グループ見出し、親ページ、子ページの階層が視覚的にも意味的にも区別できるようにする。
- トップページではどのグループも初期展開せず、参照ページの配下だけで「参照」を初期展開する。
- 設計文書は入れ子のリストによるツリーで示し、深さ別の CSS クラスや開閉状態に依存して子ページの位置を決めない。
- Context のシナリオ文書と索引リンクを「<Context> のシナリオ」と表記し、英語題名を残さない。
- 日本語の Context 子ページ名から所有者名と助詞を正しく除き、「用語集」「標準仕様」「設計判断」のように内容だけを表示する。
- Context ページの API 節を「API」に改め、HTTP メソッド、パス、TypeSpec の `@doc` から生成される利用者に読める説明を表示する。
  `summary` と `description` が無い API は `operationId` を説明欄へ流用せず、説明未記入を可視化する。
- `docs/development/process-metrics.md` の未翻訳用語を、用語の役割を失わない日本語へ置き換える。
- `docs/development/testing.md` に、プロパティベーステストとファジング、変異テスト、特性化テストの選択条件、テスト水準との関係、限界、既存ワークフローへの参照を追加する。
- `docs/README.md`、各 Context の `README.md`、`standards.md` と生成サイトの表示を調べ、`standard` を「標準仕様」、採用範囲を「採用する標準仕様」として一貫させる。
- `docs/design/data/database.md` に、`infra/schema/postgres.sql` を正本とする現行スキーマの読み方、表の責務、主要な外部キー関係を示す ER 図を追加する。
  ER 図は現行スキーマの全テーブルを網羅し、列、索引、制約の詳細はスキーマへリンクする。
- サイドバーの深さを任意の階層で一貫して表示し、データベース設計ページで祖先、兄弟、現在位置を正しく示す。
- API リファレンスが公開 OpenAPI ファイルを基準 URL として読み込み、エンドポイント詳細の `$ref` を解決できるようにする。
- モデルカタログのプロパティ説明は TypeSpec の `@doc` を優先し、未記入なら安定した共通フィールドの意味またはプロパティ名に基づく説明を表示する。
- 仕様フォーマット、作業項目フォーマット、仕様先行の開発ワークフロー、リリースの題名と見出しを日本語へ統一する。
- `docs/design/observability/logging.md` を、アプリケーションログとミドルウェアログの設計指針として拡充する。
  事象分類、重大度、必須属性、相関、例外、HTTP と非 HTTP の境界、秘匿、サンプリング、出力量、監査ログとメトリクスとトレースとの使い分けを定める。
- OpenTelemetry Logs Data Model、OpenTelemetry Semantic Conventions、Google SRE、AWS Prescriptive Guidance を参照し、採用する規則と採用しない規則を `DOCUMENTATION_GUIDE.md` の参考表へ出典とともに記録する。

## Out of Scope

- ログの収集基盤、保持期間、閲覧権限、アラート、Loki、Promtail、OpenTelemetry Collector の本番構成を決めること。
- アプリケーションコードのログ出力、ミドルウェア、メトリクス、トレース、監査イベントの実装変更。
- データベースの表、列、索引、制約、移行方式を変更すること。
- API の経路、メソッド、`operationId`、リクエストまたはレスポンスの契約を変更すること。
- 既存の Context 境界、ドメイン用語、要求 ID を改名すること。

## Design

### 生成サイトは文書の種類を四区分で示す

サイドバーの最上位は「設計文書」「フォーマット」「開発文書」「リファレンス」の四区分とする。
利用者が最初にシステム全体と各 Context の設計へ到達できるように、設計文書を先頭へ置く。
「開発」と「開発文書」、「システム」と「システム文書」のように、分類と索引が同じ意味を繰り返す階層は作らない。
索引を持つ「開発文書」と「設計文書」は見出し自体をリンクにし、同名の子リンクを置かない。
Context は全体設計から独立した最上位区分にせず、「設計文書」の中に「コンテキスト文書」として置く。
「コンテキスト文書」の先頭に「システム全体」を置き、プロダクト概要、用語集、全体の標準仕様、構造、システム横断シナリオをその子にする。
要求、アーキテクチャ、設計、検証、運用はシステム全体を分解する設計体系なので、「コンテキスト文書」と同じく設計文書の直下に置く。

サイドバーの各ノードは、一つの分類または一つのページだけを表す。
通常の文書は入れ子の `ul` の子に置き、縦線とインデントで祖先を示す。
現在位置を含む設計上の枝だけ子孫を表示し、無関係な枝は親リンクまでを表示する。
Context はすべての入口を表示し、現在の Context だけ子文書を表示する。

Context の子名は、英語名の後の空白だけを削る現在の規則ではなく、所有者名に続く空白と日本語の助詞「の」を一体として除く。
所有者名を含まない題名は変更しない。

### API の識別子と説明を分ける

`operationId` は生成、参照、実装のための安定識別子であり、利用者に向けた API の説明ではない。
Context ページでは OpenAPI の `summary` を「説明」として表示する。
`summary` がない場合は `@doc` から生成された OpenAPI の `description` を表示する。
両方がない場合は空欄にせず「説明未記入」と表示する。
見出しは、HTTP API の一覧であることを示す「API」とする。

### データベースの図は現行スキーマへの案内に留める

ER 図は現行スキーマの全テーブルと外部キー関係を示す。
列、索引、CHECK 制約の正本は SQL に保ち、図にはテーブル名と関係だけを置く。
スキーマ変更時に `check-schema` と図の網羅検査を同じ work item で行う。
Mermaid を使えるかを生成サイトで確認し、読めない場合は同じ情報を Markdown の関係表で表す。

### ログは診断用途の契約として定める

ログは監査証跡、メトリクス、トレースの代替にしない。
各ログに、事象時刻、重大度、安定した事象名、メッセージ、発生源、相関識別子を持たせる。
リクエスト、ジョブ、バッチ、外部依存、例外、拒否のそれぞれで何を属性にするか、何を記録しないかを表で定める。

OpenTelemetry のログデータモデルをフィールドの意味と名前の基準にし、Google SRE を相関と診断可能性の根拠にする。
AWS の事象分類は、認証、認可、入力検証、設定変更、高リスク操作の点検表として使う。
外部資料の記述を転載せず、このリポジトリが採用する差分だけを書く。

## Plan

1. レンダラーの現在の HTML、CSS、テストを読み、サイドバーと Context API 一覧について、親子の見た目、子ラベル、説明列を固定する RED テストを追加する。
2. TypeSpec の全 HTTP operation に `@doc` があることを確認する。
表示仕様を確定してから、OpenAPI の `summary`、`description`、`operationId` を使い分けるレンダラーを更新する。
3. 文書ツリーと `DOCUMENTATION_GUIDE.md` を調査し、「全体規範」「採用規範」および関連する `standards.md` 表記を「標準仕様」と「採用する標準仕様」に統一する。
4. 計測文書とテスト方針を、ワークフローの既存規則を正本として補強する。
特性化テストは既存挙動を変更前に固定する用途とし、新しい仕様を決める証拠と混同しない。
5. PostgreSQL スキーマから表の責務と主要関係を抽出し、正本との差分を生まない概略 ER 図と本文をデータベース設計へ追加する。
6. 参照資料と既存の観測可能性、セキュリティ、監査の文書を照合し、ログ設計の規則と例を追加する。
ログ出力の実装を変えないことを確認する。
7. 生成仕様サイト、リンク、文書検査を実行し、モバイルを含むレンダリング結果を確認する。

## Tasks

- [x] T001 [Acceptance] サイドバーの階層、Context 子ラベル、API 説明列を検証する RED テストを `tools/render-spec-docs/src/render.test.ts` に追加する。
- [x] T002 [App] Context ページの API 表示を「API」と「説明」に変更し、`summary`、`description`、`operationId` を用途別に使い分ける。
- [x] T003 [App] レンダラーと CSS を変更し、分類見出し、親ページ、子ページの階層を視覚化する。
- [x] T004 [Docs] 計測文書、テスト方針、標準仕様の名称と Context 索引を改稿する。
- [x] T005 [Docs] 現行 PostgreSQL スキーマの説明と概略 ER 図をデータベース設計へ追加する。
- [x] T006 [Docs] 英語の一次資料を根拠に、アプリケーションとミドルウェアのログ設計ガイドライン、採用範囲、参考表を追加する。
- [x] T007 [Verify] レンダラーの単体テスト、仕様検査、生成サイト、リンク検査、標準検証を実行する。
- [x] T008 [Acceptance] ER 図、サイドバー、OpenAPI 参照、モデル説明、日本語見出しの RED 検査を追加する。
- [x] T009 [Docs] ER 図を全テーブルと外部キー関係へ拡張し、主要開発文書の見出しを日本語化する。
- [x] T010 [App] サイドバーを入れ子のリストによるツリーへ再構成し、OpenAPI の基準 URL を修正する。
- [x] T011 [App] TypeSpec の説明を優先しつつ、未記入のモデルプロパティにも意味を表示する。
- [x] T012 [Verify] 追加修正を全体検証し、wi-568 のコミットを amend する。
- [x] T013 [Acceptance] 設計文書がサイドバーの先頭にあることを検査し、表示順を修正する。
- [x] T014 [Acceptance] システム全体の文書を「コンテキスト文書」の「システム全体」へまとめる。

## Verification

- `mise run test-tools-file -- render-spec-docs/src/render.test.ts`
- `mise run check-spec`
- `mise run check-rendered-spec`
- `mise run check-links`
- `mise run check-work-items`
- `mise run verify-spec`

## Risk Notes

- **名称を置換しすぎて、既存の `standards.md` の所有境界を曖昧にする。** 「標準仕様」は外部標準と採用範囲を表す文書に限り、システム要求や設計規則を移さない。
- **ER 図がスキーマの第二の正本になる。** 表、列、制約の完全性を主張せず、正本へのリンクと、更新時の見直し条件を明記する。
- **ログ指針が監査ログまたはメトリクスの責務を奪う。** 事象の用途と保存先を表にし、既存の Audit、監視、トレース設計へリンクする。
- **OpenAPI の要約補完が公開契約の説明と食い違う。** 各 `@summary` は経路、メソッド、シナリオと照合し、識別子を説明として再利用しない。
- **ナビゲーションの CSS がモバイルまたはキーボード操作を壊す。** デスクトップとモバイルの HTML をテストし、`summary` とリンクのフォーカス表示を手動確認する。

## Completion

- **Completed At**: 2026-09-13
- **Summary**:
  `mise run spec-diff` は `main` に対する規範仕様の変更がないことを示した。
  生成サイトの最上位を「設計文書」「フォーマット」「開発文書」「リファレンス」に組み直し、コンテキスト文書を設計文書へ統合した。
  プロダクト概要、用語集、全体の標準仕様、構造、システム横断シナリオを「コンテキスト文書」の「システム全体」へまとめ、個別 Context と同じ位置付けで辿れるようにした。
  PostgreSQL の通常表と `UNLOGGED` 表をすべて ER 図へ載せ、OpenAPI の参照解決、モデル説明、主要見出し、分かりにくい「被覆」の表現も修正した。
- **Acceptance RED Evidence**:
  - **Test**: `documentation-quality.test.ts`、`renderSpecificationSite > loads the published OpenAPI URL so Swagger UI can resolve schema references`、`renderSpecificationSite > keeps every available top-level section visible without disclosure state`、`renderSpecificationSite > places whole-system context documents beside bounded contexts`
  - **Requirement**: N/A: 文書生成と現在状態の説明だけを変更し、プロダクトの規範要求は変更しない。
  - **Observed Failure**: ER 図から通常表と `UNLOGGED` 表が欠け、主要四文書に英語見出しが 44 件残り、API ページは `blob:` URL を Swagger UI へ渡していた。サイドバーではフォーマットが設計文書より先に表示され、システム全体の五文書が設計文書の直下に露出していた。
  - **Detection Reason**: 現行 SQL の表集合、対象文書の見出し、Swagger UI の入力 URL、最上位区分の順序、コンテキスト文書の入れ子を直接比較するため、目視では見落とす欠落、参照基準の誤り、入口の優先順位と所属の逆転を検出する。
- **Unit RED Evidence**:
  - **Test**: `renderSpecificationSite > renders system documents as a directory tree`
  - **Requirement**: N/A: ドメインまたはユースケースの内部境界ではなく、生成 HTML の構造を検査する。
  - **Observed Failure**: ディレクトリ見出しと「概要」リンクが一つの親を二重に表し、四段目のリンクには対応する CSS がなかった。
  - **Detection Reason**: 親ページを一つのリンク、子をその直下の `ul` として検査し、深さ別クラス、重複する「概要」、平坦化した子を検出する。
- **Change-Resistance Results**:
  親子を同じ `ul` に置く変異、OpenAPI を再び `Blob` URL で渡す変異、モデル説明を「説明なし」へ戻す変異は、それぞれ追加したレンダラー検査が検出する。
  ER 図から表を一つ削除する変異と、対象文書へ英語見出しまたは「被覆」を戻す変異は、文書品質検査が検出する。
  設計文書をフォーマットまたは開発文書より後ろへ移す変異は、最上位区分の順序検査が検出する。
  システム全体の五文書をコンテキスト文書の外へ戻す変異は、システム全体と個別 Context の入れ子検査が検出する。
- **Verification Results**:
  - `mise run test-tools-file -- render-spec-docs/src/render.test.ts` - passed
  - `mise run test-tools-file -- render-spec-docs/src/documentation-quality.test.ts` - passed
  - `mise run check-rendered-spec` - passed
  - `mise run check-links` - passed
  - `mise run check-work-items` - passed
  - `mise run spec-diff` - no normative specification change against `main`
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - 26 passed、2 timed out。タイムアウトした二件は `mise run test-ui-e2e-file` による個別再実行で passed
