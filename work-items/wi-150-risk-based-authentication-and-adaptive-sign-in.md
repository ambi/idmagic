---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# リスクベース認証と adaptive sign-in 判定を導入する

## Motivation
AuthenticationEvent には IP / User-Agent / country / device fingerprint / riskScore の格納余地があるが、
現状は riskScore の算出と、それを使った認証制御がない。Microsoft Entra ID Protection、
Okta adaptive policy のように、匿名 IP、異常な国変更、password spray、未知端末などの
シグナルを認証判断に使えないと、MFA を一律要求するか、危険なサインインも通すかの二択になる。

本 WI は、認証イベントからリスクシグナルを算出し、テナント・アプリの sign-in policy で
MFA 要求、ブロック、パスワード変更要求に反映できる adaptive sign-in を導入する。

## Scope
- **scl**:
  - `Authentication` に RiskSignal / RiskScore / RiskLevel / RiskAssessment を追加する。
  - 認証イベントの riskScore を実値として算出・保存する scenario を追加する。
  - `Application` の AppSignInPolicy condition に risk level 条件と adaptive action を追加する。
  - リスク判定 events と admin report projection を追加する。
- **go**:
  - 既存 AuthenticationEvent からルールベースの risk evaluator を実装する。
  - login / authorize / step-up 経路に risk assessment を差し込み、policy action を fail-closed に適用する。
  - risk report 用 repository / query を追加する。
- **http**:
  - risk policy 設定 API と risky sign-ins / risky users の admin API を追加する。
- **ui**:
  - sign-in policy editor に risk 条件を追加し、admin に risky sign-ins / users レポートを追加する。
- **documentation**:
  - README に初期 risk signal、評価順、誤検知時の運用を追記する。

## Out of Scope
- 機械学習 / UEBA による本格的な異常検知。
- 外部 threat intelligence / Tor exit list / commercial IP reputation 連携。
- managed device posture 条件。これは `wi-151-managed-device-inventory-and-posture-access-conditions` で扱う。
- 自動アカウント復旧 workflow。

## Plan
- 初期実装は説明可能なルールベース evaluator に限定し、riskScore と RiskLevel を audit/report に残す。
- policy action は `allow` / `require_mfa` / `require_password_change` / `block` の構造化 enum とし、自由文字列にしない。
- high risk の既定挙動は管理者設定がない限りブロックではなく MFA 要求に留め、policy 明示時のみブロックを許可する。
- AuthenticationEvent の既存検索と整合し、リスク算出値は後から再計算できるよう signal の根拠も保存する。

## Tasks
- [ ] T001 [SCL] RiskSignal / RiskAssessment と policy condition / action を追加する。
- [ ] T002 [Decision] risk signal、score 境界、初期 action、監査保持方針を ADR に記録する。
- [ ] T003 [App] risk evaluator と login / authorize への適用を実装する。
- [ ] T004 [HTTP] risk policy と risky sign-in report API を追加する。
- [ ] T005 [UI] risk 条件の policy editor と report 画面を追加する。
- [ ] T006 [Verify] SCL、Go、UI、手動シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just test-go`
- `just verify-ui`
- 手動: 通常サインインは low risk として許可され、未知国または短時間失敗連続後のサインインは MFA 要求またはブロックされることを確認する。
- 手動: admin report で risk signal、risk level、適用 action、対象 user / client を確認できること。

## Risk Notes
リスク判定は誤検知でログイン不能、過小検知で侵害許容につながる。初期は説明可能なルールと保守的な action に限定し、管理者が report で根拠を確認できるようにする。policy 適用は既定拒否ではなく既定安全動作を明示し、block は管理者の明示設定に限定する。
