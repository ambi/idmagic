---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-05
---

# Postgres 列型選定ポリシーの明文化と既存 schema の棚卸し

## Motivation
`deploy/schema/postgres.sql` は、認証・認可・管理 UI・アプリケーションカタログ・
フェデレーションなど複数の bounded context の永続化を横断している。一方で、列型の
選定基準が明文化されていないため、`TEXT` / `JSONB` / `TIMESTAMPTZ` / `UUID` /
状態値表現の使い分けが実装時の局所判断に寄っている。

PostgreSQL では `TEXT` と制約なし `varchar` に実質的な性能差はなく、`TEXT` の多用
そのものは問題ではない。`TIMESTAMPTZ` もデフォルトでマイクロ秒精度を持つため、
秒精度に限られる型ではない。また、IdP の識別子には OIDC client_id、SAML entity_id、
SCIM id、tenant/user/group の domain id のように、UUID に限定しない方が自然な値もある。
それでも、現状の schema には次のような改善余地がある。

- 人間が入力する名前・URL・メール・説明など、仕様上または運用上の上限がある文字列に
  DB 側の防衛線を置くべきかが未整理。ただし、具体的な最大文字数は業務ルールとして
  別途決める必要がある。
- `roles`、`scopes`、`redirect_uris`、`grant_types`、`bindings`、policy `rules`、
  federation claim policy などの `JSONB` が、半構造化データとして妥当なものと、
  検索・参照整合性・制約を持つべきものに分けられていない。
- `users.lifecycle` のように、状態値を `JSONB` 内に置くことで部分 index とアプリ側解釈に
  依存している箇所がある。
- `status` / `state` / `kind` / `visibility` / `subject_type` などの有限集合値が、
  `TEXT CHECK`、`JSONB` 内の値、制約なし `TEXT` に分散している。
- UUID 型を使う列と domain string id を使う列の境界が、ADR-082 / ADR-083 の個別判断に
  依存しており、新規テーブル追加時の判断基準として再利用しづらい。

この WI では、列型を一括で置き換えるのではなく、Postgres 永続化の型ポリシーを先に決め、
既存 schema を棚卸しして、必要な schema / adapter / API 変更だけを段階的に適用する。

## Scope
- **decision / documentation**:
  - Postgres 列型選定ポリシーを ADR または永続化設計ドキュメントとして明文化する。
  - `TEXT` と `varchar(n)` / `CHECK (char_length(...))` の使い分け基準を決める。方針案:
    制約なし `varchar` は使わず、無制限文字列は `TEXT`、仕様・UI・運用上の上限が
    すでに決まっている値は `TEXT` + `CHECK` または `varchar(n)` のどちらかに統一する。
    具体的な最大文字数の決定は `wi-128-string-length-limits-policy` に分離する。
  - `JSONB` の許容基準を決める。方針案: 外部仕様由来の可変メタデータ、claim/policy の
    半構造化設定、監査/outbox payload は許容し、頻繁に join / filter / FK / uniqueness /
    lifecycle 制約を要する値は正規化または専用列化を検討する。
  - `TIMESTAMPTZ` の精度・丸め・JSON/API 表現の基準を明文化する。PostgreSQL の保存精度が
    マイクロ秒であることを前提に、Go adapter、HTTP JSON、テスト fixture、UI 表示で
    意図せず秒精度へ丸めていないかを確認対象にする。
  - ID 列で `UUID` を使う条件と、domain string id を維持する条件を明文化する。方針案:
    内部生成で外部プロトコル語彙に縛られない surrogate id は `UUID` 候補、OIDC/SAML/SCIM
    や tenant/user/group/client の domain id は既存 ADR と互換性を確認して個別判断する。
  - 状態値・種別値を PostgreSQL enum、`TEXT CHECK`、lookup table、domain type、アプリ側
    enum のどれで表現するかの基準を決める。PostgreSQL enum は削除・rename・並行 deploy の
    migration friction があるため、採用する場合は変更頻度の低い値に限定する。
- **schema**:
  - `deploy/schema/postgres.sql` の全列を棚卸しし、型カテゴリごとに「現状維持」「制約追加」
    「専用列化」「正規化」「型変更」「ADR で理由を残す」に分類する。
  - 特に `users.lifecycle`、`roles`、`scopes`、`applications.bindings`、
    `application_sign_in_policies.rules`、`tenant_default_sign_in_policies.rules`、
    federation `claim_policy`、audit/outbox `payload`、各種 `status` / `state` /
    `kind` / `visibility` / `subject_type` を重点確認する。
  - 型変更が必要な場合は、宣言的 schema と migration / seed / test fixture の扱いを
    既存の Postgres schema 管理方針に合わせて更新する。
- **implementation**:
  - schema 変更に追随して、Postgres adapter、memory adapter、shared spec 型、HTTP API、
    UI 型が必要最小限で整合するように修正する。
  - 状態値を enum 相当に強める場合は、DB 制約と Go 側 validation / serialization が
    同じ値集合を扱うことをテストする。
- **spec**:
  - 永続化の型選定が SCL の保証・非機能要件・公開 contract に影響する場合のみ、
    `spec/scl.yaml` を SCL-first で最小限更新し、derived artifacts を再生成する。

## Out of Scope
- 全 `TEXT` 列を機械的に `varchar` へ置き換えること。
- 名前・説明・URL・メール・外部識別子など、各文字列値の具体的な最大文字数をこの WI で
  決めること。最大文字数の業務ルール化と SQL 反映判断は
  `wi-128-string-length-limits-policy` で扱う。
