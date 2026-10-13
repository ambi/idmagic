---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-13
priority: p2
depends_on: [wi-47518-align-shared-document-placement-and-design-records]
change_kind: tooling
spec_impact:
  kind: none
  reason: "図の表現と選択基準を改善する。IdMagic と外部当事者の責任分界、実行単位の配置と通信、認証と認可の結果、永続状態、ドメインイベント、外向きの通知、REQ と TypeSpec の契約は維持する。"
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: "未公開の製品について、開発者が読む設計文書の図、設計フォーマットの基準、文書サイトの描画器と用語検査だけを変える。HTTP 契約、既存データ、運用手順に利用者が観測する差分はない。"
  references: []
initial_context:
  specification:
    - docs/design/architecture/system-boundary.md
    - docs/formats/design-format.md
    - docs/formats/documentation-guide.md
    - docs/formats/work-item-format.md
    - docs/development/specification-first-workflow.md
  typespec: []
  source:
    - tools/render-docs/src/render.ts
    - tools/render-docs/src/main.ts
    - tools/check/src/terminology.ts
  tests:
    - tools/render-docs/src/render.test.ts
    - tools/check/src/terminology.test.ts
  stop_before_reading: [backend, frontend, spec, work-items/done]
---

# 読者の問いに合わせて Mermaid の図を選び、システム境界の既存図を改善する

## 動機

Mermaid には C4 などの表現があるが、既存文書の図は主に flowchart、シーケンス図、ER 図である。
システム境界の図も flowchart で、人、システム、ゲートウェイなどの役割を同じ箱で表している。
表現する対象に合う図種を使えば、読者が責任分界を読み取るための負担を減らせる。

図種や図数を増やすことは目的にしない。
既存図の置換、維持、削除を比較し、追加情報が不要な場合は図も説明も増やさない。
変更する理由がなければ、既存図をそのまま保つ。

## 対象範囲

- [システム境界](../../docs/design/architecture/system-boundary.md)の既存 flowchart を C4Context に置き換え、人、IdMagic、外部システムと利用関係を区別する。
- [設計フォーマット](../../docs/formats/design-format.md)へ、図の必要性、図種、粒度を選ぶ短い基準を追加する。
- 生成サイトの暗色表示で C4 図の関係線とラベルを判別できるよう、描画器のスタイルを補う。
- 変更した図を生成サイトで確認し、必要な要素、関係、日本語ラベルと読みやすさを検証する。

## 対象外

- C4 の全階層、全図種の導入、図数や採用図種数の達成目標。
- runtime.md や logical.md への新規図の追加、既存図と同じ内容を描く別図の併設。
- 下の調査で維持と判断した配置図、ネットワーク図、処理図、ER 図、生成状態図の変更。
- 図の例集、Mermaid 構文の解説、新しい図の台帳、独立した ADR の作成。
- Mermaid の更新、C4 図の関係線とラベル以外の配色とテーマの変更、新しいアイコン配信基盤の導入。
- 製品の責任分界、モジュール構成、依存、実行方式、プロトコルと規範仕様の変更。
- 過去の作業項目にある図の改稿。

## 設計

### 起票時の調査と採否

2026-10-13 に docs/ の Mermaid ブロックを検索したところ、手書き図は flowchart 12 図、sequenceDiagram 4 図、erDiagram 6 図だった。
これとは別に、描画器は状態遷移表から stateDiagram-v2 を生成している。
各文書の図と周囲の説明を確認し、次のように判断した。

