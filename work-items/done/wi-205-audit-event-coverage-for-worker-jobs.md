---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-12
depends_on: []
---

# idmagic-worker から実行される業務操作を監査イベントに漏れなく記録する

## Motivation
CSV ユーザーインポート (wi-202/wi-96) の apply ジョブでユーザー作成は成功するが、
`/admin/audit_events` には何も記録されない欠落が見つかった。原因は
`backend/cmd/idmagic-worker/worker.go:51` が組み立てる `AdminUserDeps` に
`Emit` を一切設定していないこと。`adminEmit`
(`backend/identitymanagement/usecases/admin_users.go:416-421`) は
`sink == nil` を静かに no-op として扱うため、`CreateUser`
(`admin_users.go:125`) が `user_import.go:165` からループ呼び出しされるたびに
`UserCreated` イベントがエラーなく握りつぶされる。ユーザー自体は正しく作成される
ため、機能的には成功して見えるのに監査証跡だけが欠落する、気づきにくい種類の
バグである。

`spec/contexts/identity-management.yaml:2004-2015`
の既存シナリオ「管理者は CSV を検証して有効な行だけをインポートできる」は
main_success で「有効行だけが `UserCreated` として作成され」と明記しており、
実装は既存の仕様記述にも反している。

`idmagic-api` プロセス (`backend/cmd/idmagic/server.go:161-180`) は `emit`
クロージャで `EventSink.Emit` と `bootstrap.NewAuditEventRecord` →
`AuditEventRepo.Append` の両方を行い、HTTP 経由の admin 操作
(`admin_user_handler.go:269` など、すべて `Emit: d.legacyEmit()` を配線) は
正しく監査ログに残る。一方 `idmagic-worker` プロセスにはこの `emit` 相当の
配線が存在せず、worker から呼ばれる業務ロジックの `DomainEvent` はすべて宛先を
持たない。現状 worker が呼ぶ業務ロジックはインポートの `CreateUser` /
`SetUserRequiredAction` のみだが、この配線漏れは worker.go 個別のミスではなく
「worker プロセスに audit へ橋渡しする仕組み自体が存在しない」という構造的な
欠落であり、今後 worker 側に admin 系ジョブハンドラを追加するたびに同じ欠落が
再発し得る。

## 現状判明している欠落 (What's missing)
- `backend/cmd/idmagic-worker/worker.go` に `EventSink.Emit` +
  `AuditEventRepo.Append` を行う `emit` 相当の関数が存在しない
  (`cmd/idmagic/server.go:161-180` にしかない)。
- `worker.go:51` の `AdminUserDeps{...}` リテラルに `Emit` フィールドが列挙され
  ておらず、ゼロ値 (`nil`) のまま `UserImportHandler` に渡される。
- CSV インポート apply で作成された全ユーザーの `UserCreated` イベントが
  サイレントにロストする (エラーなし、ログなし)。
- worker.go 内の `usecases.RunnerDeps.Emit` (68 行目) はジョブライフサイクル
  イベント (`JobEnqueued`/`JobStarted`/`JobSucceeded`) をログ出力するだけで、
  `AuditEventRepo` には書き込まない別物であり、これと混同されないよう注意が要る。
- 監査網羅性を守るための自動テスト・構造的ガードレールが存在しない
  (`AdminUserDeps.Emit` 等、`Emit` フィールドを持つ deps 構造体が実際に監査
  シンクへ配線されているかを検証する仕組みがない)。
- import 専用の監査イベント (例: `UserImportApplied`) は定義されておらず、
  ジョブ全体の実行結果 (成功件数・失敗件数) は監査証跡としては残らない
  (行ごとの `UserCreated` のみが対象)。本 WI ではこれの新設は Out of Scope
  とし、既存 `UserCreated` の配線修復のみを扱う。

## Scope
- `backend/cmd/internal/bootstrap` に、`server.go:161-180` の `emit` クロージャ
  と同等の「`EventSink.Emit` + `AuditEventRepo.Append`」処理を共有ヘルパーとして
  切り出し、`cmd/idmagic` と `cmd/idmagic-worker` の両方から使う。
- `backend/cmd/idmagic-worker/worker.go` の `AdminUserDeps.Emit` に上記ヘルパーを
  配線する。
- `spec/contexts/audit.yaml` の `invariants` に、「業務操作が発行する
  DomainEvent は実行プロセスに関わらず AuditEventRepo に記録される」という
  不変条件を追加する。
- CSV インポート apply が実 Postgres 上で `UserCreated` を `AuditEventRepo` に
  記録することを検証する回帰テストを追加する
  (`backend/identitymanagement/usecases` または `backend/cmd/idmagic-worker` 配下)。

