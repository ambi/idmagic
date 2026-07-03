---
id: idp-wi-36-oauth2-audit-event-tenant-scoping
title: "oauth2 / token / consent / client 系の監査イベントにも emit 時点で tenant_id を載せる"
created_at: 2026-06-20
authors: ["tn"]
status: completed
risk: medium
---
# Motivation
[[wi-35-audit-event-tenant-scoping]] で Authentication + identity 系の
15 イベントに emit 時点で tenant_id を載せ、テナント所属 admin の監査
ビュー (`/admin/audit_events`) に出るようにした。しかし wi-35 は意図的に
Authentication + identity に絞ったため、**oauth2 component が owns する
イベント群は依然 tenant_id 空で記録され、テナント admin の監査ビューから
除外されたまま**になっている。

対象 (scl.yaml の OAuth2 component `owns_events`、emit 実績あり):
- クライアント: ClientRegistered / AdminClientCreated / AdminClientUpdated
  / AdminClientDeleted
- 同意: ConsentGranted / ConsentRevoked
- 認可コード: AuthorizationCodeIssued / AuthorizationCodeRedeemed
- トークン: AccessTokenIssued / RefreshTokenIssued / RefreshTokenRotated /
  RefreshTokenReuseDetected / TokenRevoked / TokenIntrospected
- PAR: PARStored
- デバイスフロー: DeviceAuthorizationRequested / DeviceAuthorizationApproved
  / DeviceAuthorizationDenied

これらは [[wi-35-audit-event-tenant-scoping]] と同じ構造的ギャップ:
newAuditEventRecord が payload["tenantId"] を拾えず tenant_id 空で記録 →
auditEventMatches がテナント絞り込みで除外。テナント admin はクライアント
登録・同意付与/失効・トークン失効などを自テナントの監査ビューで一切
確認できない。

wi-35 と同一方針 (emit 時点で tenant_id を載せ、フィルタは変えない) で
このギャップを埋める。tenant コンテキストは各 emit 箇所で取得可能と確認済み:
- HTTP 層 (authorize_handler の ConsentGranted / AuthorizationCodeIssued、
  token_handler の client_credentials AccessTokenIssued / TokenIntrospected)
  は requestTenantID(c)。
- oauth2 usecase 層 (exchange_code / refresh_tokens / device_flow /
  register_client / admin_clients / admin_consents / revoke_token /
  push_authorization_request) は tenancy.TenantID(ctx) もしくは解決済み
  aggregate (client / consent / refresh record) の TenantID。

# Scope
- **scl**: spec/scl.yaml の以下 event payload に `tenantId: { type: String }` を 追加する: ClientRegistered / AdminClientCreated / AdminClientUpdated / AdminClientDeleted / ConsentGranted / ConsentRevoked / AuthorizationCodeIssued / AuthorizationCodeRedeemed / AccessTokenIssued / RefreshTokenIssued / RefreshTokenRotated / RefreshTokenReuseDetected / TokenRevoked / TokenIntrospected / PARStored / DeviceAuthorizationRequested / DeviceAuthorizationApproved / DeviceAuthorizationDenied。, これらは OAuth2 component が owns する event。owns_events / component 所属は不変。AuditEventResponse.tenant_id (既存) も不変。, spec/idmagic.html を scl-to-html:idmagic で再生成する。
- **go**: internal/spec/events.go の対象 18 struct に `TenantID string \`json:"tenantId"\`` を追加 (SCL twin、json タグは tenantId)。ConsentGrantedEvent / ConsentRevokedEvent の Go 型名はそのまま (EventType は ConsentGranted / ConsentRevoked)。, emit 各所で tenant_id を渡す: - internal/adapters/http/authorize_handler.go: ConsentGranted /
  AuthorizationCodeIssued は requestTenantID(c)。
- internal/adapters/http/token_handler.go: client_credentials の
  AccessTokenIssued と TokenIntrospected は requestTenantID(c)。
