---
context: seeding
updated_at: 2026-08-11
---

# Seeding Specification

## Overview

環境別の seed プロファイル、投入計画、プレビューと適用の安全方針を所有する運用 Bounded Context である。各業務データの意味と永続化は記録を所有する Context に残し、Seeding は公開コマンドインターフェースを通じて依存順の実行を調整する。プロファイルは明示的に選択し、本番では `bootstrap` 以外をフェイルクローズで拒否する。

所有するのは `SeedProfile`、`SeedRequest`、`SeedPlan`、環境ポリシー、ドリフトポリシー、適用順序であり、投入するリソースの意味、検証、永続化は所有しない。この分割により、環境の安全性と適用順序を一元化しながら、不変条件の検査が重複することを避け、Context をまたぐ安全性を 1 か所で検証する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| SeedProfile | seed の内容と生成規則を表す、明示的に選択するプロファイル。`bootstrap` は稼働に必要な最小データだけ、`development` と `test` は既知のサンプル、`performance` は機密情報を含まない合成データを表す。環境名から暗黙には選ばない。 | seed プロファイル |
| SeedPlan | 現在の状態とプロファイルのマニフェストを比較して作る、シークレットを除いた変更計画。プレビューと適用は同じ計画規則を使う。 | seed計画 |
| SeedDrift | seed 管理対象の論理キーについて、現在値がマニフェストの正規値と異なる状態。デフォルトでは手動変更を上書きせず、競合として停止する。 | drift |
| BootstrapSeed | ファーストパーティークライアントなど、サービスの稼働に必要な最小データ。デモ用の資格情報やサンプルテナントのデータは含まない。 |  |
| SeedOperator | 明示したプロファイルを計画または適用するローカル運用者または自動化主体。 |  |
| SeedManifest | seed リソースと決定的な生成器を宣言する、バージョン付き YAML の望ましい状態。データベースのフィクスチャではなく、記録を所有する各 Context の公開コマンドインターフェースへの入力となる。 | seed マニフェスト |
| SeedSecretReference | マニフェストがシークレット値そのものの代わりに保持する、プロバイダー、ロケーター、バージョンの組。解決した値は計画、ログ、エラーに現れない。 | シークレット参照 |

## Design

### Internal Interfaces

#### SeedData
seed を同じ決定的な計画器でプレビューし、適用する内部運用インターフェースである。適用時は記録を所有する各 Context が公開するコマンドだけを呼び、SQL フィクスチャを直接使って不変条件を迂回しない。
- Input invariant: manifest_schema_supported(input.request)
- Input invariant: manifest_profile_matches_request(input.request)
- Input invariant: manifest_paths_are_local_and_contained(input.request)
- Input invariant: input.request.environment in ['staging', 'production'] implies manifest_secret_providers(input.request) == ['file']
- Result invariant: input.request.mode == 'dry_run' implies persistent_state_unchanged()
- Result invariant: reapply_same_request_is_noop(input.request)
- Result invariant: input.request.environment == 'production' && input.request.profile == 'bootstrap' implies production_safe_redirect_uris(input.request.first_party_redirect_uris)
- Result invariant: seed_plan_and_diagnostics_exclude_secret_values(output.plan)

### Environment policy and planning

構成を環境名から推測することは決してなく、リクエストまたは CLI で明示しなければならない。本番が受け付けるのは `bootstrap` プロファイルだけであり、`demo`、`test`、`performance` は書き込み前にフェイルクローズで拒否する。宛先を誤ったリクエストによって、本番にデモ用の資格情報が投入されることを防ぐためである。プレビューと適用は同じ計画器を共有し、同じマニフェスト、生成 seed、シークレットのバージョンで再適用しても何も変更しない。手作業によるドリフトはデフォルトでは競合として扱い、明示的な調停は後続の別契約に委ねる。適用は Context をまたぐ単一トランザクションではなく、依存順に並べた上限付きバッチの冪等なコマンドで行う。`performance` プロファイルのバッチサイズはデフォルト 250、上限 1,000 とする。専用の進捗テーブルは設けず、同じリクエストはプロファイルと生成 seed から導く論理キーと ID によって決定的に再現する。直列化には、リクエストキーごとのプロセス内ミューテックスと、プロセス間では既存の接続に対する PostgreSQL アドバイザリーロックを使用する。

### Seed manifests and secret references

`models.SeedManifest` は、バージョンを持ち厳密に解釈する YAML の望ましい状態である。`manifests_yaml` アダプターが Domain の型へ変換してから、リソースごとの既存処理へ渡す。Domain とユースケースは、パーサー、ファイルシステム、環境変数 API を直接参照しない。`include` で読み込めるのはマニフェストルート配下のローカル相対パスだけとし、深さと総数に上限を設ける。パストラバーサルや注入の余地を避けるため、YAML のマージキー、テンプレート展開、リモート URL、環境変数展開は文法から除外する。シークレット値はマニフェストに直接書かず、`models.SeedSecretReference` で参照する。`env` プロバイダーはどの環境でも使えるが、ステージングと本番で許可するのは `file` プロバイダーだけである。プレビューでは参照を解決できることを検証するが、取得した値を計画、ログ、エラーへ渡すことは一切ない。

### Design Decisions