## Out of Scope
- import 専用の集約イベント (`UserImportApplied` 等) の新設。
- worker.go 以外の場所 (HTTP ハンドラ側) の `Emit` 配線の再監査。調査時点では
  HTTP 側は全ハンドラが `legacyEmit()` を設定していることを確認済み。
- 監査イベントの保持期間・エクスポート機能の変更。

## Plan
- `bootstrap.NewAuditEventRecord` を使う「event を EventSink と AuditEventRepo
  の両方に書く」処理を `bootstrap.Deps` のメソッドまたは関数として一箇所に
  実装し、`server.go` 側もこれを呼ぶようにリファクタして重複配線を無くす
  (プロセスごとに書き方が分岐して今回のような漏れが起きた反省を反映する)。
- worker.go 側は `bootstrap.Assemble` の戻り値からこの共有 emit 関数を組み立て、
  `AdminUserDeps.Emit` に渡す。
- 回帰テストは実 Postgres (`pgtest.Require`) を使い、`UserImportHandler(apply=true)`
  相当のフローを通した後 `AuditEventRepo` から `UserCreated` が読み出せることを
  アサートする形にする (nil Emit の握りつぶしはエラーを返さないため、モックの
  `Emit` を差すだけのテストでは今回のような「実プロセスの配線漏れ」は検出できず、
  実際に worker.go の組み立てコードに近い経路を通す必要がある)。

## Tasks
- [x] T001 [SCL] `spec/contexts/audit.yaml` に監査記録の不変条件を追加し
      `just scl-render` で再生成する。
- [x] T002 [App] `cmd/internal/bootstrap` に共有 emit ヘルパーを実装し、
      `cmd/idmagic/server.go` をこれを使うようにリファクタする。
- [x] T003 [App] `cmd/idmagic-worker/worker.go` の `AdminUserDeps.Emit` に
      共有 emit ヘルパーを配線する。
- [x] T004 [Test] CSV インポート apply が実 Postgres 上で `UserCreated` を
      `AuditEventRepo` に記録することを確認する回帰テストを追加する。
- [x] T005 [Verify] `just verify` を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just test-go`
- `just verify`
- 手動確認: `just dev` で CSV インポートを実行し、`/admin/audit_events` に
  `UserCreated` が表示されることを目視確認する。

## Risk Notes
- `emit` ヘルパーの共通化は `cmd/idmagic/server.go` の起動シーケンスに触れるため、
  既存の HTTP 経由の監査記録を壊さないよう、リファクタ前後で `server.go` 側の
  挙動が変わらないことをテストで確認する。
- 監査ログはコンプライアンス (GDPR Art. 30、`AuditLogRetention` objective)
  に関わるため、記録漏れは軽微に見えても実際には監査要件違反になり得る。
  リスクレベルは medium とするが、放置期間が長引くほど実質的な影響は大きくなる。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  `bootstrap.Dependencies.NewEmitFunc` として `EventSink.Emit` + `AuditEventRepo.Append`
  の共有ヘルパーを実装し、`cmd/idmagic/server.go` の重複した `emit` クロージャをこれに
  置き換えた。`cmd/idmagic-worker/worker.go` は `newAdminUserDeps` ヘルパーを新設し、
  `AdminUserDeps.Emit` を同じ共有ヘルパー経由で配線した (`legacyEmit` と同じ
  fire-and-forget adaptation)。`spec/contexts/audit.yaml` に
  `DomainEventsAreAuditedRegardlessOfProcess` 不変条件を追加した。
- **Affected Guarantees State**:
  `idmagic-worker` が実行する業務操作 (CSV インポート apply の `UserCreated` 等) が
  発行する DomainEvent は、`idmagic` (HTTP) プロセスと同じ経路で `AuditEventRepo` に
  記録されるようになった。ジョブライフサイクルイベント (`JobEnqueued` 等) は従来通り
  ログ出力のみで、監査対象外のまま区別されている。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just scl-render` — passed
  - `just verify` — passed (Go race tests + lint、UI format/lint/typecheck/unit/build を含む)
  - `go test ./backend/cmd/idmagic-worker/... -run TestUserImportApplyRecordsUserCreatedAuditEvent -v`
    — passed (embedded PostgreSQL 上で `newAdminUserDeps` → `UserImportHandler(apply=true)`
    → `AuditEventRepo.List` の経路を通し `UserCreated` が 1 件記録されることを確認)
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: macOS、embedded PostgreSQL (pgtest)
