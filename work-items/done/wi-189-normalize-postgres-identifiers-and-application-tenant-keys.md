---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-11
depends_on: [wi-184-transactional-event-log-foundation]
---

# PostgreSQL の識別子と Application 子テーブルの tenant key を正規化する

## Motivation

現在の PostgreSQL schema は、グローバル識別子として扱いたい UUID に tenant を含む複合主キーを
与える一方、関連する child table に tenant_id を重複して持たせている。Application の UUID は
DB 制約としてはグローバル一意でなく、child の tenant_id は親 Application から導出できる。
この重複は query ごとの tenant 条件漏れと、親子の所属整合性を二重に維持する負担を生む。

同様に、RefreshToken は global な User / OAuth2Client を参照するにもかかわらず tenant_id を
保持するが、実装済みの repository に tenant 単位の検索・失効利用ケースはない。Password history
だけが BIGSERIAL surrogate key を使い、他の内部生成 ID の UUID 方針から外れている。Event log の
event_id も Kafka header の名前と永続レコードの主キー名を混同している。

親のグローバル UUID による参照、DB 制約での参照整合性、外部契約の語彙を分離し、永続化モデルを
一貫した識別子方針へ戻す。

## Scope

- **scl**:
  - `spec/contexts/oauth2.yaml` の RefreshTokenRecord から、検索・保持・partitioning・監査隔離の
    実要件を持たない `tenant_id` の永続化依存を除き、User / OAuth2Client の global ID を介した
    tenant-local invariant を定義する。
  - `spec/contexts/application.yaml` の Application、ApplicationIcon、ApplicationAssignment、
    ApplicationSignInPolicy の identity を Application の global UUID と親参照に整理する。
    テナント境界は Application の tenant_id と、command / query における actor tenant の照合で
    fail-closed に保つことを明記する。
  - `spec/contexts/system.yaml` の EventLogRecord identity を `id` に変更する。Kafka header と
    EventDeliveryRecord が参照する値の外部語彙 `event_id` は維持し、`event_id = event_logs.id` の
    写像を明記する。
  - `spec/scl.yaml` の include と、上記変更から導出される HTML / JSON Schema / OpenAPI を同期する。
- **persistence / go**:
  - `refresh_tokens` から `tenant_id`、その FK、tenant prefix index を削除し、SQL query、sqlc 生成物、
    Postgres / memory repository、domain record、bootstrap とテストを更新する。ユーザー削除と token
    rotation / family revoke が tenant_id なしで従来どおり動くことを確認する。
  - `event_logs` の UUID primary key を `id` にし、`event_deliveries.event_id` は
    `event_logs(id)` を参照する共有 primary key として残す。relay が Kafka header に `event_id` を
    出す契約は維持する。
  - `applications.id` を UUID primary key とし、`tenant_id` は Application 自身の所属列として残す。
    `application_icons`、`application_sign_in_policies`、`application_assignments` は
    `application_id -> applications(id)` の FK だけで親を参照し、冗長な tenant_id、複合 primary key /
    FK、query 引数を削除する。
  - Application の一覧・取得・更新・icon 取得・assignment・sign-in policy の各 repository / use case
    で、request actor の tenant と親 Application の tenant を照合する。ApplicationAssignment の
    User / Group subject が Application と同じ tenant に属することも検証し、tenant_id 列の削除で
    fail-open にしない。
  - `password_history.id` を UUID primary key に統一し、履歴取得の安定順序を
    `(user_id, created_at DESC, id DESC)` で維持する。`(user_id, created_at)` を unique にして
    正当な同時刻レコードを拒否する設計は採らない。
  - declarative schema の適用方式に沿って、既存 PostgreSQL データの migration / rollout / rollback
    方針を決定し、必要な migration 又は互換読み取り期間を実装する。

## Out of Scope

- 公開 HTTP API、URL parameter、JSON response の `application_id` を一律に `id` へ改名すること。
- Kafka header `event_id`、イベント本文、消費側冪等性キーの名称変更。
- tenant model、realm URL、tenant 一覧・削除 API の変更。
- Application assignment の polymorphic User / Group 参照を単一テーブルへ統合すること。
- 未要件の tenant 単位 RefreshToken 管理 API を追加すること。

## Plan

1. SCL で「DB 内部主キー」「親を参照する FK」「外部プロトコル・イベントの field name」を別の概念として
   固定する。Event log は `id`、Kafka header は `event_id` とし、値の写像を契約として残す。
