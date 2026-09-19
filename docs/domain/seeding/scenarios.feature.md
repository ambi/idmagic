# Feature: Seeding のシナリオ

## Rule: REQ-SEEDING-001 環境別の明示プロファイルが選択される

Primary actor: `SeedOperator`

### Example: EX-SEEDING-001-01 通常経路

- Given `SeedOperator` が環境とプロファイルを明示している
- When `SeedOperator` が `SeedData` を `dry_run` で呼ぶ
- Then 計画器は環境ポリシーで許可されたマニフェストだけを選ぶ
- Then レスポンスは機密値を除去した `SeedPlan` を返し、永続状態を変更しない

## Rule: REQ-SEEDING-002 明示したマニフェストまたはプロファイルのデフォルトマニフェストを選択する

Primary actor: `SeedOperator`

### Example: EX-SEEDING-002-01 通常経路

- Given `SeedOperator` が環境とプロファイルを明示している
- When `SeedOperator` がマニフェストのパスを明示して `SeedData` を呼ぶ
- Then ローダーは指定したパスのマニフェストと、その配下に収まる `include` を厳密にデコードする
- Then 計画器はマニフェストに記載された型付きの望ましいリソースを計画する

### Example: EX-SEEDING-002-02 マニフェストのパスが未指定である

- Given `SeedOperator` が環境とプロファイルを明示している
- When `SeedOperator` がマニフェストのパスを明示して `SeedData` を呼ぶ
- But マニフェストのパスが未指定である
- Then ローダーはプロファイルごとの Repository にあるデフォルトマニフェストを選ぶ

## Rule: REQ-SEEDING-003 マニフェストと指定プロファイルの不一致を拒否する

Primary actor: `SeedOperator`

### Example: EX-SEEDING-003-01 通常経路

- Given `SeedOperator` がリクエストと異なるプロファイルのマニフェストを指定している
- When `SeedOperator` が `SeedData` を呼ぶ
- Then `SeedData` はシークレットの解決と書き込みの前に `SeedRejectedError` で拒否する

## Rule: REQ-SEEDING-004 不正なマニフェストは書き込み前に拒否する

Primary actor: `SeedOperator`

### Example: EX-SEEDING-004-01 通常経路

- Given マニフェストに未知のキー、重複する論理キー、未対応のスキーマバージョン、`include` の循環、またはルート外のパスがある
- When `SeedOperator` が `SeedData` を呼ぶ
- Then ローダーはシークレットの解決と書き込みの前に `SeedRejectedError` で拒否する
- Then 診断にはシークレット値を含めない

## Rule: REQ-SEEDING-005 本番では env シークレットプロバイダーを拒否する

Primary actor: `SeedOperator`

### Example: EX-SEEDING-005-01 通常経路

- Given 環境が本番である
- And マニフェストが `env` シークレットプロバイダーを参照している
- When `SeedOperator` が `SeedData` を `dry_run` または `apply` で呼ぶ
- Then `SeedData` はシークレットの解決と書き込みの前に `SeedRejectedError` で拒否する
- Then 永続状態は変更されない

## Rule: REQ-SEEDING-006 同じ seed を再適用しても何も変更しない

Primary actor: `SeedOperator`

### Example: EX-SEEDING-006-01 通常経路

- Given 同じマニフェスト、生成 seed、シークレットのバージョンで seed を適用済みである
- When `SeedOperator` が同じ `SeedRequest` を再度 `apply` する
- Then `SeedPlan` のすべての操作は `noop` である
- Then パスワード履歴と `created_at`、`updated_at` は変更されない

## Rule: REQ-SEEDING-007 本番では development または performance プロファイルを拒否する

Primary actor: `SeedOperator`

### Example: EX-SEEDING-007-01 通常経路

- Given 環境が本番である
- When `SeedOperator` が `development` または `performance` プロファイルを指定して `SeedData` を呼ぶ
- Then `SeedData` は書き込み前に `SeedRejectedError` で拒否する
- Then 既知のデモ資格情報は作成されない

## Rule: REQ-SEEDING-008 本番の bootstrap には明示的なリダイレクト URI が必要である

Primary actor: `SeedOperator`

### Example: EX-SEEDING-008-01 通常経路

- Given 環境が本番である
- And プロファイルが `bootstrap` である
- When `SeedOperator` が `first_party_redirect_uris` を指定して `SeedData` を `apply` する
- Then ファーストパーティークライアントは指定した URI だけをリダイレクト URI として持つ

### Example: EX-SEEDING-008-02 リダイレクト URI が未指定、localhost、または HTTP URI である

- Given 環境が本番である
- And プロファイルが `bootstrap` である
- When `SeedOperator` が `first_party_redirect_uris` を指定して `SeedData` を `apply` する
- But リダイレクト URI が未指定、localhost、または HTTP URI である
- Then `SeedData` は書き込み前に `SeedRejectedError` で拒否する

## Rule: REQ-SEEDING-009 手動変更によるドリフトは上書きせず競合とする

Primary actor: `SeedOperator`

### Example: EX-SEEDING-009-01 通常経路

- Given seed 管理対象の論理キーが手動で変更されている
- When `SeedOperator` が対応するプロファイルを `apply` する
- Then `SeedData` は `SeedConflictError` を返す
- Then 手動変更は維持される

## Rule: REQ-SEEDING-010 部分失敗後に同じリクエストを再実行すると目的の状態へ収束する

Primary actor: `SeedOperator`

### Example: EX-SEEDING-010-01 通常経路

- Given `SeedData` の適用が一部の操作を完了した後に失敗している
- When `SeedOperator` が同じ `SeedRequest` を再度 `apply` する
- Then 完了済みの論理キーは `noop` と判定される
- Then 未完了の論理キーだけが適用され、重複なく目的の状態へ収束する
