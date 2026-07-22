---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# テナント単位のリソースクォータを定義して強制する

## Motivation
大規模テナントを扱うには、一覧のページングだけでなく、テナントが保持できるリソース量と高コスト操作の上限を
明示する必要がある。User、Group、Agent、Application、Client、Consent、AuditEvent、SigningKey、
セッション、MFA 要素、ジョブ、エクスポート成果物などが無制限に増えると、DB、検索、監査、UI、バックアップ、
運用復旧のコストが予測できなくなる。

クォータがない状態では、単一テナントの増加や誤操作が他テナントの可用性へ波及する。idmagic は multi-tenant IdP
として、テナントごとの resource budget、超過時の拒否、警告、監査、system admin による調整を仕様化する必要がある。

## Scope
- **decision**:
  - 新規 ADR: quota の分類、既定値、hard / soft quota、超過時挙動、system admin override、既存テナント移行方針を決める。
- **scl**:
  - `Tenancy` context に `TenantQuota` / `TenantUsage` / `QuotaExceeded` 相当の model / event / invariant を追加する。
  - `IdentityManagement`、`Authentication`、`OAuth2`、`Application`、`SigningKeys`、`Jobs` などの作成系 interface に quota precondition を関連づける。
  - `authorization` と interface `access` に quota 参照・更新権限を追加し、tenant admin と system admin の境界を定義する。
  - `scenarios` に quota 内作成、soft warning、hard quota 超過拒否、system admin override、tenant 境界違反を追加する。
  - `objectives` に quota check の latency、fail-closed、usage counter の整合性、監査発火を追加する。
  - `flows` と `scenarios` に AdminSettings または system tenant 画面での quota / usage 表示を追加する。
- **go/domain/usecase**:
  - quota policy、usage read model、作成前チェック、作成/削除/retention 後の usage 更新を導入する。
  - 競合時に上限を超えないよう、DB 制約、transaction、advisory lock、counter table などの方式を決めて実装する。
  - quota 超過は安定した error key と監査イベントを返す。
- **persistence**:
  - tenant quota / usage の保存、migration、既存データからの backfill / reconciliation job を追加する。
  - memory / postgres の両方で同じ quota enforcement を満たす。
- **ui**:
  - system admin が tenant ごとの quota / usage を確認・更新できる画面を追加する。
  - tenant admin が自 tenant の使用量と上限、近い上限、超過時の理由を確認できる表示を追加する。
- **operations**:
  - quota 逼迫、超過拒否、usage reconciliation 差分を metrics / structured log / runbook で扱えるようにする。

## Out of Scope
- 課金・請求システムとの連携。
- プラン管理や self-service upgrade。
- API レート制限。endpoint rate limit は [[wi-27-endpoint-rate-limit-and-bot-mitigation]]、agent budget は [[wi-59-agent-governance-guardrails-audit-inventory]] の範囲。
- 一覧 API のページング。これは [[wi-159-admin-resource-cursor-pagination]] で扱う。

## Plan
- 最初に ADR で quota 分類を固める。初期対象は `users`、`groups`、`agents`、`applications`、`oauth2_clients`、`consents`、`active_sessions`、`audit_events_retained`、`jobs_active`、`export_artifacts_bytes` を候補にする。
- 作成系 usecase は quota precondition を shared service として呼ぶ。ただし bounded context の所有権を崩さず、各 context は自分の resource usage を publish する。
- Usage は強整合が必要な hard quota と、遅延集計でよい soft quota を分ける。hard quota は transaction 内で超過を防ぐ。
- 既存テナントはまず十分に大きい既定値で移行し、backfill 後に warning を出す。突然のロックアウトや作成不能は避ける。
- UI は quota を「制限」だけでなく「現在値 / 上限 / 近い上限 / 最終再計算」を見せる。

## Tasks
- [x] T001 [ADR] quota 分類、既定値、hard/soft、override、移行方針を記録する。
- [x] T002 [SCL] Tenancy の quota model / interface / permission / invariant / scenario / objective / UX を追加し、各 context の作成系 interface に関連づける。
- [x] T003 [Render] `just scl-render` で派生物を更新する。
- [ ] T004 [Go] quota policy、usage counter、作成前 enforcement、超過 error / audit event を実装する。
- [x] T005 [Persistence] quota / usage schema、migration、backfill / reconciliation を実装する。
- [x] T006 [UI] system admin / tenant admin の quota usage 表示と更新 UI を追加する。
- [x] T007 [Ops] metrics、structured log、runbook を追加する。
- [ ] T008 [Verify] `just yaml-check`、`just verify-go`、`just verify-ui`、必要に応じて `just test-ui-e2e` を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
  - reason: quota 表示・更新・超過時エラーは管理 UI の主要操作を含むため。
- 手動: quota 上限直前の tenant で作成が成功し、上限超過の作成が安定した error key と監査イベント付きで拒否されることを確認する。
- 手動: system admin の quota override 後に同じ操作が成功し、tenant admin には他 tenant の quota / usage が見えないことを確認する。
- 手動: backfill / reconciliation 実行後、usage と実データ件数の差分が許容範囲に収まることを確認する。

## Risk Notes
Quota は可用性を守る一方、誤った既定値や counter 不整合で正当な操作を拒否しやすい。
特に concurrent create で上限を超える競合、削除・retention・restore による usage 差分、既存 tenant への導入時ロックアウトが主なリスクである。
Hard quota は transaction 境界で fail-closed にし、soft quota は警告と観測に寄せる。移行時は backfill と十分な初期上限で安全側に倒す。

## Reopen Note

2026-07-23 の完了状態監査で、quota domain/repository、管理 API・UI、metrics、runbook は存在する一方、
作成系 use case から `CheckQuotaAndIncrement` / `DecrementQuota` を呼ぶ実配線と、その enforcement を
保証するテストが存在しないことを確認した。T004 と T008 を未完了へ戻し、実際の作成拒否と usage 更新を
実装・検証するまで `status: pending` のまま `work-items/` で管理する。
