---
depends_on: []
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-10
---

# 監査イベントを OAuth2 から独立した audit bounded context へ切り出す

## Motivation

`ListAdminAuditEvents` / `ExportAdminAuditEvents` / `GetAdminAuditEvent` は現在
`spec/contexts/oauth2.yaml` と `internal/oauth2/...` に属しているが、扱う対象は OAuth2 に閉じない。
監査イベントは authentication、identity-management、oauth2、tenancy、signing-keys、application、
saml / wsfederation など複数 bounded context から発火される横断的なセキュリティ調査用 read model
である。

このまま OAuth2 context に置き続けると、OAuth2 が監査基盤の所有者に見え、今後の検索属性追加、
保持期間、SIEM export、PII governance、admin UI の契約が OAuth2 の責務として肥大化する。
一方で identity-management に寄せると、ユーザー / グループ管理の一部であるかのように見え、
token / client / tenant / key / protocol federation の監査を説明しにくい。

監査は OAuth2 でも identity-management でもなく、横断的な audit bounded context として独立させる
方が自然である。本 WI は、既存の OAuth2 配置を互換的に audit context へ移し、監査 API / port /
usecase / persistence / UI navigation の所有境界を明確にする。

## Scope

- **scl**:
  - 新しい `spec/contexts/audit.yaml` を追加し、監査ログ read model と管理 API の所有 context を
    audit に移す。
  - `AuditEventQuery` / `AuditEventRecord` / `AdminAuditEventResponse` /
    `AdminAuditEventListResponse` / `AuditEventSearchAttribute` /
    `AuditEventFilterExpression` / operator / transform enum を audit context の models とする。
  - `ListAdminAuditEvents` / `ExportAdminAuditEvents` / `GetAdminAuditEvent` と
    `AdminAuditEventsRead` permission を audit context の interfaces / permissions とする。
  - OAuth2 context から上記の監査 API 所有定義を削除し、必要な参照だけを残す。
  - `spec/scl.yaml` の context 一覧、navigation / user_experience 上の監査ページ参照を audit
    context に同期する。
- **architecture**:
  - `ARCHITECTURE.md` に audit bounded context と Go package ownership を追加する。
  - shared persistence adapter が audit port を実装する構図を明記する。
- **go**:
  - `internal/audit/{ports,usecases,adapters/http}` を作り、現在 `internal/oauth2` にある監査
    event repository port、search registry、filter parser / extractor、admin audit HTTP handler を移す。
  - `internal/bootstrap` / `internal/shared/adapters/http/server` の DI を audit context へ向ける。
  - `internal/shared/adapters/persistence/{memory,postgres}` の audit event store / repository は
    audit port を実装するよう import を更新する。
  - OAuth2 / authentication / account activity など既存呼び出し側は、新しい audit port を参照する。
- **ui**:
  - 監査ログ画面の API client / route / navigation の SCL 所有 context 表現を audit に合わせる。
    既存 URL は互換維持する。
- **work-items**:
  - 後続の `wi-46-authentication-event-attribute-emit-and-correlation-search` が audit context を
    参照するよう必要な記述を更新する。

## Out of Scope

- HTTP path の破壊的変更。既存 `/api/admin/audit_events` と
  `/api/admin/audit_events/export` は維持する。
- 監査イベント schema / wire response の意味変更。
- username / IP の emit 値 populate、平文入力からの PII 検索、UI 検索ビルダーの追加
  (wi-46 の範囲)。
- SIEM streaming、long-term archive、outbox replay、監査イベント署名 / tamper evidence。
- identity-management への移管。監査は IM ではなく audit context として独立させる。

## Plan

- **SCL-first**: 先に `spec/contexts/audit.yaml` を追加し、OAuth2 から監査 API / models /
  permission を移す。`just yaml-check` で context 解決と ID 整合を確認する。
- **互換維持**: HTTP route と JSON shape は不変。移動後も既存 handler tests をそのまま通す。
- **移設順**:
  1. audit SCL context と architecture map を追加する。
  2. Go の inner layer (`internal/audit/ports`, `internal/audit/usecases`) を作り、型と parser /
     extractor を移す。
  3. HTTP handler を `internal/audit/adapters/http` へ移し、server router から登録する。
  4. persistence adapter / bootstrap / callers の import を audit port へ更新する。
  5. UI の context 表現と生成物を同期する。
