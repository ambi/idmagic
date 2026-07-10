---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# 管理デバイス台帳とデバイスポスチャ条件を導入する

## Motivation
現状の sign-in policy は network CIDR と再認証時間を評価できるが、Microsoft Entra の
compliant device、Google Context-Aware Access、Okta device assurance のような
端末管理状態を条件にできない。企業向け IdP では、管理端末、OS、証明書、紛失・侵害状態を
アクセス判断に使う需要が高い。

本 WI は、テナント内の device inventory と device posture を導入し、アプリごとの
sign-in policy で「管理済み端末のみ」「承認済み端末のみ」などを評価できるようにする。

## Scope
- **scl**:
  - `Authentication` に Device / DeviceUserBinding / DevicePosture / DeviceStatus を追加する。
  - `Application` の AccessCondition に managed device / approved device / posture 条件を追加する。
  - device 登録、承認、拒否、失効、posture 更新 events を追加する。
  - account portal と admin console の device 管理 interfaces を追加する。
- **go**:
  - device repository、device binding、posture evaluator、sign-in policy 評価への差し込みを実装する。
  - memory / postgres adapter と migration を追加する。
- **http**:
  - account の device 一覧 / 失効 API、admin の device 検索 / 承認 / 拒否 / posture 更新 API を追加する。
- **ui**:
  - AccountSecurityPage に自分の device 一覧、admin に device inventory と posture 表示を追加する。
- **documentation**:
  - README に device 登録・承認・失効と sign-in policy での使い方を追記する。

## Out of Scope
- 特定 MDM 製品との本番連携。
- 端末エージェントの配布、OS-level attestation の実装。
- trusted-device による MFA skip。これは `wi-91-trusted-device-remember-mfa` が扱う。
- 高度な device fingerprinting による追跡。

## Plan
- 初期実装は IdMagic 管理下の device inventory と手動/管理 API による posture 更新に限定する。
- sign-in policy では device が未登録・未承認・posture 不明の場合は条件不一致として拒否または step-up に進める。
- device identity はユーザーが見える表示名と管理者向け posture を分離し、raw fingerprint は保存しない。
- 将来 MDM connector を追加できるよう、posture 更新を port 経由にする。

## Tasks
- [ ] T001 [SCL] Device model、posture、policy condition、events / scenarios を追加する。
- [ ] T002 [Decision] device identity、posture 信頼境界、MDM 連携余地を ADR に記録する。
- [ ] T003 [App] device repository と posture evaluator を実装する。
- [ ] T004 [HTTP] account/admin device API を追加する。
- [ ] T005 [UI] account/admin の device 管理画面を追加する。
- [ ] T006 [Verify] SCL、Go、UI、手動シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just test-go`
- `just verify-ui`
- 手動: approved device 条件付きアプリに未承認端末からアクセスすると拒否され、承認後に許可されることを確認する。
- 手動: device を失効すると既存 session / 次回 authorize の扱いが仕様どおりになることを確認する。

## Risk Notes
device posture は信頼境界を誤ると見せかけのセキュリティになる。初期は外部 MDM 信号を信頼したふりをせず、管理 API で明示更新された posture だけを評価する。未登録・不明状態は許可に使わず、device 条件は policy editor で実際に評価できるものだけ保存する。