- `Seeding` は独立した運用 Context であり、記録を所有する各 Context の既存の冪等なコマンドインターフェースを通じて、環境ポリシー、ドリフトポリシー、適用順序を横断的に所有する。投入構成を各 Context へ分散し、Context をまたぐ安全性を検証する単一の地点を失う案は採用しない。
- 入力マニフェストはバージョンを持ち、厳密に解釈する YAML とする。`include` とシークレット参照の文法を制限し、マージキー、テンプレート展開、リモート URL、`${ENV}` の直接展開はいずれも許可しない。同じマニフェスト、生成 seed、シークレットのバージョンでの再適用は、専用の進捗テーブルに頼らず決定的に再現する。

## Scenarios

### REQ-SEEDING-001: 環境別の明示プロファイルが選択される
- ACTOR SeedOperator
- GIVEN `SeedOperator` が環境とプロファイルを明示している
- WHEN `SeedOperator` が `SeedData` を `dry_run` で呼ぶ
- THEN 計画器は環境ポリシーで許可されたマニフェストだけを選ぶ
- THEN レスポンスは機密値を除去した `SeedPlan` を返し、永続状態を変更しない

### REQ-SEEDING-002: 明示したマニフェストまたはプロファイルのデフォルトマニフェストを選択する
- ACTOR SeedOperator
- GIVEN `SeedOperator` が環境とプロファイルを明示している
- WHEN `SeedOperator` がマニフェストのパスを明示して `SeedData` を呼ぶ
  - ALT マニフェストのパスが未指定である → ローダーはプロファイルごとの Repository にあるデフォルトマニフェストを選ぶ
- THEN ローダーは指定したパスのマニフェストと、その配下に収まる `include` を厳密にデコードする
- THEN 計画器はマニフェストに記載された型付きの望ましいリソースを計画する

### REQ-SEEDING-003: マニフェストと指定プロファイルの不一致を拒否する
- ACTOR SeedOperator
- GIVEN `SeedOperator` がリクエストと異なるプロファイルのマニフェストを指定している
- WHEN `SeedOperator` が `SeedData` を呼ぶ
- THEN `SeedData` はシークレットの解決と書き込みの前に `SeedRejectedError` で拒否する

### REQ-SEEDING-004: 不正なマニフェストは書き込み前に拒否する
- ACTOR SeedOperator
- GIVEN マニフェストに未知のキー、重複する論理キー、未対応のスキーマバージョン、`include` の循環、またはルート外のパスがある
- WHEN `SeedOperator` が `SeedData` を呼ぶ
- THEN ローダーはシークレットの解決と書き込みの前に `SeedRejectedError` で拒否する
- THEN 診断にはシークレット値を含めない

### REQ-SEEDING-005: 本番では env シークレットプロバイダーを拒否する
- ACTOR SeedOperator
- GIVEN 環境が本番である
- GIVEN マニフェストが `env` シークレットプロバイダーを参照している
- WHEN `SeedOperator` が `SeedData` を `dry_run` または `apply` で呼ぶ
- THEN `SeedData` はシークレットの解決と書き込みの前に `SeedRejectedError` で拒否する
- THEN 永続状態は変更されない

### REQ-SEEDING-006: 同じ seed を再適用しても何も変更しない
- ACTOR SeedOperator
- GIVEN 同じマニフェスト、生成 seed、シークレットのバージョンで seed を適用済みである
- WHEN `SeedOperator` が同じ `SeedRequest` を再度 `apply` する
- THEN `SeedPlan` のすべての操作は `noop` である
- THEN パスワード履歴と `created_at`、`updated_at` は変更されない

### REQ-SEEDING-007: 本番では development または performance プロファイルを拒否する
- ACTOR SeedOperator
- GIVEN 環境が本番である
- WHEN `SeedOperator` が `development` または `performance` プロファイルを指定して `SeedData` を呼ぶ
- THEN `SeedData` は書き込み前に `SeedRejectedError` で拒否する
- THEN 既知のデモ資格情報は作成されない

### REQ-SEEDING-008: 本番の bootstrap には明示的なリダイレクト URI が必要である
- ACTOR SeedOperator
- GIVEN 環境が本番である
- GIVEN プロファイルが `bootstrap` である
- WHEN `SeedOperator` が `first_party_redirect_uris` を指定して `SeedData` を `apply` する
  - ALT リダイレクト URI が未指定、localhost、または HTTP URI である → `SeedData` は書き込み前に `SeedRejectedError` で拒否する
- THEN ファーストパーティークライアントは指定した URI だけをリダイレクト URI として持つ

### REQ-SEEDING-009: 手動変更によるドリフトは上書きせず競合とする
- ACTOR SeedOperator
- GIVEN seed 管理対象の論理キーが手動で変更されている
- WHEN `SeedOperator` が対応するプロファイルを `apply` する
- THEN `SeedData` は `SeedConflictError` を返す
- THEN 手動変更は維持される

### REQ-SEEDING-010: 部分失敗後に同じリクエストを再実行すると目的の状態へ収束する
- ACTOR SeedOperator
- GIVEN `SeedData` の適用が一部の操作を完了した後に失敗している
- WHEN `SeedOperator` が同じ `SeedRequest` を再度 `apply` する
- THEN 完了済みの論理キーは `noop` と判定される
- THEN 未完了の論理キーだけが適用され、重複なく目的の状態へ収束する
