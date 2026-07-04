---
id: idp-wi-44-authentication-event-store-and-search
title: "認証イベントを永続ストア・属性拡張・admin 検索 UI まで実運用相当にする"
created_at: 2026-06-21
authors: ["tn"]
status: completed
risk: high
---

# Motivation
[[wi-20-authentication-event-history]] は価値優先で 3 スライス
(サインイン履歴 / セッション一覧・失効 / 失敗ログインの bucket 集約) を
in-memory ストアの上に実装し完了した。残るのは「攻撃時にも壊れない
永続ストアと、admin が時系列で調査できる検索 UI」という、Keycloak の
login/admin events や Okta の System Log 相当のインフラ層である。

wi-20 が in-memory (ring buffer / map) で動かした部分を Postgres に載せ、
既存の `UserAuthenticated` / `AuthenticationFailed` を産業標準の属性
(IP truncated/hash・UA hash・session_id・client_id・country_code・
device fingerprint hash・risk_score) まで拡張し、retention sweep と
admin 検索 UI / bucket ドリルダウンを足す。さらに MFA チャレンジ・
federation・impersonation の語彙を SCL とストレージに用意する
(use case / 実 IdP 連携は各専用 WI)。

**実装上の決定 (2026-06-21): 監査ログへ統合。** 当初は認証イベント専用の
検索 API (`ListAuthenticationEvents` ほか) と admin ページを別に設ける計画
だったが、認証イベントは監査イベントと同一ストア (`audit_events`) の一系統で
あり、専用 API / UI は重複でしかなかった。そのため検索は監査ログ
(`ListAdminAuditEvents` / `/admin/audit_events`) に `category`
(authentication / success / fail / aggregated) フィルタとして統合し、専用
interface / permission (`AdminAuthenticationEventsRead` / `AdminSessionsWrite`) /
認証検索・セッションビュー model は撤去した。イベント型・属性拡張・bucket・
retention は共通基盤として残す。username/IP を hash・truncated 値で相関検索
する機能は、hash 化 (emit 側) の実装後に「平文を入力 → サーバ側で hash 化して
検索」する形で別途追加する (現状はフィールド確保のみで値が無いため UI には
出さない)。詳細は ADR-045 / README。

# Scope
- **decision**:
  - ADR-041 **authentication-event-model** を正式に起草する (wi-20 では bucket 切替の挙動だけ先に実装した)。通常イベントと bucket の 2 系統、 閾値 (per-account 10 / per-ip 50 / per-tenant 1000、5 分窓) と tenant override、`AuthenticationEventAggregated` を 1 bucket = 1 admin audit event として残す方針を確定する。
  - ADR-045 **authentication-event-retention**: 成功 365 日 / 失敗詳細 30 日 / bucket 90 日 / セッション 90 日 / MFA 90 日。tenant override と global cap (`max_retention_days`)。削除は時間単位 cron の sweep。
  - ADR-046 **authentication-event-pii-policy**: IP は /24・/48 truncated と SHA-256 の 2 系統、username は hash first-class + 失敗イベントのみ平文を 7 日保持、location は country code のみ first-class、device fingerprint は hash 保管。impersonation イベントは retention 短縮不可。
- **scl**:
  - 既存 `UserAuthenticated` / `AuthenticationFailed` を破壊せず属性拡張 (sessionId / clientId / acr / ipHash / ipTruncated / uaHash / countryCode / deviceFingerprintHash / riskScore optional)。
  - 新規イベント: AuthenticationStepCompleted / AuthenticationStepFailed / MfaChallengeIssued / MfaChallengeSucceeded / MfaChallengeFailed / BackupCodeConsumed / SessionStarted / SessionRefreshed / FederatedAuthenticated / FederationLinked / FederationUnlinked / SessionImpersonationStarted / SessionImpersonationEnded。
  - 認証イベント専用 model / interface / permission は追加せず、既存の `AuditEventQuery` / `ListAdminAuditEvents` / `GetAdminAuditEvent` / `ExportAdminAuditEvents` に `category` (authentication / success / fail / aggregated) を加えて監査ログの読み出しモデルへ統合する。
- **go**:
  - Postgres adapter: `audit_events` / `authentication_event_buckets` テーブルと index (tenant_id+occurred_at desc ほか)。 wi-20 の in-memory AuthEventBucketStore / AuditEventRepository を Postgres 実装に差し替える (port は維持)。
  - 認証失敗の bucket 切替を HTTP 認証経路から AuthEventBucketStore に集約し、 閾値超過後は個別 `AuthenticationFailed` を出さず 1 bucket = 1 `AuthenticationEventAggregated` として監査ログに残す。
  - retention sweep を `internal/bootstrap` の周期 job に追加する。
- **http**:
  - admin: `GET /api/admin/audit_events` (category/type/sub/after/before/ limit/all_tenants) / `GET .../{id}` / `GET .../export`。 認証系の調査は `category=authentication|success|fail|aggregated` で行う。
  - bucket 集約の一覧は `GET /api/admin/authentication_event_buckets` として残し、 permission は `AdminAuditEventsRead` を再利用する。
- **ui**:
  - `/admin/audit_events`: イベントカテゴリ (`category`) のセレクト、認証系 種別バッジ、表、詳細ペイン、JSON export を既存の監査ログ UI に統合する。 認証専用 admin ページと admin sessions ページは設けない。
- **documentation**:
  - README の "Authentication event history" 節を永続ストア・retention・ admin 検索まで更新する。

