---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 開発記録の配置と完了操作だけを変更し、製品の利用方法と互換性には影響しない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/check/src/check-work-items.ts
    - tools/check/src/check-spec-impact.ts
    - tools/check/src/markdown-links.ts
    - tools/workspace/src/workspace.ts
    - tools/brief/src/main.ts
    - tools/brief/src/traces.ts
    - tools/work-item-number/src/main.ts
    - tools/render-docs/src/main.ts
    - mise.toml
  tests:
    - tools/check/src/check-work-items.test.ts
    - tools/check/src/spec-impact.acceptance.test.ts
    - tools/check/src/repository-checks.acceptance.test.ts
  stop_before_reading: [backend, frontend]
spec_impact:
  kind: none
  reason: "作業項目の配置と参照の解決を変える。製品の HTTP 応答、認証と認可、永続状態、イベント、通知と規範要件は維持する。"
---

# 未完了と完了の作業項目を同じ深さに分け、完了時の相対リンクの書き換えをなくす

## 動機

[作業項目の形式](../../docs/formats/work-item-format.md#完了記録)は、完了すると work-items 直下から done へ移動するよう定める。
深さが一つ増えるため、docs へのリンクは ../docs から ../../docs へ変える必要がある。
通常の Markdown リンクで作業項目を参照する側にも修正が波及し、完了処理に内容と無関係な差分が増える。
利用者は、ファイル一覧でも未完了と完了を区別できることを求めている。
現在は未リリースであり、当面も未リリースを継続するため、公開後の互換性整備より日々の完了処理を軽くすることを優先する。

## 対象範囲

- 未完了と完了または中止済みの項目を、ファイル一覧で区別でき、同じ深さに置く配置。
- 完了によって変わらないリポジトリ文書への相対リンクと、移動に依存しない作業項目同士の参照。
- 一度だけの既存配置の移行、採番、依存、検査、brief、生成サイト、スキルと規約の同期。

## 対象外

- 全項目を一つのディレクトリへ置き、一覧で status を読まないと区別できなくすること。
- 作業項目の識別子と題名の一括変更、過去の証拠の意味の変更。
- 過去の証拠を現在まで解決する要求の変更は[履歴参照](../active/wi-29959-resolve-completed-work-item-evidence-at-its-historical-revision.md)が扱う。
- リンク修正を目的とした独自の URL 体系や文書管理サービスの導入。

## 設計

work-items/active と work-items/done を同じ深さの兄弟ディレクトリとして採用する。
active は pending と in_progress、done は completed と cancelled を置く。
共通のリポジトリ文書へのリンクは移動の前後で同じ相対パスになる。
分類は status から検査し、同じ項目の複製や手書きの状態索引は作らない。

作業項目同士の参照は、通常の Markdown リンクを移動と同時に機械的に直す案を採用する。
識別子だけを地の文に書く案では GitHub とエディターの Markdown 閲覧でクリックできない。
独自の識別子 URL、転送ファイル、symlink の索引は、閲覧環境の依存や同一記録の複製を増やすため採らない。
`depends_on` は既存の識別子を保ち、リンクの位置とは独立に解決する。
先頭に / を付けて既存のリンク検査を迂回する案は、閲覧環境ごとの解決と参照切れを保証できないため採らない。

`mise run move-work-item -- <id>` は完了または中止の status を確認し、active から done へ移動して、入ってくるリンクと項目間の外向きリンクを更新する。
共通文書への相対リンクは同じ深さなので変更しない。
旧配置の直下にある項目は一度だけ active に移し、リンクの相対位置を補正する。
現在の検査では旧配置と status に反する分類を拒否し、Git 履歴の仕様影響検査だけは旧配置も探索する。
生成サイトは作業項目を掲載しない既存方針を保ち、開発手順のリンクと brief の探索を新配置で検証する。

主要な操作は `relocateMarkdownLinks(source, from, to, moves): string` と `moveWorkItem(root, id): Promise<void>` とする。
移動先の決定とリンクの相対位置は値から計算し、ファイルの読み書きと rename はコマンド側で行う。
製品のモジュール境界、公開パッケージ、作用、テーブル所有者は変更しない。

## 計画

1. 配置に依存する検査、照会、生成と参照を棚卸しする。
2. 同じ深さの配置と参照の方針を決める。
3. 未完了項目と必要な参照を一度だけ移行し、ツールと規約を同期する。
4. 完了と中止への移動で、文書リンクの書き換えが不要なことを確かめる。

## タスク

- [x] T001 [Inventory] 探索は check-work-items、brief、採番、Git 履歴の仕様影響検査に依存する。リンク検査と trace の探索は再帰的であり、生成サイトは作業項目を掲載しない。
- [x] T002 [Design] 兄弟配置と通常の Markdown リンクの自動更新を採用する。製品要件は N/A: 開発ツールの変更である。
- [x] T003 [Tooling] 探索、採番、依存、検査、表示を新配置へ合わせた。`test-tools-file` の新配置探索が 1 件しか収集せず、重複番号を受理する RED を観測し、分類検査とともに GREEN にした。Git 履歴では旧配置から active と done への移動を通す。
- [x] T004 [Migration] 直下の 164 件と、done にあった pending の 1 件を active へ移し、Markdown リンク、規約とスキルを同期した。status と過去の証拠は変更していない。`check-work-items` は 749 件、`check-links` は 1290 文書を検査して成功した。
- [x] T005 [Verify] 完了と中止の CLI 移動、参照定義、画像、表、引用、コードの保持を確認した。リンクの対象保存と本文の往復を 300 入力で検査した。active の探索を外すと 4 テスト、移動のリンク更新を外すと completed と cancelled の 2 テストが失敗した。復旧後の `mise run verify` は成功した。

受け入れ境界と単体境界の製品要件は N/A: 開発記録の移動である。
代替 RED は `mise run test-tools-file -- check/src/check-work-items.test.ts` の新配置の探索と分類検査、および `work-item/src/move.test.ts` のリンクを保つ移動とする。
GREEN 後に同じ検査、`check-work-items`、`check-links`、`check-agent-guidance` を実行する。
変更耐性は active の探索を外す故障、移動に伴うリンク更新を外す故障を同じテストで検出する。

移行中、`wi-305` が done にありながら pending のままだと分類検査で分かった。
現在の status を一次情報として active へ戻し、完了したという証拠を推測して書き足さない。

## 検証

- ファイル一覧で未完了と完了をディレクトリから区別できる。
- 完了時に docs やコードへの相対リンクの本文を変更せず、参照が同じ対象へ解決する。
- 他項目からの参照、depends_on、採番の一意性を移動後も維持する。
- brief と生成サイト、通常の Markdown 閲覧で対象へ到達できる。
- mise run test-tools-file、mise run check-work-items、mise run check-links、mise run check-agent-guidance、mise run verify を実行する。

## リスク

同じ深さへ移すだけでは、通常の Markdown による項目間リンクの参照切れは残る。
外向きと入ってくる参照の両方を検証し、完了時の修正を人が探索し続ける方式を残さない。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff` は main に対する規範仕様の変更なしを示した。
  未完了を active、完了と中止を done の兄弟配置にし、完了処理で共通文書への相対リンクを書き換える手間をなくした。
  項目間の通常の Markdown リンクは移動コマンドで更新し、探索、採番、状態検査と開発手順を同期した。
  既存の 165 件を status に合わせて移行し、過去の判断と証拠、および旧配置の Git 履歴の解決を保った。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run test-tools-file -- work-item/src/move.test.ts` の completed と cancelled の CLI 移動。
  - **要件**: N/A: 開発記録の移動であり、製品の規範要件を変更しない。
  - **観測した失敗**: 未実装の移動コマンドが成功を返しても active のファイルが残り、存在しないという期待に対して実際値が true になった。
  - **検出できる理由**: 正式な CLI からファイルの移動と双方のリンクの解決を観測し、成功を返すだけの処理やリンク更新の配線を外した処理を区別する。
- **単体 RED の証拠**:
  - **テスト**: `mise run test-tools-file -- check/src/check-work-items.test.ts` と `mise run test-tools-file -- work-item/src/move.test.ts`。
  - **要件**: N/A: 製品のドメイン計算を変えない。探索と Markdown 変換の検査を代替境界にする。
  - **観測した失敗**: 旧探索は active を収集せず、2 件の期待に対して 1 件となり、重複する番号も受理した。変換の未実装は `../docs/README.md` を保ち、期待した `../../docs/README.md` を生成しなかった。
  - **検出できる理由**: 一覧の件数、識別番号の衝突、状態と配置の一致を検査する。リンクは対象の保存、コードと不完全な構文の保持を独立した期待値で検査する。
- **変更耐性の結果**:
  active の探索を旧直下に戻すと、探索、重複番号、件数、分類の 4 テストが失敗した。
  移動処理からリンク更新の呼び出しを外すと、completed と cancelled の両テストが外向きリンクの期待で失敗した。
  レビューで見つけた不完全なリンク構文の誤更新も RED を確認し、解析が成功したリンクだけを更新する実装で GREEN にした。
  フォールトを復旧した状態で標準検証を通した。
- **検証結果**:
  - `mise run verify` - 成功。ツール 940 テストを含む標準検証をすべて通過した。
  - `mise run check-work-items` - 749 件の依存、識別子、分類と記録を検査して成功。
  - `mise run check-links` - 1290 文書の Markdown リンクを検査して成功。
  - `mise run check-agent-guidance`、`mise run check-command-map` - 成功。
  - `mise run brief -- wi-35829`、`mise run work-item-number` - 新配置で成功。
  - `mise run render-docs`、`mise run check-rendered-docs`、`mise run check-api-compat` - 成功。1352 ページを生成し、API の破壊的変更はなかった。
  - 七つの視点で変更をレビューし、既存依存による構文解析、計算とファイル操作の分離、最小の公開操作、参照位置だけの移行を確認した。
  - `test-ui-e2e` は N/A: 製品の画面と通信経路は変更しない。文書サイトは描画検査で検証した。