- **却下案**: identity-management context へ寄せる案は採らない。監査イベントはユーザー管理より広く、
  token / client / tenant / key / federation の調査軸を持つため、IM の責務を不自然に広げる。
- **移行リスク**: package 移動による import churn が大きいため、挙動変更を混ぜず、既存テストの
  green 維持を主な回帰ネットにする。

## Tasks

- [x] T001 [SCL] `spec/contexts/audit.yaml` を追加し、監査 models / interfaces / permissions /
  user_experience を audit context へ移す。`just yaml-check`。
- [x] T002 [Arch] `ARCHITECTURE.md` に audit bounded context と package ownership を追加する。
- [x] T003 [Go/ports-usecases] `backend/audit/ports` / `backend/audit/usecases` を作り、監査
  repository port、search registry、filter parser / extractor を移す。`just test-go`。
- [x] T004 [Go/adapters] admin audit HTTP handler と route registration を audit context に移す。
  既存 route / response は互換維持。`just test-go`。
- [x] T005 [Go/infrastructure] bootstrap / shared server DI / memory・postgres repository / 呼び出し側の
  imports を audit port に更新する。`just test-go && just lint-go && just build-go`。
- [x] T006 [UI] admin audit page の context ownership 表現と生成物を audit context に同期する。
  `just verify-ui`。
- [x] T007 [Verify] `just verify`、completion 追記、`done/` へ移動、commit。

## Verification

- `just yaml-check`
- `just test-go`
- `just lint-go`
- `just build-go`
- `just verify-ui`
- `just verify`

## Risk Notes

medium。監査 API は admin security workflow の中心であり、テナント境界、system_admin の
all_tenants、export、PII を含まない検索属性、retention sweep といった保証を壊せない。
ただし本 WI の意図は ownership / context 境界の移設であり、HTTP contract と検索意味論は不変にする。
実装時は SCL と Go package の移動を先に行い、route / JSON / authorization の差分を出さないことを
検証ゲートにする。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: 監査イベントの読み出し API (`ListAdminAuditEvents` / `GetAdminAuditEvent` /
  `ExportAdminAuditEvents`)、検索属性 registry、PII 変換、`AdminAuditEventsRead`
  permission、保持期間 objective を OAuth2 context から独立した `Audit` bounded context
  (`spec/contexts/audit.yaml`) へ移設した。Go 側は `backend/audit/{ports,usecases,
  adapters/http,adapters/persistence/{memory,postgres}}` を新設し、`backend/oauth2`・
  `backend/authentication`・`backend/bootstrap`・中央 HTTP router の import 参照を
  audit port へ切り替えた。`TenantSaltStore` port も、監査検索の PII 相関ハッシュに
  本質的な役割を持つため audit context へ移した (OAuth2 の emit 経路は
  `auditports.TenantSaltStore` を参照する側になる)。`ARCHITECTURE.md` の Context Map
  に `Audit` 行を追加し、`sqlc.yaml` / postgres の queries・sqlcgen も audit 側へ分離した。
  HTTP route (`/api/admin/audit_events` 系)・JSON レスポンス形状・検索意味論・認可判定は
  変更していない。UI 側には SCL bounded context を示す明示的な参照が無く、既存 URL も
  変えていないため追加の改修は不要だった。副次的に、`just yaml-check` に抜けていた
  `--architecture` チェックを `justfile` の `yaml-check` composite recipe へ組み込んだ。
- **Affected Guarantees State**: HTTP route (`/api/admin/audit_events`,
  `/api/admin/audit_events/export`, `/api/admin/audit_events/{id}`)、JSON レスポンス
  shape、`AdminAuditEventsRead` の認可条件、監査検索の PII 変換 (hash / IP truncate) と
  tenant 境界は不変。所有 bounded context のみ OAuth2 → Audit に変わった。
- **Verification Results**:
  - `just yaml-check` — passed (SCL / work-items / ids / architecture)
  - `just test-go` — passed
  - `just verify-go` — passed (golangci-lint 0 issues, race-enabled Go tests)
  - `just build-go` — passed
  - `just verify-ui` — passed (format / lint / typecheck / unit test / build)
  - `just verify` — passed
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: ローカル開発環境
  - 実行主体: Claude (Sonnet 5)
  - 対象ソース版: main（コミット前）
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
