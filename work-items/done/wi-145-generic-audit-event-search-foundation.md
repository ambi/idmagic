---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-10
---

# 監査イベントに汎用検索属性 registry と filter 式を導入し、tenant salt で相関ハッシュを統一する

## Motivation
[[wi-44-authentication-event-store-and-search]] は認証イベントに産業標準の PII 属性
(usernameHash / ipTruncated / ipHash / uaHash …) を **フィールドとして確保した**が、
「ユーザー名 / IP での相関検索」は見送った。ここで `AuditEventQuery` に `username` / `ip` の
個別フィールドを足すと、今後 `actor.id` / `target.id` / `client.id` / `session.id` /
`outcome` などを足すたびに API / port / store / UI が横並びで肥大化する。

Okta System Log、Microsoft Entra sign-in logs、Keycloak events などの主要 IdP は、
イベント種別・期間・actor・client IP・outcome・session / correlation ID を、個別の画面項目
だけでなく構造化された検索軸として扱う。本 WI は [[wi-46-authentication-event-attribute-emit-and-correlation-search]]
から **PII に依存しない汎用検索基盤** を切り出す。検索可能属性の registry / filter 式 / sidecar
ストアと、per-tenant salt による相関ハッシュ (ADR-046) の単一ヘルパを先に作り、
throttle (ADR-029) / bucket の keyHash をそのヘルパに統一する。

username / IP を最初の PII 属性として emit・UI 検索ビルダーに載せる作業は
[[wi-46-authentication-event-attribute-emit-and-correlation-search]] で行う。本 WI は
その土台を用意し、**新規 PII 列や平文入力の hash 化検索は導入しない**（extractor が sidecar に
載せるのは既存の非 PII raw id のみ）。この分割により、throttle / bucket 相関を壊していないかの
レビューを PII ゲート (ADR-046 の 2 名レビュー) と切り離せる。

## Scope
- **scl** (`spec/contexts/oauth2.yaml`):
  - `models` に `AuditEventSearchAttribute` / `AuditEventFilterExpression` (value_object) と
    `AuditEventFilterOperator` / `AuditEventSearchTransform` (enum) を追加する。属性ごとに
    raw 保存可否・transform (none/hash/ip_truncate)・tenant salt 要否・許可 operator・UI 表示可否を
    宣言する。
  - `AuditEventQuery` を `q` (free-text contains) と `filter`
    (`AuditEventFilterExpression[]`) で拡張する。既存 `type/category/user_id/after/before/limit`
    は据え置き (後方互換)。username / IP は将来 `filter` の field として表現し、専用トップレベル
    query param にはしない。
  - `ListAdminAuditEvents` / `ExportAdminAuditEvents` の description に `q`/`filter` と
    registry allowlist を追記し、`glossary` に registry / filter-expression を追加する。
- **go**:
  - 検索可能属性 registry (`AuditSearchRegistry`) を Go 側にも置き、field/operator/transform の
    allowlist の単一の正とする。filter は registry allowlist の field/operator のみ許可し、
    parser / validator (`ParseAuditFilter`) で cardinality (eq=1 / in=N / time_range=2) を検証する。
  - `AuditEventSearchAttribute` 抽出器 (`ExtractSearchAttributes`) を追加し、DomainEvent から
    sidecar 検索属性を生成する。本 WI では非 PII raw id (event.type / outcome / actor.id /
    client.id / session.id / transaction.id / correlation.id / request.id) のみ対象。
  - PostgreSQL では `audit_event_search_attributes` sidecar テーブルに保存し、memory store も
    同じ port 契約で照合する。`AuditEventQuery` / handler / repository を parsed filter で動かす。
  - tenant salt store (生成 / 取得) を追加し、既存の per-tenant secret (KeyStore) 方針に従う。
    salted correlation helper (`SaltedHash` / `TruncateIP`) を単一ヘルパとして置き、
    `hashThrottleKey` (ADR-029) と bucket の emit keyHash をそのヘルパに統一する。

## Out of Scope
- username / IP の emit 値実装 (usernameHash / ipTruncated / ipHash / uaHash の populate) と、
  平文入力からのサーバ側 hash / 丸め化検索、UI 検索ビルダー。すべて
  [[wi-46-authentication-event-attribute-emit-and-correlation-search]]。
- 失敗イベント平文 username の 7 日 null 化 (ADR-045 / ADR-046)。wi-46 で追加する。
- GeoIP / country_code / device fingerprint / risk_score の算出。
- 任意 SQL / JSONPath / OData / SCIM filter の外部公開。registry allowlist の field/operator のみ。
- throttle / bucket store の内部 namespacing (`hashThrottleIdentifier` / `throttleKey`) の変更。
  live counter を無効化するため触らない。統一するのは emit される keyHash のみ。

## Plan
- registry / filter 式は SCL に `concepts` 節が無いため `models` の value_object / enum で表現し、
  Go 側は SCL value_object の twin となる compile-time map (`AuditSearchRegistry`) を単一の正とする。