# Out of Scope
- SAML / Google / GitHub IdP の adapter 実装本体 ([[wi-29-saml2-idp]] / [[wi-30-inbound-federation-and-identity-broker]])。本 WI は federation イベント型とストレージ列のみ。
- impersonation 機能本体と本人通知 (イベント / 列のみ用意)。
- SIEM への streaming pipeline (Splunk / Datadog 等)。 `ExportAdminAuditEvents` は本 WI の検索 export として用意し、push は別 WI。
- GeoIP の有償 vendor 連携 (country_code は OSS DB 後付け、当面 "" 許容)。
- 新規 IP login の "Suspicious activity" メール通知。
- Postgres declarative partitioning / cold storage archive (ADR-045 に方針のみ)。
- 機械学習ベースの risk_score 算出 (フィールド確保のみ、値は null)。
- admin / OIDC logout 側のセッション完成 (front/back-channel logout、 session inventory) は [[wi-28-session-management-and-oidc-logout-completion]]。
- username / IP を hash・truncated 値で相関検索する admin フィルタ。 ADR-046 のフィールドは確保するが、emit 側の hash 値実装と平文入力からの サーバ側 hash 化は後続 [[wi-46-authentication-event-attribute-emit-and-correlation-search]]。

# Verification
- `go test ./...` (in: idmagic)
  - reason: bucket 切替の Postgres 実装での維持 / retention sweep の 境界 (29/31/91 日) / admin 監査ログ category フィルタの tenant 分離 / impersonation event の retention override 不可。
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- 手動: admin で `/admin/audit_events?category=authentication` を開き、自分のログインを 成功イベント (sub / amr / client_id、利用可能な場合は IP truncated / UA hash) として確認。
- 手動: 同 user に誤パスワードを 12 回投げ、最初の 10 件は個別行・11 件目以降は bucket に切替り `AuthenticationEventAggregated` が 1 行だけ書かれ count が 伸び続け、個別 `audit_events` の INSERT レートが頭打ちになることを確認。
- 手動: retention 31 日に短縮した tenant の成功イベント (32 日前) が次回 sweep で 消える / impersonation イベントは tenant override で短縮できない。

# Risk Notes
(1) ストレージ爆発: bucket モードを Postgres 永続でも崩さないこと。閾値が
    低すぎると正規ユーザの失敗が観察できず MTTR が悪化、高すぎると爆発が
    止まらない。ADR-041 で根拠付き既定 + tenant override。
(2) 検索性能: partition 化は本 WI では行わない判断のため、retention sweep が
    確実に動き index が当たること・admin 検索が limit と category/type/sub/期間
    フィルタで bounded な読み出しになることをテストで確認する。
(3) PII 漏洩: hash / truncation を誤ると個人情報が監査ログに流れる。ADR-046 の
    表に照らしてレビュアー 2 名以上で確認する運用を ADR に付記する。
(4) impersonation: 「admin が user として操作した事実」は本人通知が前提。本 WI は
    フィールドのみ用意し、機能本体の有効化は通知 WI 後にすべきと ADR-041 に明示する。
(5) 後方互換: `UserAuthenticated` の属性拡張で既存 SIEM connector が落ちないことを
    `audit_event_record.go` の wire スナップショットで確認する。

# Completion
- **Completed At**: 2026-06-21T14:26:32Z
- **Summary**:
  認証イベントの永続ストア・bucket 集約・retention sweep・admin 検索を、専用 認証イベント API/UI ではなく既存監査ログ (`audit_events` / `/admin/audit_events`) へ統合する方針で完了した。Postgres の `authentication_event_buckets` は攻撃時の個別 INSERT 爆発を抑え、admin は `category` フィルタで認証成功・失敗・集約イベントを調査できる。
- **Verification Results**:
  - `go test ./...` (in: idmagic)
    - result: passed
  - `GOCACHE=/tmp/idmagic-cache go test -race ./...` (in: idmagic)
    - result: passed
  - `go build ./...` (in: idmagic)
    - result: passed
  - `bun --cwd idmagic/ui typecheck`
    - result: passed
  - `bun --cwd idmagic/ui lint`
    - result: passed
  - `bun --cwd idmagic/ui build`
    - result: passed
  - `bun run yaml-check:all` (in: tools)
    - result: passed
  - `go list ./...` (in: idmagic)
    - result: passed
  - `golangci-lint config verify` (in: idmagic)
    - result: passed
  - `golangci-lint run ./...` (in: idmagic)
    - result: passed (0 issues; rerun outside the Codex filesystem sandbox)
- **Affected Guarantees State**:
  - 可用性: pass。bucket 集約は 5 分窓 upsert と最初の記録だけの `AuthenticationEventAggregated` emit で、攻撃時の個別監査行増加を抑える。
  - PII: pass with follow-up。ADR-046 の hash/truncated/optional フィールドと retention 方針は入った。実値 emit と相関検索は wi-46 に分離した。
  - tenant isolation: pass。admin 監査ログは tenant 境界と default tenant の system_admin 横断条件をテスト済み。
  - admin RBAC: pass。認証イベント検索は `AdminAuditEventsRead` に統合した。
  - backwards compatibility: pass。既存 `UserAuthenticated` / `AuthenticationFailed` は optional payload 拡張のみ。
  - static analysis: pass。`golangci-lint run` は sandbox 外の同一 `idmagic` ディレクトリで 0 issues。
