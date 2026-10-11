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
  reason: "初回公開前の開発用検証だけを変更し、既存データと製品の利用契約を維持する。開発規約と検証設計は今回更新する。"
  references: []
initial_context:
  specification:
    - docs/development/task-context.md
    - docs/development/specification-first-workflow.md
    - docs/development/coding-style.md
    - docs/development/testing.md
    - docs/development/writing-language.md
    - docs/development/continuous-integration.md
    - docs/development/release.md
    - docs/formats/work-item-format.md
    - docs/formats/documentation-guide.md
    - docs/formats/design-format.md
    - docs/design/verification/README.md
  typespec: []
  source:
    - tools/README.md
    - tools/check/src/runner.ts
    - tools/check/src/check-contract-drift.ts
    - tools/check/src/check-status-drift.ts
    - tools/check/src/contract-drift.ts
    - tools/check/src/status-drift.ts
    - tools/check/src/repository-inputs.ts
    - tools/check/src/registry.ts
    - tools/check/src/normative-coverage.ts
    - tools/spec-review-candidates/src/candidates.ts
    - tools/spec-review-candidates/src/main.ts
    - tools/security-test-gap-report/src/report.ts
    - tools/security-test-gap-report/src/main.ts
    - tools/workspace/src/workspace.ts
    - backend/shared/storage/testing_postgres/pgtest.go
    - mise.toml
    - .github/workflows/idmagic-ci.yaml
  tests:
    - tools/check/src/runner.test.ts
    - tools/check/src/contract-drift.test.ts
    - tools/check/src/status-drift.test.ts
    - tools/security-test-gap-report/src/report.test.ts
  stop_before_reading:
    - backend/authentication
    - frontend/src
spec_impact:
  kind: none
  reason: "検証結果の分類と CI の必須検証の判定を変える。製品の HTTP 応答、認証と認可、永続状態、ドメインイベント、外向きの通知は維持し、検出した製品の不整合は別項目で是正する。"
---

# 検査成功と未検証を区別し、必須検証の未実行を CI で拒否する

## 動機

契約の差分検査は、解析できた操作に差分がなければ成功するが、解析できない操作も残る。
2026-10-10 の調査では、契約本文の比較は 342 操作中 183 操作、状態コードの完全な解析は 94 操作だった。
これらは調査時点の値であり、着手時に再計測する。
PostgreSQL の[テスト基盤](../../backend/shared/storage/testing_postgres/pgtest.go)は、起動できない環境で DB テストをスキップして成功を維持する。
AI が終了コードだけを読めば、対象の契約と永続化が検証済みだと誤解し得る。

現在は未リリースであり、当面も未リリースを継続する。
公開済みリリースとの互換性検査の優先度は低いが、日々の開発で未実行や解析対象外を成功と取り違えない仕組みは今から必要なので p1 とする。

## 対象範囲

- 契約本文と状態コードの検査結果に、検証済み、解析対象外、検査失敗を操作ごとに区別して残す。
- 規範 ID への参照の存在、対応テストの実行結果、スキップ、変更耐性の証拠を異なる事実として表示する。
- CI では必須の DB 検証を実行できなければ失敗とし、手元で許容するスキップも未検証として明示する。
- 変更した操作が静的解析の対象外である場合、契約テストなどの代替証拠を要求する範囲と判定を定める。
- [テスト方針](../../docs/development/testing.md)、[検証設計](../../docs/design/verification/README.md)、CI の結果表示を同じ区分へそろえる。

## 対象外

- 製品コードの細かな不具合の是正。
- すべての動的な応答を静的解析すること。
- 解析対応の拡張と既存違反の削減は[既存の検査拡張](../active/wi-12283-detect-undeclared-errors-and-unread-operations.md)が扱う。
- 公開済みリリースとの互換性ゲート。
- 既存 UI テストの並行負荷時の不安定さの是正。今回の期待結果は変更せず、失敗したファイルの局所検査と同時実行数を抑えた全体検証で確認する。

## 設計

既存の検査器とテスト結果を一次情報とし、結果の対象、実行状態、未検証の理由、代替証拠を派生表示する。
参照の存在からテスト成功を推定せず、成功したテストから仕様の完全性や検出能力を推定しない。
判定できない結果は合格へ丸めず、必須検証なら失敗、任意の調査なら未検証として報告する。
結果は対象の Git revision と検査条件に対応付け、別の変更で得た証拠を流用しない。

