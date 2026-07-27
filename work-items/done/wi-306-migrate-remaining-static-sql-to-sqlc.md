---
status: completed
authors: [Antigravity]
risk: medium
created_at: 2026-07-28
depends_on: []
---

# Migrate remaining static SQL to sqlc

## Motivation
`ADR-090` にて、SQL の実行は原則として `sqlc` を利用し、動的クエリ（`WHERE` 句が可変になるなど）のみ `pgx` を用いたエスケープハッチ（生SQL）を許容するという方針が決定されました。しかし、Federation, Provisioning, SigningKeys, IdGovernance, IdManagement, Authentication の各コンテキストの一部において、完全に静的な SQL であるにもかかわらず `sqlc` を利用せず直接生SQLを発行している箇所（技術的負債）が残存しています。本作業ではこれらの残存する静的 SQL をすべて `sqlc` へ移行し、ドキュメントに規約を明記することでアーキテクチャへの準拠を徹底します。

## Scope
- `ARCHITECTURE.md` (コーディング規約・永続化方針の追記)
- `sqlc.yaml`
- `backend/signingkeys/db_postgres`
- `backend/provisioning/db_postgres`
- `backend/authentication/federation/db_postgres`
- `backend/idgovernance/db_postgres`
- `backend/idmanagement/group/db_postgres`
- `backend/authentication/password/db_postgres`

## Out of Scope
- 動的クエリのエスケープハッチとして残す必要のある箇所の移行（`backend/audit/db_postgres/audit_events.go`, `backend/application/db_postgres/applications.go` の `ListBySubjects`, `backend/tenancy/db_postgres/quota_repository.go` など）。

## Design
- `sqlc.yaml` に漏れていたコンテキスト（`signingkeys`, `provisioning`, `authentication/federation`）を生成対象として追加する。
- Go ソース内にベタ書きされている静的 SQL 文字列を `.sql` ファイル（`queries/*.sql` ではなくコンテキストの `db_postgres/*.sql`、フラット構造の採用 ADR-267 に準拠）に切り出し、`just sqlc-generate` を実行する。
- 生成された `Queries` 構造体を用いて、既存の `r.Pool.Query`, `r.Pool.Exec` 等を置き換える。
- `ARCHITECTURE.md` に ADR-090 に基づく「静的クエリの sqlc 必須化と、生SQLは動的クエリの例外としてのみ許容する」旨を明記する。

## Plan
1. `sqlc.yaml` を更新。
2. 各対象コンテキストに対して `.sql` ファイルを作成し、Go ファイルから SQL を抽出。
3. `just sqlc-generate` 実行。
4. Go 側の実装をリファクタリング。
5. `ARCHITECTURE.md` を更新。
6. テスト実行（`just verify-go`）。

## Tasks
- [x] T001 [Config] `sqlc.yaml` を更新する。
- [x] T002 [App] `signingkeys` コンテキストを移行する。
- [x] T003 [App] `provisioning` コンテキストを移行する。
- [x] T004 [App] `authentication/federation` コンテキストを移行する。
- [x] T005 [App] `idgovernance` コンテキストの未移行分を移行する。
- [x] T006 [App] `idmanagement/group` コンテキストの未移行分を移行する。
- [x] T007 [App] `authentication/password` コンテキストの未移行分を移行する。
- [x] T008 [Doc] `ARCHITECTURE.md` を更新する。
- [x] T009 [Verify] 全体のビルドとテストを検証する。

## Verification
- `just verify-go` がエラーなくパスすること。

## Risk Notes
- 移行対象が多岐にわたるため、既存のクエリと少しでも挙動が変わるとエンバグするリスクがある（Medium）。SQL のコピーミスや、`RETURNING` パラメータの対応関係に特に注意する。

## Completion
- **Implementation Done**: すべての対象コンテキストにおける静的 SQL の `sqlc` への移行を完了し、各所の不要な生 SQL と手書き Scan メソッドを削除しました。
- **Documentation Done**: `ARCHITECTURE.md` に ADR-090 に基づく `sqlc` デフォルトの方針を明記しました。
- **Test Passed**: `just verify-go` がすべてエラーゼロでパスしました。
- **Out of Scope**: 動的クエリとして残す予定であった `audit` 等のクエリには触れていません。
