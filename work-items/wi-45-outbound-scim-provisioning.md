---
depends_on: [wi-126-async-job-runner, wi-184-transactional-event-log-foundation]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-21
---

# 下流 SaaS への outbound provisioning (SCIM client / lifecycle management) を実装する

## Motivation
[[wi-31-scim2-provisioning]] は idmagic が SCIM **server** として外部
system of record (Okta / Google IAM / Entra ID) から user / group を
受け取る inbound 方向を扱い、`out_of_scope` に「idmagic が SCIM client
として外部アプリへ push provisioning すること」を明示的に除外している。

しかし Okta / Entra ID / OneLogin の中核機能の 1 つは逆方向、すなわち
IdP 自身が source of truth となり、接続済みの下流 SaaS (Slack /
Salesforce / GitHub / Zoom 等) へ user / group を **push** して
provisioning / deprovisioning / group sync する outbound lifecycle
management である。これが無いと「入社で IdP に user を作る → 各 SaaS に
自動で account が生える / 退職で一斉に無効化される」という、企業が IdP に
最も期待する運用が成立しない。idmagic は inbound SCIM (wi-31) と
outbound SCIM (本 WI) の双方を持って初めて provisioning hub になる。

本 WI は idmagic を SCIM 2.0 **client** にする。downstream connector を
tenant-scoped に登録し、user/group のライフサイクル変化 (作成・属性更新・
disable・delete・group membership 変更) を outbound event として下流の
SCIM service provider へ反映する。確実性のため push は同期呼び出しでなく
既存の outbox / queue を介した非同期・冪等・retry 付きの provisioning job
として実装する。

## Scope
- **decision**:
  - 新規 ADR: outbound provisioning model を定義する。connector (下流 app) の設定 (base URL / bearer token / 対象 group による割当 / attribute mapping)、push trigger となる内部イベント、active=false / delete の 下流への翻訳、冪等性 (external_id 相関) と retry / backoff、部分失敗時の per-connector エラー状態と再同期方針を確定する。
  - provisioning の真実源は内部 user/group aggregate であり、下流は射影 (mirror) とする。下流での手動変更は次回 reconcile で上書きする方針を ADR で明示する (Okta の "Sync Password" 相当の例外は out_of_scope)。
- **scl**:
  - 新規 model: ProvisioningConnector (下流 app の接続定義) / ProvisioningAssignment (user|group → connector の割当) / ProvisioningJob (poly pending|succeeded|failed、external_id 相関)。
  - 新規イベント: ConnectorRegistered / ConnectorUpdated / ConnectorDisabled / UserProvisioned / UserDeprovisioned / UserPushFailed / GroupMembershipPushed / ProvisioningReconciled。
  - 新規 interface (admin): RegisterConnector / UpdateConnector / ListConnectors / AssignToConnector / ListProvisioningJobs / RetryProvisioningJob / ReconcileConnector。 permission `AdminProvisioningRead` / `AdminProvisioningWrite`。
- **go**:
  - SCIM client adapter: 下流 `/scim/v2/Users` `/scim/v2/Groups` への POST / PATCH / DELETE を仕様準拠で送る。external_id で相関し、409/404 を 冪等に解決する。bearer token は secret として扱いログに出さない。
  - push を outbox / queue 経由の非同期 ProvisioningJob として実装し、 指数 backoff の retry と dead-letter、per-connector の rate limit を持つ。
  - Postgres adapter: `provisioning_connectors` / `provisioning_assignments` / `provisioning_jobs` テーブルと index。connector token は既存の secret 保管方針に従う。
  - 内部 user/group lifecycle イベント (create / attribute update / disable / delete / membership 変更) を購読して対象 connector への job を起こす。
- **ui**:
  - admin settings に「プロビジョニング (下流連携)」: connector の登録 / 編集 / 無効化、attribute mapping、割当 group、last sync、エラー履歴、 個別 job の retry と connector 単位の reconcile。

## Out of Scope
- inbound SCIM server 機能 (wi-31 が扱う)。
- 各 SaaS 固有の非標準 provisioning API (SCIM を話さない app 向け connector)。
- password sync / 下流からの逆方向 import (それは inbound = wi-31)。
- HRIS / ディレクトリ connector (LDAP / AD)。
- SaaS app カタログ (事前定義テンプレート) の整備。初期は汎用 SCIM connector のみ。

## Plan
- 既存 `backend/scim` は inbound SCIM server なので再利用せず、Application context が所有する下流 SaaS connection と outbound delivery capability を分離する。SCIM wire client/serializer は `backend/scim` の outbound adapter として protocol-local に置く。
- user/group/assignment の確定 event を [[wi-184-transactional-event-log-foundation]] の event log/outbox から受け、`(tenant, connection, source aggregate, source version)` を idempotency key に配送 intent を作る。HTTP request 内で下流を呼ばない。
- connection は base URL、auth credential reference、capabilities、attribute mapping、deprovision policy を持つ。接続 test/discovery で `/ServiceProviderConfig` と schema を取得し、任意 URL・redirect を制限して SSRF/credential leak を防ぐ。
- delivery は per-connection 順序、指数 backoff、Retry-After、dead-letter、manual retry を持つ。core runtime は [[wi-126-async-job-runner]] を完了前提に追加するか、本 WI 内に互換 handler/repository を実装して後で移行するか着手時に決め、二重 queue は作らない。
- remote SCIM ID と local resource の mapping を durable に保持し、PATCH 非対応先は PUT fallback、delete は connection policy により disable/delete を選ぶ。

## Tasks
- [ ] T001 [Dependency/ADR] wi-126/wi-184 の利用可否を確定し、outbound ownership、delivery guarantee、deprovision semantics、secret 保護を ADR に記録する。
- [ ] T002 [SCL] application/scim に connection、mapping、remote link、delivery lifecycle、management interfaces/events/constraints/contracts/scenarios を追加して再生成する。
- [ ] T003 [Domain] ProvisioningConnection、AttributeMapping、RemoteResourceLink、DeliveryIntent を実装し、source version の単調性と tenant/connection uniqueness をテストする。
- [ ] T004 [Persistence] connection/link/intent repository、credential reference、lease/retry fields を memory/PostgreSQL に追加する。
- [ ] T005 [SCIM Client] discovery、Users/Groups CRUD、PATCH→PUT fallback、pagination/error/Retry-After、redirect/TLS 制限を実装し mock server contract test を作る。
- [ ] T006 [Event/Worker] user/group/assignment event projector と idempotent job handler、ordering、retry/dead-letter を実装する。
- [ ] T007 [Admin HTTP/UI] connection CRUD/test、mapping editor、delivery list/detail/retry と secret write-only 表示を追加する。
- [ ] T008 [Verify] create/update/disable/delete、group membership、duplicate/out-of-order event、429/5xx、credential rotation、tenant 越境を end-to-end 検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: connector を登録 → 内部で user 作成 → 下流 (mock SCIM server) に POST が届く → user disable → 下流に PATCH active=false が届く流れを確認する。
- 手動: 下流が一時的に 5xx を返すとき job が retry され、復旧後に収束する (冪等) ことを確認する。

## Risk Notes
outbound provisioning は外部システムの状態を idmagic が書き換える契約であり、
誤割当・誤った deprovision は下流 SaaS のアカウント大量無効化など破壊的影響を
持つ。active=false / delete の下流への翻訳と割当条件を ADR で先に固定し、
push は必ず冪等・retry 可能な非同期 job とする。inbound (wi-31) と outbound
(本 WI) で active=false の意味を一貫させる。