- 全 domain id を UUID に統一すること。
- 全 `TEXT CHECK` を PostgreSQL enum に置き換えること。
- 外部プロトコル仕様や公開 API の識別子語彙を、DB 型都合だけで変更すること。
- `JSONB` を完全に排除すること。監査イベント、outbox、外部仕様由来の claim / metadata /
  policy 表現では、正当な利用を残す。

## Verification
- `just yaml-check-work-items`
- `just check-ids`
- `just yaml-check`（SCL を変更した場合）
- `just scl-render`（SCL を変更した場合）
- `just verify-go`
- `just verify`
- 手動確認: `deploy/schema/postgres.sql` の各 `TEXT` / `JSONB` / `TIMESTAMPTZ` /
  `UUID` / 状態値列について、ポリシーに照らした判断結果が ADR または実装差分に残っている。
- 手動確認: 時刻値を保存・取得する主要フローで、PostgreSQL、Go、HTTP JSON、UI 表示の
  どこで精度を落とすか、または落とさないかが明確になっている。

## Risk Notes
列型変更は migration、既存データ、外部 API、テスト fixture、UI 型へ波及しやすい。
特に ID 型と状態値表現は一度公開 contract に出ると戻しづらい。実装時は、まず ADR /
ドキュメントで基準を固定し、既存 schema の棚卸し結果を小さな変更単位に分割する。
UUID 化・enum 化・JSONB 正規化は、それぞれ互換性と migration friction を評価してから
個別に進める。

## Completion
- **Completed at**: 2026-07-05
- **Summary**: Postgres 列型選定ポリシーを ADR として明文化したうえで、レビュー判断により
  「idmagic が内部生成する id 列を UUID 型に閉じる」方針転換を取り込み、schema・adapter・
  seed・UI・テストを実際に整列させた。tenants.id の UUID 化と文字列長制約は follow-up に分離。
  - `decisions/ADR-084-postgres-column-type-policy.md` を追加。文字列（制約なし varchar
    不使用）・JSONB 許容基準・TIMESTAMPTZ 精度・UUID と domain string id の境界・有限集合値
    表現（既定 TEXT+CHECK、PG enum は原則不採用）の 5 基準と、全列の型カテゴリ別棚卸し表を掲載。
    §4 で「内部生成 id は UUID 型、外部が値を決める id（`entity_id`/`wtrealm`/`scim_id`/`kid`）は
    TEXT」を確定。ADR-071 / ADR-082 / ADR-083 と整合。
  - **内部生成 id の UUID 型化**: `deploy/schema/postgres.sql` で `users.id`・`clients.client_id`・
    `groups.id`・`agents.id`・`audit_events.id`・`scim_tokens.id` と、それらを参照する FK 列
    （`user_id` 系・`owner_user_id`・`group_id`・`agent_id`・`client_id` 系・`subject_id`・
    `audit_events.user_id`）を `TEXT`→`UUID` に変更。外部が値を決める id と `tenants.id`（slug）は
    TEXT を維持。
  - **Go は string を維持**: `internal/shared/adapters/persistence/postgres/base.go` の
    `AfterConnect` で uuid OID に text codec を登録（`RegisterUUIDAsText`）し、UUID 列を Go の
    `string` で read/write。test harness も同 codec を登録。id 型は全層で `string` のまま。
  - `application_assignments.ListBySubjects` の UNNEST 比較を、param は `text[]` のまま列側を
    `subject_id::text` にキャストして uuid=text 不一致を解消。
  - **seed / UI の固定 UUID 化**: `internal/bootstrap/seed.go` の非 UUID id（`user_alice` /
    `user_root` / `demo-client` / `idmagic-admin-console` / `idmagic-account-portal` /
    `group_engineering` / `group_support`）を固定 UUID（`00000000-0000-4000-8000-...`）へ定数化。
    application binding と UI の OIDC 設定（`ui/src/api/oidc.ts` / `authFlow.ts`）を同値に追随。
  - Postgres roundtrip テストの id fixture / inline リテラルを UUID（`newUUID(t)`）に更新。
  - 時刻精度マップを ADR §3 に明記（PostgreSQL マイクロ秒保存、pgx `time.Time` 精度保持、内部
    JSON は RFC3339Nano、秒精度化は SCIM/SAML/WS-Fed/audit クエリの外部境界のみ、UI はミリ秒）。
- **follow-up 分離**: `tenants.id` の不変 UUID + mutable `realm` 分割は
  `wi-140-tenant-uuid-key-and-realm-identifier` を新規作成。文字列長は
  `wi-128-string-length-limits-policy`、`users.lifecycle` 正規化・CHECK なし有限集合列の CHECK
  付与は ADR に follow-up 記録。
- **SCL impact**: なし（永続化層の内部方針。id の JSON 表現は従来どおり文字列で公開 contract
  不変。`spec/scl.yaml` 変更・derived artifacts 再生成は不要）。
- **Verification results**:
  - `just build-go` パス
  - `just test-go` パス（embedded-postgres が uuid 列型を強制、全 roundtrip green）
  - `just test-go-race` パス（DATA RACE なし）
  - `just lint-go` パス（0 issues）
  - `just verify-ui` パス（format / lint / typecheck / unit / build）
  - `just yaml-check-work-items` / `just check-ids` パス
  - 手動確認: 各 TEXT / JSONB / TIMESTAMPTZ / UUID / 状態値列の判断を ADR-084 棚卸し表に記録。
  - 手動確認: 時刻値の精度落とし箇所を ADR-084 §3 に明記。