- filter の wire 表現は繰り返し `filter=field:op:value[,value2]` query param
  (`?filter=actor.username:eq:alice&filter=client.ip:eq:203.0.113.7`)。先頭 2 個の `:` で
  field/op を切り出し、残りを CSV value とする (IPv6 の `:` を許容)。export URL / ログで可読、
  既存 param と後方互換。却下案: base64(JSON) 単一 param (ログ / export URL で不透明)。
- 検索属性は `audit_events` の JSONB を index するのではなく sidecar テーブル
  `audit_event_search_attributes` (event_id, attr_name, attr_value) に持つ。PII governance が
  clean (sidecar は transform 済み値のみ)、registry-shaped で列追加なしに属性を足せる、
  (tenant_id, attr_name, attr_value, occurred_at) の covering index で bounded な eq/in が効く。
- tenant salt store は KeyStore パターン (per-tenant map / generate-on-first-use /
  `tenancy.TenantID(ctx)`) を踏襲。空 tenant `""` でも panic しない。
- ビルド順は内層→外層だが、salt store を extractor より前に置く (hash のソースを先に用意)。

## Tasks
- [x] T001 [SCL] `AuditEventSearchAttribute` / `AuditEventFilterExpression` / operator・transform enum を追加し `AuditEventQuery` に `q`/`filter` を足す。`just yaml-check`。
- [x] T002 [Go/ports] `audit_search_attribute.go` registry、`tenant_salt_store.go` port、`AuditEventQuery`/`AuditEventRecord` 拡張、`shared/spec/correlation.go` (SaltedHash/TruncateIP) + テスト。`just test-go && lint-go`。
- [x] T003 [Go/adapters] tenant salt store (memory + postgres) と `tenant_correlation_salts` テーブル + テスト。`just test-go && lint-go`。
- [x] T004 [Go/usecases] `ParseAuditFilter` / `TransformFilterValues` と `ExtractSearchAttributes` (raw id のみ) + テスト。`just test-go && lint-go`。
- [x] T005 [Go/adapters] handler の filter/q parse、`hashThrottleKey` の salted 化、sidecar `Append`/`List` (PG + memory)、schema + テスト (後方互換含む)。`just test-go && lint-go && build-go`。
- [x] T006 [Go/bootstrap] `SaltStore` 注入、emit closure で extractor 配線・salt 解決 + E2E テスト。`just test-go && build-go`。
- [ ] T007 [Verify] 全ゲート green、completion 追記、`done/` へ移動、commit。

## Verification
- `just yaml-check`
- `just test-go`
  - reason: filter parser / validator、registry allowlist、raw id 抽出、memory / PostgreSQL store の
    同一検索結果、`SaltedHash` の tenant 分離 (同一値でも tenant が違えば hash が異なる)、
    salted keyHash への統一で throttle / bucket 集約が壊れないこと、**後方互換** (旧クエリ文字列が
    同一結果を返す)。
- `just lint-go`
- `just build-go`

## Risk Notes
汎用 filter は便利だが、任意クエリ実行や PII フィールドへの不用意な検索を許すと監査基盤自体が
情報漏洩面になる。field/operator は registry allowlist に限定する。相関ハッシュ統一では
**emit される keyHash のみ** を salted 化し、store 内部の namespacing は変えない (live throttle
counter を無効化しないため)。salt store は全 bootstrap 経路で non-nil、空 tenant `""` で panic
しないことをテストで担保する。tenant salt の取り違えは cross-tenant 相関漏洩につながるため、
salt 取得は必ず対象 tenant に紐づける。本 WI は新規 PII 列を持たないため ADR-046 の 2 名ゲートは
[[wi-46-authentication-event-attribute-emit-and-correlation-search]] に集約する。

## Completion

- **Completed At**: 2026-07-10
- **Summary**:
  監査イベント検索の汎用基盤を追加した。SCL に検索属性 registry / filter 式 / `q` を追加し、Go 側に registry twin、filter parser / validator、非 PII raw id 抽出器、tenant salt store、相関 hash / IP 丸めヘルパを実装した。HTTP handler は `filter=field:op:value[,value2]` と `q` を受け、memory / PostgreSQL repository は sidecar 検索属性で照合する。
  - `audit_event_search_attributes` sidecar と `tenant_correlation_salts` を PostgreSQL schema に追加。
  - bootstrap の audit emit 経路で `ExtractSearchAttributes` を接続し、memory / postgres の両 persistence mode へ `TenantSaltStore` を注入。
  - throttle / authentication event bucket の外部公開 `keyHash` は `SaltedHash` helper 経由へ統一し、内部 throttle/bucket namespacing は変更しないまま維持。
  - username / IP の emit 値 populate、平文入力からの PII 検索、UI 検索ビルダーは予定どおり wi-46 に残置。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just test-go` - passed
  - `just lint-go` - passed
  - `just build-go` - passed
  - `just verify` - passed
- **Affected Guarantees State**:
  SCL に追加した `AuditEventQuery.q` / `AuditEventQuery.filter`、`AuditEventSearchAttribute` registry allowlist、`AuditEventFilterExpression` の operator/cardinality 制約を実装済み。任意 SQL / JSONPath を受けないこと、PII raw 値を sidecar に保存しないこと、tenant salt による cross-tenant 相関分離をテストで確認した。
