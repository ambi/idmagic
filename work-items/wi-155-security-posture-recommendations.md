---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-10
---

# セキュリティ姿勢レコメンドを管理者に提示する

## Motivation
IdMagic には MFA、鍵、SCIM token、sign-in policy、監査、HTTP hardening などの個別機能があるが、
管理者が「どの設定が弱いか」「どの改善タスクを優先すべきか」を一覧する security posture
画面はない。Okta HealthInsight のような推奨タスクがないと、機能があっても有効化漏れや
古い設定が残りやすい。

本 WI は、テナントとアプリの設定を検査し、MFA 未強制、長寿命 secret、古い signing key、
未保護アプリ、弱い password policy などを改善タスクとして提示する security posture
recommendations を導入する。

## Scope
- **scl**:
  - `System` に SecurityPostureRecommendation / RecommendationSeverity / RecommendationStatus を追加する。
  - `Tenancy` / `Application` / `Authentication` / `SigningKeys` の設定検査 scenario を追加する。
  - recommendation dismissed / resolved events を追加する。
- **go**:
  - posture check registry、各 context の read model 取得 port、recommendation evaluator を実装する。
  - dismissed 状態と自動 resolved 判定を保存する。
- **http**:
  - admin の recommendation 一覧、詳細、dismiss / restore API を追加する。
- **ui**:
  - admin dashboard または settings に security posture 画面を追加する。
- **documentation**:
  - README に初期 check 一覧、severity、dismiss の意味を追記する。

## Out of Scope
- 自動修復。
- 外部 compliance benchmark の認証やスコアリング。
- SIEM / ticketing system 連携。
- 複雑な organization-wide analytics。

## Plan
- 初期 check は既存データだけで決定できるものに限定し、推奨の根拠と対象 resource を明示する。
- Recommendation は dismissed 可能だが、対象設定が改善された場合は resolved として自動的に完了扱いにする。
- posture check は read-only にし、設定変更は既存の各管理画面へ誘導する。
- severity は `info` / `low` / `medium` / `high` のような固定 enum とし、危険度と実装優先度を混同しない。

## Tasks
- [ ] T001 [SCL] Recommendation model、status、events、初期 check scenarios を追加する。
- [ ] T002 [Decision] check registry、severity 基準、dismiss / resolved 方針を ADR に記録する。
- [ ] T003 [App] posture evaluator と dismissed state を実装する。
- [ ] T004 [HTTP] recommendation API を追加する。
- [ ] T005 [UI] security posture 画面を追加する。
- [ ] T006 [Verify] SCL、Go、UI、手動シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just test-go`
- `just verify-ui`
- 手動: MFA 未強制アプリ、古い signing key、期限なし SCIM token が recommendation として表示されることを確認する。
- 手動: recommendation を dismiss でき、対象設定を改善すると resolved 扱いになることを確認する。

## Risk Notes
recommendation は誤検知が多いと管理者に無視される。初期 check は確実に判断できる設定に限定し、根拠、対象、改善先を明示する。自動修復は行わず、read-only evaluator として安全に導入する。