- internal/oauth2/usecases/exchange_code.go / refresh_tokens.go /
  device_flow.go / register_client.go / admin_clients.go /
  admin_consents.go / revoke_token.go / push_authorization_request.go:
  tenancy.TenantID(ctx) もしくは scope 内の client/consent/refresh
  record の TenantID。, newAuditEventRecord は変更なし (wi-35 と同じく既存抽出が効くようになる)。
- **ui**: 変更なし。AdminAuditEventsPage / AdminDashboardPage は既存のまま、 クライアント・同意・トークン系イベントがテナント絞り込みで返るように なる。
- **documentation**: README には新規記述を増やさない。意味的差分は completion に記録する。

# Out of Scope
- SigningKeyRotated: 現状の署名鍵はインスタンス全体で 1 系統 (per-tenant 鍵は [[wi-32-kms-hsm-and-per-tenant-signing-keys]] の領分) のため、 帰属テナントが定義できない。tenant_id は載せず、wi-32 で per-tenant 鍵を 入れる際に併せて対応する。
- フィルタ semantics の変更 (auditEventMatches / parseAuditEventQuery)。 wi-35 と同様、emit 時 tenant_id 付与のみで解決する。
- 高頻度トークンイベントの aggregation / bucketing / retention。これは [[wi-20-authentication-event-history]] の専用サブシステムの領分。本 WI は 既存イベントの「テナント帰属の可視化」だけに絞り、量・保管は変えない。
- 既存 (legacy) の tenant_id 空レコードの遡及補正 / バックフィル。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動 1 (client): admin でクライアントを登録 → `/admin/audit_events` を type=ClientRegistered (および AdminClientCreated) で絞ると自テナントに出ること。
- 手動 2 (consent / token): authorization code フローを 1 回通す → ConsentGranted / AuthorizationCodeIssued / AuthorizationCodeRedeemed / AccessTokenIssued / RefreshTokenIssued が自テナントの監査ビューに出ること。
- 手動 3 (cross-tenant 隔離): 2 テナント構成で、テナント A の admin が テナント B の token / consent イベントを見られないこと (all_tenants OFF)。
- 手動 4 (key 除外): 署名鍵ローテーションを起こしても SigningKeyRotated は tenant_id 空のままで、テナント admin ビューには出ないこと (意図どおり)。

# Risk Notes
event struct と scl.yaml を twin で 18 件触るため、片方だけの更新は
spec↔impl drift になる。events_test と coherence_test を緑にして担保する。

oauth2 flow は emit 箇所が usecase / HTTP 両層に散らばる。各箇所で
「正しいテナント」を渡すのが肝。token_handler の client_credentials は
user が存在せず client が RS/AS をまたぐため、requestTenantID(c)
(リクエストパス由来) を正とする。TokenIntrospected は RS client の経路だが
introspection も realm path 配下なので requestTenantID(c) で一貫させる。

本 WI 完了で、SigningKeyRotated を除く監査イベントが概ねテナント帰属を
持つ。SigningKeyRotated はインスタンス全体鍵のため意図的に空のままで、
[[wi-32-kms-hsm-and-per-tenant-signing-keys]] で per-tenant 鍵を入れる
ときに対応する。境界を completion に明示する。

# Completion
- **Completed At**: 2026-06-20
- **Summary**:
  [[wi-35-audit-event-tenant-scoping]] の残課題だった OAuth2 component の
  18 イベントに tenantId を追加し、emit 時点で所属テナントを載せた。これで
  クライアント登録・同意付与/失効・認可コード・トークン発行/失効/再利用検出・
  PAR・デバイスフローの各イベントが、テナント所属 admin の
  `/admin/audit_events` に出るようになった。wi-35 と同じく newAuditEventRecord
  と監査フィルタは一切変更していない (既存の payload["tenantId"] 抽出が効く
  ようになっただけ)。

  SigningKeyRotated は意図的に対象外。現状の署名鍵はインスタンス全体で
  1 系統のため帰属テナントが定義できず、tenant_id 空のまま残る。per-tenant
  鍵を入れる [[wi-32-kms-hsm-and-per-tenant-signing-keys]] で対応する。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