静的検査は `OperationVerification { operationId, state, reasons }` を返す。
`state` は `verified`、`unverified`、`failed` とし、差分があれば失敗、解析が不完全なら未検証とする。
契約本文の未解決理由は操作ごとに重複を除いて集計する。
通常の検査の終了コードは検出した違反だけで決め、解析対象外は常に件数と理由を残す。

Go テストは既存の pgtest を変更せず、標準タスクの runner が `go test -json` のイベントを保存する。
`TestExecution { package, name, state }` は成功、失敗、スキップ、未実行を区別し、子テストのスキップを親の成功へ丸めない。
CI は `VERIFICATION_REQUIRE_DB=1` を指定し、pgtest を使うパッケージのスキップ、DB 利用不可の出力、テストの未実行を失敗にする。
手元の任意モードは同じ事実を未検証として表示する。
テストの実行結果を確かめる標準タスクは結果キャッシュを使わず `-count=1` で実行する。

変更操作は指定した基準から作業ツリーまでの差分から保守的に選ぶ。
本番 Go の変更は対応する backend モジュールと、それを推移的に使うモジュールの操作を対象にする。
shared、構成入口、依存宣言、TypeSpec、DB スキーマの変更は全操作を対象にする。
ハンドラーを解決できない操作も変更対象から落とさない。
テストだけ、文書だけ、tools だけの変更は製品の変更操作を増やさない。
関数単位の影響推定は採らず、モジュール内の対象が広がることを制約として公開する。

静的解析が不完全な変更操作の代替証拠は、Go テストの直上に置く `//verify:operation <operationId>: <検査内容>` と、そのテストの同一実行での成功を結び付ける。
このディレクティブは契約本文と状態コードの両方を観測する契約テストに限る。
実行結果の集約 `evaluateVerification(operations, changedOperations, tests, alternatives, requireDB)` は値の上の計算とし、Git、ファイル、Go プロセスは runner が扱う。
失敗した静的検査を代替証拠で免除せず、関係のない操作の未検証では停止しない。

結果には Git revision、作業ツリーの fingerprint、基準 revision、コマンドと DB モードを保存する。
規範 ID への参照は既存の宣言と引用の検出器から導き、テスト結果との対応を別フィールドにする。
変更耐性はテスト成功から推定せず、独立の証拠がなければ未収集と表示する。
手書きの被覆台帳や、別の実行から持ち込んだ成功結果は使わない。

再計測結果は契約本文 183/342、状態コード完全解析 94/342、部分解析 247、未到達 1 だった。
拒否後の作用のソース調査は 243 テスト中 4 テストに読み戻しの候補がなかった。
これらの件数は実行成功と検出能力を意味しない。
操作ごとの重複を除いた結果は、契約本文 verified 196、unverified 146、状態コード verified 94、unverified 248 である。
従来の本文の件数は、一つの操作に複数の未解決理由があると重複して差し引いていた。
解析器の対応範囲は変更していない。

全体検証で、OAuth2 token の DB テストに `TestMain` がなく、DB の利用可否と無関係に常にスキップしていたことが判明した。
製品の振る舞いと期待結果は変更せず、既存の共有契約を実行するための pgtest の起動配線だけを追加する。
これは必須 DB モードを運用するためのテスト基盤の修正として対象に含める。
生成した生のテストイベントは、既存の使い捨て資格情報を表示する場合がある。
追跡しない派生結果として既存のシークレット走査の除外へ加え、入力となるソースの走査は維持する。

| 案 | 判断 |
| --- | --- |
| 既存検査の結果を構造化して集約する | 採用候補。対象外と未実行を、結果の一次情報から区別できる |
| 解析対象外を一律に失敗とする | 動的応答を含む既存検査を運用できなくなるため採らない |
| 文書へ注意書きだけを追加する | AI と CI の判定が変わらないため採らない |

## 計画

1. 既存検査と DB テストの成功、対象外、スキップの条件を再計測する。
2. 必須検証と任意の調査の区分、変更した操作への代替証拠、結果の形式を決める。
3. 未実行を成功扱いする誤った結果で RED を確認し、集約と CI の判定を実装する。
4. 対応する規約、結果表示、実行タスクを同期する。

