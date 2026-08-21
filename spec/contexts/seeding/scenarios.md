# Seeding Scenarios

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
