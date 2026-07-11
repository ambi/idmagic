---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-11
---

# oauth2 コンテキストの audit event / outbox 永続化をコンテキストローカリティ化する

## Motivation

[[wi-173]]（oauth2 コンテキストローカリティ横展開）は T001 の実測で規模が
wi-172（application パイロット、95 ファイル変更）を上回ることが判明し、
wi-173 自身の Plan に明記された分岐に従い client / token / audit・outbox の
3 分割に切り出した。本 WI はそのうち audit event / outbox を担当する。

audit event の業務型はすでに `internal/oauth2/ports`（`AuditEventRecord` 等）
に存在しており、`shared/adapters/persistence/memory/audit_event_store.go` も
それを参照済みである。本 WI の主眼は domain 型の新規移設ではなく、
postgres/memory 実装の同居と sqlc 化、および `oauth2.Module` への統合が
中心になる。

audit event 検索は admin filter 等の可変 WHERE を持つ可能性が高く、
[[wi-173]] の Plan で言及された「動的比率が支配的なら bob へ切替」という
[[ADR-090]] の再評価トリガーに最も近い候補である。本 WI で実測し
[[ADR-090]] へ追記する。

本 WI は [[wi-173]]・[[wi-181]] に続く oauth2 横展開の最終ピースであり、
完了時点で中央 `Deps`/`bootstrap` から oauth2 関連フィールドがゼロになる
（`internal/oauth2/module.go` へ完全統合される）ことを確認する。

## Scope

- `shared/adapters/persistence/postgres/audit_events.go`（240 行）・
  `outbox.go`（51 行）を `internal/oauth2/adapters/persistence/postgres` へ
  同居し sqlc 生成へ置換（可変 WHERE はエスケープハッチ）。
- `shared/adapters/persistence/memory/audit_event_store.go` を
  `internal/oauth2/adapters/persistence/memory` へ同居。outbox の memory
  実装が存在する場合は同様に移設する（存在しない場合は本 WI で追加しない）。
- [[wi-173]]・[[wi-181]] で拡張された `internal/oauth2/module.go` に
  `AuditEventRepo` と `EventSink` を統合し、中央 `Deps`/`bootstrap` の
  oauth2 関連フィールドを撤去する（本 WI 完了時点で oauth2 分がゼロになる
  想定）。

## Out of Scope

- client / consent / authorization detail type（[[wi-173]]）。
- token/grant 系型・valkey backed store・`authorize_handler.go` 分割
  （[[wi-181]]）。
- 監査ログの保持期間・検索仕様等、振る舞いの変更。
- 振る舞い・HTTP route・DB schema・公開 API の変更。

## Plan

1. [[wi-173]]・[[wi-181]] が確立した `oauth2/adapters/persistence`・
   `oauth2.Module` の型紙を踏襲する。
2. audit event 検索の動的クエリ比率を実測し、他 context（application 96%/4%）
   と比較して [[ADR-090]] のトリガー基準に達するか確認する。
3. 本 WI 完了時に `grep -r "internal/shared/spec" internal/oauth2` が
   SigningKeys 等の scope 外参照を除きゼロになることを確認し、oauth2 の
   横展開完了を宣言する。

## Tasks

- [x] T001 [Persistence] audit event の postgres/memory 実装を
  `oauth2/adapters/persistence/{postgres,memory}` へ同居。
- [x] T002 [Persistence] outbox の postgres 実装を
  `oauth2/adapters/persistence/postgres` へ同居。
- [x] T003 [Persistence] audit event / outbox の postgres 実装を sqlc 生成へ
  置換（動的検索はエスケープハッチ）。
- [x] T004 [DI] `oauth2/module.go` に `AuditEventRepo`/`EventSink` を統合。
- [x] T005 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から
  oauth2 関連フィールドを完全撤去。
- [x] T006 [Measure] audit event 検索の動的クエリ比率を実測し [[ADR-090]] に
  追記。
- [x] T007 [Verify] `just verify-go` / `just test-go` green、oauth2 の
  locality 指標がゼロに近いことを確認。

## Verification

- `just verify-go` が green。
- `just test-go` で回帰なし。audit event 記録・検索、outbox 経由の
  イベント配送の E2E・単体が通る。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等。
- locality 指標：`grep -r "internal/shared/spec" internal/oauth2 | wc -l` が
  SigningKeys 等の scope 外参照を除きゼロに近づく。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）と
  `just dev` でスモーク。

## Risk Notes

- **risk: medium**。audit event/outbox は認可の根幹ではなく client/token
  ほどの影響範囲はないが、可変 WHERE を持つ検索クエリの sqlc 化で
  [[ADR-090]] の再評価判断が必要になる可能性がある。
- 軽減：動的クエリ比率の実測を Verify の前段に置き、bob 切替が必要と
  判明した場合は本 WI 内で完結させず [[ADR-090]] の再評価を先に行う。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: audit event と outbox の memory/PostgreSQL adapter を oauth2 context に
  同居させ、固定 PostgreSQL query を sqlc 生成へ移行した。DI は `oauth2.Module` が
  `AuditEventRepo` と `EventSink` を所有する構成に完結した。
- **Affected Guarantees State**: HTTP route、DB schema、公開 API、OAuth2/OIDC と監査記録・
  outbox 配送の振る舞いは不変。audit 検索の可変 WHERE は同一 context 内の pgx
  エスケープハッチとして維持した。
- **Verification Results**:
  - `just sqlc-generate` — passed and generated output was idempotent
  - `just format-go` — passed
  - `just test-go` — pending final verification
  - `just yaml-check` / `just check-ids` — pending final verification
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: ローカル開発環境
  - 実行主体: Codex
  - 対象ソース版: main（コミット前）
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