## タスク

- [x] T001 [Design] 検査結果とテスト結果の区分、代替証拠、必須 DB モードを決める。再計測に `check-contract-drift`、`check-status-drift`、`report-security-test-gaps` を使い、上記の結果を確認した。
- [x] T002 [Acceptance] DB 起動失敗、テストのスキップ、変更した操作の解析対象外を検出する RED を確認する。`test-tools-file -- verification/src/main.test.ts` で必須 DB のスキップ時に終了コード 0 が返り、期待値 1 に対して失敗した。`test-tools-file -- verification/src/evidence.test.ts check/src/operation-verification.test.ts` では未実行を成功へ丸める判定、子のスキップ、代替証拠なし、解析対象外を検証済みと数える判定を拒否した。規範 ID は N/A: 開発用検査器の判定である。
- [x] T003 [Tooling] 結果の構造化と、必須検証の未実行を拒否する CI 判定を実装する。局所検査は `test-tools-file -- verification/src check/src/operation-verification.test.ts`、`typecheck-tools`、`lint-tools`。プロセス統合テストは任意 DB、必須 DB、変更操作の代替証拠の欠落と成功、子のスキップ、入力の実行中変更を確認した。`test-go-package -- ./backend/saml/db_postgres` では実 DB の 10 テストが成功し、DB verified、スキップ 0 を観測した。
- [x] T004 [Docs] 参照の存在、実行結果、検出能力の意味を規約と結果表示へ反映する。`mise run -c check-spec ::: check-boundaries ::: lint-repo ::: test-tools` が成功した。検証設計は必要な証拠と合否、テスト方針は区分と操作引用の記法、CI 文書は実行モードと結果保存を担当する。
- [x] T005 [Verify] 正常な実行と未検証の分類を検証する。`VERIFICATION_REQUIRE_DB=1 MISE_JOBS=2 mise run verify` が成功した。Go 6,123 件、ツール 1,018 件、UI 単体 709 件が成功し、DB 39 パッケージが verified、Go のスキップは 0 件だった。集約の配線を外すと、プロセス統合テスト 2 件が終了コードの期待値で失敗し、復元すると成功した。`mise run spec-diff` は規範仕様の差分なしと報告した。

## 検証

- 規範要件と製品の受け入れ境界は N/A: 開発用検査器の変更であり、製品仕様を変えない。代替検査は `mise run test-tools-file -- check/src/operation-verification.test.ts` と `mise run test-tools-file -- verification/src/evidence.test.ts` および runner のプロセス統合テストとする。
- Go の製品ソースと pgtest は変更しないため、特性化と Go の構文変異器は適用しない。集約の配線を外すフォールト注入で検出能力を確認する。発見した token DB テストの起動配線は `TestMain` の接続だけを変更し、既存の `TestRefreshPersistenceContract` が実行されることを確認する。
- mise run test-tools-file で結果分類と集約の対象テストを実行する。
- mise run check-contract-drift、mise run check-status-drift、mise run report-security-test-gaps で現行結果を比較する。
- DB を起動できない条件で、CI の必須モードは失敗し、手元の任意モードは未検証を表示する。
- 規範 ID を名指すだけの未実行テストを、実行済みの証拠へ昇格させない。
- 代替証拠のない変更操作は必須判定を通らず、無関係な操作の解析対象外を理由に全検査を停止しない。
- mise run check-work-items、mise run verify を実行する。

## リスク

スキップや動的応答を一律に拒否すると、検証できる範囲まで使えなくなる。
未検証の事実と、それが今回の変更を止める条件を分け、変更対象と必須境界から判定する。
検証できる範囲を減らして成功率を上げない。

## 完了

- **完了日**: 2026-10-11
- **要約**:
  `mise run spec-diff` は main に対する規範仕様の差分なしを報告した。
  製品の HTTP 応答、認証、認可、永続状態、ドメインイベント、通知は変更していない。
  契約検査の結果を操作ごとの検証済み、未検証、失敗へ分類し、重複した未解決理由で件数を過少にしないようにした。
  Go の実行イベントと規範への引用を別々に保存し、未実行と子のスキップを成功へ昇格させない。
  CI は DB の未実行と、代替証拠のない解析対象外の変更操作を拒否し、結果を summary と artifact に残す。
  結果は revision、入力の fingerprint、基準とコマンドと DB モードに対応付け、変更耐性は別の証拠として扱う。
  常にスキップしていた OAuth2 token の共有 DB 契約は、起動配線を追加して実行できるようにした。