2. Application ID を schema で global UUID primary key にしたうえで、child table から tenant_id を除く。
   tenant-scoped API は必ず Application 親との join / lookup で境界を確認する。子テーブルの冗長な
   tenant_id による境界保証には依存しない。
3. RefreshToken の tenant_id は実在する tenant-scoped query が無いことを repository test で確認した
   うえで除去する。将来この列を再導入する場合は、利用ケース、索引、User / Client との所属一致を
   DB 制約で保証することを前提にする。
4. Password history は UUID に統一する。時刻は意味のある並び順の第一キー、UUID は同一時刻の安定した
   tie-breaker とし、連番生成を識別子方針の例外にしない。

## Tasks

- [x] T001 [SCL] OAuth2、Application、System context の identity・tenant isolation・event header
  mapping を更新し、導出成果物を再生成する。
- [x] T002 [Schema] schema と migration / rollout 方針を実装する。Application の global primary key、
  child FK、RefreshToken / PasswordHistory / EventLog の列・索引・制約を更新する。
- [x] T003 [Persistence] SQL query、sqlc 生成物、Postgres / memory adapter、domain record を新しい
  identity と FK 契約へ更新する。
- [x] T004 [App] Application の tenant guard と assignment subject の tenant consistency を use case /
  repository 境界で実装し、Kafka header `event_id` の互換を確認する。
- [x] T005 [Verify] cross-tenant insert / read / mutation、token rotation / revocation、password history の
  同時刻順序、event delivery / Kafka header の回帰テストを追加して全検証を通す。

## Verification

- `just yaml-check-scl`
- `just scl-render`
- `just yaml-check-work-items`
- `just check-ids`
- `just sqlc-generate`
- `just test-go`
- `just verify-go`
- Postgres repository tests: Application UUID の global uniqueness、child の親 FK、別 tenant の
  Application / subject の read・write 拒否、RefreshToken の User / Client tenant 一致を確認する。
- relay integration test: `event_logs.id` を source に、Kafka header の `event_id` と
  `event_deliveries.event_id` が同じ UUID であることを確認する。

## Risk Notes

high。主キー・FK・query 引数の変更は sqlc、repository、memory adapter、seed、既存 PostgreSQL data に
横断的に影響する。特に Application child から tenant_id を外すと、親を通さない query が tenant
境界を確認できなくなる。各 read / write path を Application の tenant で照合し、cross-tenant の
negative test を先に追加して fail-closed を固定する。

`wi-184` は EventLog と EventDelivery の所有契約を定義しているため、本 WI はその完了後に実施する。
Kafka の `event_id` は外部の冪等性契約であり、DB primary key の列名を `id` にしても値・header 名を
変えない。既存データが存在する環境では、rename / FK 再作成の deploy 順序と rollback 可否を先に
確定する。

## Completion

- **Completed At**: 2026-07-11
- **Summary**:
  Application の識別子を global UUID primary key に正規化し、icon、assignment、sign-in policy は
  親 Application の foreign key のみで参照するようにした。リポジトリの tenant-scoped read / delete は
  親 Application への join / exists guard を通し、冗長な child tenant key に依存しない fail-closed
  境界へ変更した。RefreshToken の永続 tenant_id とその索引を除去し、EventLog の内部 primary key を
  `id` に変更しながら、Kafka / EventDelivery の `event_id` 値は同一 UUID のまま維持した。Password
  history は UUID primary key を DB 生成し、既存の `(user_id, created_at DESC, id DESC)` 順序を維持した。
- **Affected Guarantees State**:
  Application child は親の global UUID foreign key により参照整合性を維持し、tenant-scoped 操作は
  親 tenant と actor tenant の一致を要求する。RefreshToken の tenant 所属は global User / OAuth2Client
  から導出され、event delivery と Kafka の冪等性キーは `event_logs.id` と同じ UUID 値である。
- **Verification Results**:
  - `just yaml-check-scl` — passed
  - `just scl-render` — passed
  - `just sqlc-generate` — passed
  - `just test-go` — passed
  - `just verify` — passed
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: ローカル開発環境
  - 実行主体: Codex (GPT-5)
  - 対象ソース版: main（コミット前）
  - 保存先: 外部成果物なし。`just verify` の成功結果と本 completion に要約を記録。