| 対象 | 確認した内容 | 判断と理由 |
| --- | --- | --- |
| [システム境界](../../docs/design/architecture/system-boundary.md) | 人、連携アプリケーション、IdMagic、外部の提供者が同じ箱で描かれ、辺には関係の説明がない | 既存の一図を C4Context に置換する。人とシステムを区別し、辺に利用、連携、運用の目的を付けることで、責任表を読む前に関係を把握できる |
| [共通トポロジー](../../docs/design/architecture/deployment.md#共通トポロジー) | API、Worker のレーン、Batch、Seed、スキーマ適用、起動主体と通信を一図で示す | flowchart を維持する。C4Container は実行単位の分担に適するが、この図は起動と配置の関心も含む。変換だけでは理解の改善が明確でなく、別の Container 図を足すと同じ構成の二重管理になる |
| [デプロイプロファイル](../../docs/design/architecture/deployment.md#デプロイプロファイル) | Compose の観測経路、Kubernetes の Service、Deployment、HPA、PDB、NetworkPolicy と、未作成要素の破線がある | flowchart を維持する。C4Deployment や architecture-beta への統一より、通信、制御、未作成の区別を保つ方が、この文書の問いに合う |
| [プラットフォーム](../../docs/design/infrastructure/platform.md#google-cloud) | リソース階層、production の構成、リリースの順序を別々に示す | 維持する。階層、共通プロジェクトの利用、工程順序は異なる関心であり、一つの C4 図へまとめない。配置図と重なる部分だけを理由に、共有リソースや管理関係まで消さない |
| [ネットワーク](../../docs/design/infrastructure/network.md) | キャッシュと拒否の分岐、上流 IdP の戻り、トークン取得の順序、ネットワーク区画を示す | flowchart と sequenceDiagram を維持する。処理分岐と通信順序には既存の表現が合っており、C4Dynamic への置換や境界図の追加は不要 |
| IdManagement の[CSV](../../docs/modules/identity-management/design/csv-transfer.md#プレビューと適用)、[エクスポート](../../docs/modules/identity-management/data-export/README.md#アーキテクチャ)、[監査](../../docs/modules/identity-management/design/audit-events.md#ドメインイベントの配信点) | プレビューと適用、ジョブと成果物、保存後のイベント配信を示す | シーケンス図を維持する。処理順序と確定の前後が読者の関心であり、静的な構成図へ変えるとその情報を失う |
| [データベース](../../docs/design/data/database.md#スキーマの読み方) | カラムを省き、テーブルと外部キーの多重度を示す | ER 図を維持する。classDiagram への置換やカラムの追記は、目的に合わず SQL の複製を増やす |
| [生成の関係](../../docs/development/local-development.md#生成の関係)と生成状態図 | 前者は入力と生成物、後者は規範の状態遷移表を図示する | 維持する。C4 の対象ではなく、状態図を手書きすると規範表と二重管理になる |

今回の採用は既存図一つの置換とし、新しい図は追加しない。
維持する図に、今回の検討結果を説明する文章は追記しない。
この比較と不採用の理由は本作業項目に保つ。

### C4Context に描く範囲

IdMagic を一つのシステムとして描き、利用者と管理者、運用者、連携アプリケーション、上流の IdP とアイデンティティ源、下流の送信先、外部のデータベース基盤と運用基盤を結ぶ。
既存の本文と責任表を根拠に、各関係の目的を日本語で付ける。
ブラウザー、ゲートウェイ、API プロセスなどの内部経路はこの俯瞰図へ描き込まず、既存のランタイム、配置、ネットワーク設計への参照を使う。
図の「利用する」関係は物理的な直接接続を意味しないことを、図の導入で短く示す。
ゲートウェイの担い手と PostgreSQL の具体的な配置については、既存の本文と表にある責任分界を保つ。

同じ図の flowchart 版を併設しない。
責任表は当事者ごとの義務を扱うため保つが、ノード一覧や関係一覧を別の表として足さない。
Context 図を描くために、モジュールや実行単位の境界を再設計しない。

### 図種の選択基準と表示環境

[設計フォーマット](../../docs/formats/design-format.md)に、読者が知りたい関係を本文や表だけで十分に理解できるなら図を増やさないこと、既存図を改善できる場合は置換することを短く定める。
システムと外部当事者には C4、分岐には flowchart、通信順序には sequenceDiagram、関係と多重度には erDiagram、状態には既存の表から生成する状態図を選ぶ基準を置く。
図種ごとの採用義務や、全ページへの図の設置義務は作らない。
C4Container の Container は Docker のコンテナとは区別し、既存の[実行単位の語彙](../../docs/formats/documentation-guide.md#47-architecturedeploymentmdとdesigninfrastructure)に従う。

起票時の tools/package.json は Mermaid 12.1.0 を指定している。
描画器は Mermaid の全ブロックを mermaid.parse で検査し、生成サイトでは securityLevel: strict で描画する。
専用の C4 導入処理を足す前提は置かない。
[Mermaid の C4 文書](https://mermaid.js.org/syntax/c4.html)は C4 を実験的とし、レイアウトの自動調整とテーマ対応にも制約を示しているため、構文検査の成功だけで表示品質を判断しない。
[C4 の公式方針](https://c4model.com/diagrams)も価値のある階層だけを使うとしており、[Context](https://c4model.com/diagrams/system-context)と[Container](https://c4model.com/diagrams/container)の粒度を区別する。

置換後の表示で日本語の欠け、関係ラベルの重なり、配色による判別の悪化が解消できなければ、既存 flowchart のまま人と関係を明示する改善へ戻す。
図の追加で置換を成立させない。
描画器の変更は、下の再検証で判明した暗色表示の関係線とラベルの補正に限る。
この採否条件は表示の検証で確定し、比較結果を本作業項目へ残す。

### 着手時の照合と表示の検証による採否

着手時に先行項目の完了後の文書を照合した。
システム境界の図、本文、責任表は起票時の調査と同じであり、Mermaid も 12.1.0 のままだったため、判断を変える差分はなかった。

C4Context の案は、生成サイトと同じ初期化設定とスタイルで描画し、明色と暗色で確認した。
最初の検証では C4 を不採用と判断したが、その理由を測り直すと、次のとおり確認方法の誤りと描画器の不足が混じっていた。

| 最初の判断 | 再検証の結果 | 扱い |
| --- | --- | --- |
| `UpdateLayoutConfig` の一行あたりの要素数が反映されない | Mermaid の C4 は一行の幅の上限に `screen.availWidth` を使う。ヘッドレスの Chrome の既定の画面幅が狭く 2 列に折り返していただけで、`--screen-info` で画面幅を与えると 3 列に並んだ | 誤り。確認時は画面幅を指定する |
| 関係ラベルが要素や他のラベルと重なる | 3 列の配置で重なりは大きく減り、残りは `UpdateRelStyle` の `$offsetX`、`$offsetY` で避けられる | 図の記述で解消する |
| 暗色表示で関係の線とラベルが背景に沈む | Mermaid の C4 は関係の線とラベルの色を既定で `#444444` の属性として書く。生成サイトのスタイルは flowchart と状態図の線だけを暗色に合わせて上書きし、C4 の関係を対象にしていない | 描画器の不足。下のスタイルを補う |
| 要素の種類が `[Person]`、`[Software System]` と英語で出る | C4 の記法の一部であり、日本語の名前と説明は欠けない | 専門用語として受け入れる |

最初の判断に基づいて設計フォーマットへ「Mermaid の C4 図は使わない」と書いたが、一図と描画器の現状からの一般化だったため取り下げ、C4Context を採用する基準へ改める。

### 描画器のスタイルの補正

生成サイトのスタイルへ、C4 図で既定色（`#444444`）のまま描かれた関係の線を `--diagram-line`、ラベルを `--text` で描く規則を足す。
対象は `aria-roledescription="c4"` を付けた SVG の中の、既定色を属性に持つ要素だけとする。
`UpdateRelStyle` で色を明示した関係は、図の作者の指定を保つため上書きしない。
要素の塗りは C4 の既定の配色のまま、明色と暗色の両方で文字と背景を判別できることを表示で確かめる。

| 候補 | 利点 | 欠点 | 採否 |
| --- | --- | --- | --- |
| 既定色の属性を選ぶ CSS を足す | 既存の flowchart の線の上書きと同じ仕組みで、明色と暗色を CSS 変数で切り替えられる | Mermaid が既定色を変えると効かなくなる | 採用。既定色の変化は生成サイトの目視と、色を固定したテストで見つける |
| `mermaid.initialize` の C4 設定やテーマ変数で色を与える | CSS を足さない | C4 の関係の色はテーマ変数を読まず、初期化の設定でも変えられない | 不採用 |
| 図ごとに `UpdateRelStyle` で色を与える | 描画器を変えない | 明色と暗色で色を切り替えられない | 不採用 |

### 実装中に見つけた C4 の表示の問題と対処

配色を直した後、実ページの確認で次の二つが見つかった。
どちらも C4 図だけに起きる問題であり、暗色対応と同じく描画器の C4 対応として今回の範囲に含めた。

| 問題 | 原因 | 対処 |
| --- | --- | --- |
| 本文の幅で図の文字が読めない大きさになる | Mermaid の C4 は SVG の表示範囲を画面幅から決め、要素の周りに大きな余白を残す。余白ごと本文の幅へ縮めていた | 描画の後、C4 図の表示範囲を描いた中身（`getBBox`）と 16 px の余白へ切り詰める。拡大表示は切り詰めた表示範囲を使う |
| スマートフォンの画面で要素が一列に並び、線が箱を貫く | 一行に並べる幅の上限を `screen.availWidth` から決めるため、狭い画面では一行に一つしか置けない | Mermaid の描画の間だけ `screen.availWidth` を 1600 に固定し、描画後に元の値へ戻す。図はどの画面でも同じ配置になり、狭い画面では他の図と同じく縮小と拡大表示で読む |

`screen.availWidth` の置き換えは、Mermaid が配置に使う値を外から与える手段が他にないために選んだ。
置き換えは描画の間だけとし、元の記述子を戻すことをテストで固定する。

### 用語検査の例外

用語検査は、廃止した設計の単位を指す「コンテキスト」と「Context」を拒否する。
C4 のビュー名「システムコンテキスト図」と Mermaid のキーワード `C4Context` は設計の単位ではないため、既存の「System Context」と同じく理由付きの例外に加えた。

## 計画

1. [wi-47518](wi-47518-align-shared-document-placement-and-design-records.md)の文書と用語の整理を先に取り込む。同じ設計文書とフォーマットを扱うため、その整理後の状態に置換を適用する。
2. 着手時に対象図、責任表、Mermaid のバージョンを再確認し、起票時の判断を変える差分だけを確認する。
3. 生成サイトのスタイルへ C4 の関係線とラベルの暗色対応を、テストを先に失敗させてから足す。
4. 設計フォーマットへ必要性と図種の選択基準を書き、システム境界の一図を C4Context に置き換える。
5. 構文と参照を検査し、生成サイトで画面幅を指定して既存図との比較を終える。採否条件を満たさなければ同じ一図の flowchart を改善する。
6. 実際の採否と検証結果を記録して完了する。

## タスク

- [x] T001 [Docs] 起票時の対象と判断を先行項目の完了後の文書へ照合し、着手時の文脈を記録する。
- [x] T002 [Tooling] C4 図の既定色の関係線とラベルを、生成サイトの線と文字の色で描くスタイルを、`render.test.ts` の RED から加える。実ページの確認で見つけた余白の切り詰めと配置幅の固定、用語検査の例外も同じく RED から加えた。
- [x] T003 [Docs] 図の必要性、置換と維持、図種と粒度の選択基準を設計フォーマットへ追加し、C4Context を使う基準にする。
- [x] T004 [Docs] システム境界の既存図を C4Context に置換し、責任分界と関係の意味を保つ。
- [x] T005 [Verify] 日本語ラベル、配色、通常表示と拡大表示を既存図と比較し、C4 の採否条件に従って最終図を確定する。
- [x] T006 [Verify] 参照と形式、Mermaid の構文、生成サイト、ツールのテストと静的検査を通し、図が増えていないことと規範の維持を確認する。
- [x] T007 [Records] 実際の採否、検証結果と完了を記録し、作業項目を移動する。

## 検証

- `mise run check-work-items`、`mise run check-links` で、変更した記録と文書の参照を検証する。
- `render.test.ts` に、C4 図の既定色の関係線とラベルを上書きする規則と、色を明示した関係を上書きしないことの検査を足し、実装前の失敗を観測する。`mise run test-tools` と `mise run lint-tools`、`mise run typecheck-tools` で変更したツールを検査する。
- `mise run check-rendered-docs` で文書サイトの生成、日本語の文章検査、Mermaid の構文検査を通す。
- `mise run render-docs` で再生成したページをヘッドレスの Chrome で開き、`--screen-info` で画面幅を指定して確認する。ブラウザーの起動はサンドボックス外で実行する。
- 明色と暗色、通常表示と拡大表示で、ラベルの欠けや重なり、関係の判別、ページ内での収まりを確認する。
- 旧図と最終図で、当事者、責任分界、利用と連携の関係を照合する。内部経路を省いた部分は既存の担当文書へたどれることを確認する。
- 差分で図数の増加、同じ内容の二図の併設、規範本文や TypeSpec の変更がないことを確認する。

## リスク

C4 の箱を使うだけでは、粒度と関係の意味が曖昧なままになる。
責任分界を読むための Context 図に絞り、技術的な経路は担当する既存文書へ委ねる。
表示上の制約で読みにくくなる可能性は、既存図との比較と採否条件で扱う。
先行項目で文書や責任の説明が変わるため、起票時の本文を写すのでなく、着手時の現行文書を図の根拠にする。

## 完了

- **完了日**: 2026-10-14
- **要約**:
  `mise run spec-diff` は main に対して「no normative specification change against main」を報告した。REQ、例、状態遷移、標準の行、TypeSpec の宣言に意味上の差分はない。
  システム境界の図を、同じ一図のまま C4 のシステムコンテキスト図へ置き換えた。人、IdMagic、外部システムを C4 の要素で区別し、全ての関係へ目的を付け、ゲートウェイなどの内部経路をランタイム、配置、ネットワークの文書へ委ねた。関係のラベルは `UpdateRelStyle` で要素と線から離した。
  文書サイトの描画器は C4 図に三つの対処を加えた。既定色の関係線とラベルをサイトの線と文字の色で描き、描画の後に表示範囲を描いた中身へ切り詰め、描画の間だけ配置に使う画面幅を固定する。flowchart など他の図の表示は変わらない。
  用語検査に、C4 のビュー名「システムコンテキスト図」と Mermaid のキーワード `C4Context` の例外を加えた。
  設計フォーマットに図の節を加え、図を置く条件、併設より改善を選ぶこと、読者の問いごとの図種と粒度、C4 図のラベル位置の調整と表示の確認を定めた。理由はフォーマットの理由に置いた。
  一度は確認方法の誤りから C4 を不採用とし、Mermaid の C4 図を使わない規則を書いたが、再検証で取り下げた。経緯は設計の節に残した。
- **受け入れ RED の証拠**:
  - **テスト**: `tools/render-docs/src/render.test.ts` の「draws C4 relations left in the default color with the site line and text colors」「crops a rendered C4 diagram to its drawing and leaves other diagrams as Mermaid sized them」「lays out diagrams at a fixed screen width and restores the screen afterwards」。
  - **要件**: N/A: 製品の振る舞いを変えない、文書サイトの描画器と設計文書の変更である。
  - **観測した失敗**: 実装前は、生成した `site.css` に C4 の関係を上書きする規則がなく、描画後の C4 図の `viewBox` が `0 -10 1428 1175` のまま残り、描画中の `screen.availWidth` が 1600 でなく 1024 だったため失敗した。復元を `delete` だけで実装した時点では、描画後の値が `undefined` になり失敗した。
  - **検出できる理由**: 生成サイトと同じ `renderDocumentationSite` の出力と、その `site.js` を happy-dom で実行した結果を比べるため、規則の欠落、切り詰めの呼び出し漏れ、画面幅の固定と復元の欠落を区別できる。
- **単体 RED の証拠**:
  - **テスト**: `tools/check/src/terminology.test.ts` の「設計の単位としての Context と公開言語を指摘し、Go の型と C4 のビュー名は通す」。
  - **要件**: N/A: 検査ツールの例外の追加である。
  - **観測した失敗**: 例外を足す前は、「システムコンテキスト図」を「コンテキスト」、`C4Context` を「Context」として指摘し失敗した。
  - **検出できる理由**: 例外の literal を一つでも落とすと、同じ入力で指摘が出る。
- **変更耐性の結果**:
  手作業で次の故障を注入し、それぞれ局所テストが検出した。
  - 描画後の切り詰めの呼び出しを外す：`render.test.ts` の 1 件が失敗した。
  - 既定色の関係線を上書きする規則の宣言を空にする：`render.test.ts` の 1 件が失敗した。
  - 画面幅の固定を外す、または元の値を戻さない：開発中の RED として `render.test.ts` の 1 件が失敗した。
  変更は TypeScript のツールと Markdown なので、Go のミューテーションテストは適用しない。
- **検証結果**:
  - `mise run test-tools`：描画器と用語検査の変更が検査、描画、着手支援で共有されるため、ツール全体を選んだ。1017 件が成功した。
  - `mise run lint-tools`、`mise run typecheck-tools`：変更したツールのコードを対象に選んだ。整形を適用した後に成功した。
  - `mise run check-rendered-docs`：変更した文書の生成、文章検査、全 Mermaid ブロックの `mermaid.parse` を対象に選んだ。成功した。
  - `mise run check-links`、`mise run check-work-items`、`mise run check-work-item-references`、`mise run check-terminology`：追加したリンク、作業記録、現在状態の文書の参照、用語を対象に選んだ。成功した。
  - `mise run spec-diff`：規範の意味上の差分がないことの確認として選んだ。差分なしだった。
  - `mise run render-docs` で再生成したページを、ヘッドレスの Chrome で `file://` から開き、`--screen-info` で画面幅を指定して確認した。1440 px と 2560 px の画面の明色と暗色、拡大表示、390 px の画面で、関係の線とラベルが判別でき、ラベルが要素と重ならず、どの画面でも同じ配置になった。狭い画面では他の図と同じく縮小して表示され、拡大表示で読める。デプロイメントアーキテクチャの flowchart は従来どおり表示された。
  - 旧図と照合し、利用者と管理者、連携アプリケーション、運用者、上流、下流、PostgreSQL、運用基盤の当事者と関係を保ち、責任表を変えていないことを確認した。docs/ の手書き Mermaid ブロックは 22 のままで、図は増えていない。