- **受け入れ RED の証拠**:
  - **テスト**: `tools/verification/src/main.test.ts` の「実プロセスの DB スキップは任意モードで未検証、必須モードで失敗になる」。`mise run test-tools-file -- verification/src/main.test.ts` で実行した。
  - **要件**: N/A: 製品の規範要件を変えない開発用検証器であり、実際の runner の終了コードを代替境界にした。
  - **観測した失敗**: DB 利用不可の出力と skip がある必須モードでも終了コード 0 を返し、期待値 1 に対して失敗した。
  - **検出できる理由**: 同じ Go プロセスの JSON 出力を使い、任意モードの未検証表示と必須モードの拒否を独立して観測する。別の結果ファイルから成功を持ち込めない。
- **単体 RED の証拠**:
  - **テスト**: `tools/verification/src/evidence.test.ts` の「規範参照だけがあり実行イベントがないテストは未実行のまま残る」と子の skip、DB 必須判定、代替証拠のテスト。`tools/check/src/operation-verification.test.ts` の操作別分類も同じ `test-tools-file` で実行した。
  - **要件**: N/A: 検証状態の分類と集約の判定を固定するツールのテストである。
  - **観測した失敗**: 未実行が passed、解析対象外が verified へ昇格し、代替証拠の欠落と必須 DB の未検証を失敗へ反映しない判定で 7 件が失敗した。
  - **検出できる理由**: 引用の存在を保ったまま実行イベントを欠落させる、親の pass と子の skip を組み合わせる、無関係な操作を未検証にするなど、異なる事実を合格へ丸める誤りを値の上で区別する。
- **変更耐性の結果**:
  runner の `evaluateVerification` への配線を一時的に外すと、DB 必須モードと変更操作の代替証拠のプロセス統合テストがそれぞれ期待値 1、実際値 0 で失敗した。
  配線を復元した実行は 2 件とも成功した。
  子テストの深さ 1〜32 に対するスキップの無効化と、64 組の JSON 往復と不正な形の拒否も成功した。
  Go の製品ソースは変更しておらず、追加した Go コードは既存 DB テストの起動配線だけなので、構文変異器は適用していない。
  その配線がない状態では実 DB を使える環境でも 1 件がスキップされ、追加後には同じ共有契約 1 件が成功した。
- **検証結果**:
  - `VERIFICATION_REQUIRE_DB=1 MISE_JOBS=2 mise run verify` - サンドボックス外で成功。Go 6,123 件（子テストを含む）、ツール 1,018 件、UI 単体 709 件が成功。DB 39 パッケージが verified、Go のスキップ、失敗、未終了は 0 件。
  - `mise run report-verification` - 同じ入力を current、DB 必須モードを true、ゲートの失敗を 0 と表示。関数に対応付けた規範引用 1,253 テストが passed。
  - `mise run check-contract-drift`、`mise run check-status-drift` - 差分 0 件。契約本文は verified 196、unverified 146、状態コードは verified 94、unverified 248。解析対象外は解消したと主張しない。
  - `mise run report-security-test-gaps` - ソース調査の候補 243 件、不足候補 4 件を表示し、実行と変更耐性が未収集であることを明示。
  - `mise run test-tools-file -- verification/src`、`mise run typecheck-tools`、`mise run lint-tools`、`mise run check-spec`、`mise run check-boundaries`、`mise run check-work-items` - 成功。
  - `mise run spec-diff` - main に対する規範仕様の差分なし。
  - 最初の全体検証はサンドボックスのポートと共有メモリの制限で HTTP テストが失敗し、DB の未検証を記録した。Go lint のパッケージ読み込みも制限外の再実行で成功した。
  - 制限外の通常並行実行では、生成ログに含まれるテスト用の値のシークレット検出と、既存の AccountDataPage の UI テスト 1 件の失敗があった。派生ログの除外後の `lint-repo` と対象 UI ファイルの 5 件は成功し、期待結果を変えず同時実行数 2 の全体検証が成功した。
  - ブラウザーの振る舞いへ到達する変更がないため `test-ui-e2e` は実行していない。remote CI、push、main への取り込みは実行していない。
