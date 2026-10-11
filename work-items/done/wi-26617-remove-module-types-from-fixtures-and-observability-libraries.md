---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: [wi-13263-verify-unresolved-http-operation-contracts]
change_kind: refactor
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 内部の配置と型の変換だけを変え、計測の名前とラベル、イベント出力、HTTP 応答、永続状態は維持するため、利用者の操作変更や告知は不要である。
  references: []
initial_context:
  specification:
    - docs/modules/system/operations/README.md#REQ-SYSTEM-001
  typespec: []
  source:
    - backend/shared/testing/fixtures_postgres/pgfixtures.go
    - backend/shared/observability/metrics_prometheus/metrics.go
    - backend/shared/events/sinks_console/console_sink.go
    - backend/shared/events/sinks_console/console.go
    - backend/jobs/usecases/metrics.go
    - backend/cmd/idmagic-worker/worker.go
    - backend/oauth2/ports/event_sink.go
    - backend/shared/spec/events.go
    - docs/design/application/backend.md
    - docs/design/application/design-guidelines.md
    - docs/design/architecture/logical.md
    - docs/design/architecture/risks.md
    - docs/modules/jobs/design/architecture.md
    - tools/check/boundary-debt.json
  tests:
    - backend/shared/observability/metrics_prometheus/metrics_test.go
    - backend/shared/events/sinks_console/console_test.go
    - backend/jobs/usecases/runner_test.go
    - backend/jobs/db_postgres/contract_test.go
  stop_before_reading: [frontend, spec, backend/oauth2/usecases]
spec_impact: { kind: none, reason: "テストの固定データを投入するパッケージの置き場所と、計測とイベントの出力が受け取る値の型だけを変える。計測の名前と値、イベントの出力、HTTP の応答、永続状態は変えない。" }
---

# 固定データと観測の共有ライブラリからモジュールの型を外す

## 動機

`tools/check/boundary-debt.json` の `shared-dependency` のうち 6 件は、次の共有ライブラリからモジュールへの依存である。

| 共有ライブラリ | 参照先 | 使っているもの |
| --- | --- | --- |
| `storage/fixtures_postgres` | IdManagement、OAuth2、SigningKeys、Tenancy | 各モジュールの保存先と型を使った固定データの投入。本番の利用元はなく、テストだけが使う |
| `observability/metrics_prometheus` | Jobs | ジョブの型 |
| `events/sinks_console` | OAuth2 | OAuth2 のポートの型 |

## 対象範囲

- `fixtures_postgres` を、テスト専用のパッケージとして扱える置き場所へ移す。
- 計測とイベントの出力が、モジュールの型ではなく共有の値を受け取るようにする。
- 解消した違反 ID を台帳から消す。

## 対象外

- 計測の名前、値、イベントの出力形式の変更。

## 設計

[境界の負債の順位付け](wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D5：共有ライブラリはモジュールの型を必要としない。
- D6：ジョブの種類やイベントの型を計測の値へ変換する意味は、提供側のモジュールが持つ。

固定データの投入は、テスト用の組み立てを置く場所（`backend/shared/http/testing_stack` と同じ扱いの組み立て地点）へ移すか、各モジュールのテスト用パッケージへ分けるかを、着手時に利用元の数で比べる。

### 着手時の境界判断

入力は固定データの全 import、JobsMetrics のシグネチャ、Worker の配線、既存の計測とイベント出力のテストである。
固定データの import は 55 ファイルにあり、すべてテストである。
固定データは各所有モジュールの repository に保存し、SQL とトランザクションの責務は移さない。
D5 と D6 に従い、共有ライブラリへ業務の型や変換を残さない。

| 候補 | 制約と変更時の波及 | 判断 |
| --- | --- | --- |
| 現状の共有ライブラリ | 固定データの組み立てと業務型が共有ライブラリへ漏れる | 不採用 |
| 各モジュールのテスト支援へ分割 | Tenant、User、Group、Client の組み立てを多くの利用元が別々に import する必要がある | 不採用。今回の固定データは一つのテスト前提を組み立てる |
| `backend/shared/testing/fixtures_postgres` を組み立て地点にする | 既存の投入処理を保ち、テストの import だけを変える。本番の入口からは使わない | 採用。業務判断を追加せず、所有モジュールのアダプターを結線する |
| 共有計測のレーンを `string` にし、Jobs のアダプターで変換する | JobsMetrics の型付きポートを保ち、レーンからラベルへの変換が Jobs に閉じる | 採用 |
| ConsoleSink の戻り値を OAuth2 のポートから具象型へ変える | 共通の DomainEvent を受け取り、利用側のポートを Go の構造的適合で満たす | 採用。出力処理とエラーの扱いは維持する |

Jobs の `metrics_prometheus.Metrics` は共有 Metrics を埋め込み、五つのジョブ計測で `ExecutionLane` を文字列に変換する。
`NewMetrics(serviceName, serviceVersion string) (*Metrics, error)` は共有の provider と registry を組み立てる。
Worker の HTTP、監査、shutdown も同じ provider を使う。
既存の特性化テストはパスと本文を保ち、外部テストパッケージから Jobs のアダプターを経由して観測する。
共有の出力には新しい分岐、エラー、イベントを追加しない。

残る仮定は、固定データの利用元が引き続きテストだけであることと、レーンの文字列が現状のラベル値を表すことである。
本番の固定データ投入や、レーンの語彙変更は対象外である。

### 証拠の計画

テストの import を変えるため、テストを変えない構造変更の証拠免除は使わない。
受け入れ RED の代替には、対象の六つの違反を台帳から先に外した `check-boundaries` の失敗を使う。
単体 RED の代替には、既存 ConsoleSink の特性化で意図的に異なる JSON を期待した失敗を使う。
製品結果の追加はないため、新しい主要ユースケースは定義しない。
計測の既存特性化と ConsoleSink の追加特性化を、本番の変更前に成功させて記録する。
Go の構文変異は変更した出力と Jobs アダプターで実行し、変換を切断する故障は手で注入する。

## タスク

- [x] T001 [Design] 固定データの置き場所と、計測とイベントが受け取る値の型を決める。
- [x] T002 [App] 移して利用側を書き換える。
  固定データをテストの組み立て地点へ移し、55 ファイルの import を更新した。
  既存の投入処理の本文は移動前と一致する。
  Jobs の型付きポートから共有計測へレーン文字列を渡し、Worker は同じ provider を使う。
  ConsoleSink は具象型を返し、共通の DomainEvent を受け取る。
  ConsoleSink、既存計測、Jobs アダプターの race テストは成功した。
- [x] T003 [Tooling] 解消した違反 ID を台帳から消す。
  `check-boundaries` と `check-boundary-debt-ratchet -- main` が成功し、共有依存の違反は 16 件から 10 件に減った。
- [x] T004 [Verify] 変更を検証する。
  完了した wi-13263 を専用ブランチへ反映し、新しく組み合わさったコードの全体検証が成功した。

## 検証

- `mise run test-go-package -- ./backend/shared/events/sinks_console`：イベント出力とエラーの扱い。
- `mise run test-go-package -- ./backend/shared/observability/metrics_prometheus`：既存の計測名、ラベル値、カーディナリティを Jobs アダプター越しに観測する。
- `mise run test-go-mutation -- <変更パッケージ>`：共有計測、ConsoleSink、Jobs アダプターの構文変異。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet -- main`：六つの負債の削除と、新しい違反の不在。
- `mise run verify`：固定データの利用元と Worker、本番の組み立てへの波及が広いため、全 Go の race 検査と標準の集約検査を選ぶ。
- `mise run check-links`：組み立て地点の設計と移動後の作業記録の参照。

## リスク

- 計測の値の型を変えると、ラベルの値が変わり監視の設定が合わなくなる。
  変更の前後で出力する計測の名前とラベルを比べる。

## 実装結果と完了の前提

2026-10-11 に実装と局所検証を行った。
`spec-diff` は `no normative specification change against main` だった。
製品結果、公開 HTTP 契約、イベント出力と永続状態の変更はない。

- **受け入れ RED の代替証拠**：実装前に六つの違反を台帳から外し、`check-boundaries` が固定データの四つ、計測の一つ、イベント出力の一つをすべて拒否した。製品要件の追加はなく、配置と依存の規則を検査する代替証拠である。
- **単体 RED の代替証拠**：`TestCharacterizeConsoleSinkOutput` で `{}\n` を期待すると、実際の At、occurredAt、type を含む JSON と異なるため失敗した。観測した一行の JSON と、エンコード失敗時に無出力で nil を返す振る舞いを期待値へ固定し、本番コードを変えないコミットへ残した。
- **変更耐性**：ConsoleSink は変更前後とも一つの変異を検出した。Jobs アダプターも一つの変異を検出した。共有計測は 22 個中 21 個を検出し、無変更の `BeginHTTPRequest` の in-flight 減算を加算へ変える変異が生存した。この HTTP 指標の検査不足は対象外であり、等価な変異とは判定していない。レーンを bulk に固定する手動の故障では、`TestWorkerJobMetricsPreserveLaneLabelsAndValues` の latency_sensitive と default が失敗し、修復後は全レーンで成功した。変異ツールの canary は初回とキャッシュ済みの判定とも成功した。
- **局所検証**：ConsoleSink の race テスト 6 件、既存計測の 2 件、Jobs アダプターの親と子を合わせた 4 件が成功した。Jobs の検査は三つのレーンで計測名、時間の秒換算、結果、再試行、queued と running の深度を比較する。`lint-go`、`lint-tools`、`check-spec`、`check-links`、境界検査と負債の増加防止検査も成功した。
- **全体検証**：`mise run verify` で Go テスト 6,140 件は成功し、失敗、スキップ、未実行は 0 件、DB 依存の 39 パッケージは verified だった。tools の 1,018 件と UI の 698 件、UI のビルドと型検査も成功した。当初の JSON 整形と HTTP ボディ指定の lint 指摘は修正し、各 lint を再実行して成功した。
- **過去の未達**：全操作を保守的に選ぶため、282 操作の未解析の HTTP 契約に同じ実行の成功した契約テストがなく、初回の `verify` は失敗した。この不足を担当する [wi-13263](wi-13263-verify-unresolved-http-operation-contracts.md) は、2026-10-12 に全操作の成功した契約証拠と標準検証を確認して完了した。ゲートを維持したまま、前提のコミットへ rebase して本項目の最終検証を行う。
- **レビュー**：七つの視点で差分を確認した。レーン変換を Jobs に閉じ、既存の型付きポートを保ち、イベント出力は共通型だけを受け取る。固定データの組み立てと所有モジュールの永続化は分離し、追加した仕組みは実際の Worker の配線だけに使う。既存の特性化本文と投入処理を維持し、出力検査で変換の切断を検出する。

## 完了

- **完了日**: 2026-10-12
- **要約**:
  固定データの投入をテストの組み立て地点へ移し、55 ファイルの import を更新した。
  Jobs が型付きのレーンを共有計測の文字列へ変換し、ConsoleSink は共通のイベントと具象型だけを使う。
  共有ライブラリからモジュールへの六つの依存違反を解消し、計測名と値、イベントの出力、投入処理を維持した。
  `spec-diff work-item/wi-13263` は規範仕様の変更なしだった。
  完了した前提の 0c1aadf65 へ rebase し、既存の契約証拠不足を解消した構成で最終検証を確認した。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-boundaries`。
  - **要件**: N/A: 製品結果の追加ではなく、共有ライブラリの依存規則を満たす構造変更である。
  - **観測した失敗**: 実装前に対象の六つの違反を台帳から外すと、固定データの四つ、計測の一つ、イベント出力の一つを拒否した。
  - **検出できる理由**: 許容リストの削除だけでは通らず、実際の import が境界規則を満たすことを要求する。
- **単体 RED の証拠**:
  - **テスト**: `TestCharacterizeConsoleSinkOutput`。
  - **要件**: N/A: 共通のイベント出力の既存の振る舞いを保つ特性化である。
  - **観測した失敗**: 期待を空の JSON オブジェクトの一行へ変えると、実際の At、occurredAt、type を含む JSON と異なり失敗した。
  - **検出できる理由**: 実際の出力を固定し、型の依存を外す過程でイベントの送受信表現が変わる故障を検出する。
- **変更耐性の結果**:
  ConsoleSink と Jobs のアダプターは、それぞれ一つの構文変異を検出した。
  共有計測は 22 変異中 21 変異を検出した。
  生存した変異は変更していない BeginHTTPRequest の in-flight 減算を加算へ変えるもので、等価な変異ではなく対象外の既存の検査不足として記録する。
  レーンを bulk へ固定する手動故障は latency_sensitive と default の検査が検出し、復旧後は全レーンが成功した。
  既存の特性化は本番変更の前の別コミットへ残し、canary の分類の検算も成功した。
- **検証結果**:
  - rebase 後の `mise run verify` が成功した。Go の race 検査は 6,357 テストが成功し、失敗、スキップ、未実行は 0、DB の 39 パッケージは verified、282 操作すべてに同じ実行の成功した契約証拠があった。
  - tools の 1,020 テストと UI の 698 テスト、各 lint、型検査、UI のビルドが成功した。
  - 境界検査と負債の増加防止検査が成功し、共有依存の違反は 16 件から 10 件へ減った。
  - 新しく組み合わさった差分を七つの視点で再確認した。特性化と投入処理を保ち、型の変換は Jobs に閉じ、共有の出力へモジュールの依存を戻していない。
  - `spec-diff work-item/wi-13263` は規範仕様の変更なしを確認した。局所検証と変更耐性の検査は実装結果欄の成功を使い、rebase で変えた組み合わせは全体の race 検査で確認した。
