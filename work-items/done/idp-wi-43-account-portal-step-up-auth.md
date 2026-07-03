---
id: idp-wi-43-account-portal-step-up-auth
title: "高 sensitivity な self-service 操作に step-up 再認証を要求する"
created_at: 2026-06-21
authors: ["tn"]
status: completed
risk: medium
---
# Motivation
[[wi-21-end-user-account-portal]] は self-service の mutation を CSRF +
same-origin で保護したが、**高 sensitivity な操作 (パスワード変更 / MFA factor の
解除 / primary email 変更 / 全セッション失効)** については、セッションが乗っ取られた
場合の被害を抑えるため、業界標準どおり **step-up 再認証** を要求すべきである
(Google の "確認のためもう一度ログイン"、Okta の re-auth、Keycloak の
`max_age` 相当)。

wi-21 時点では所持証明 (TOTP コード等) で個別に代替したが、横断的な step-up gate を
独立 WI として設計・実装する。これにより wi-26 の WebAuthn 解除など将来の sensitive
操作にも一貫したゲートを掛けられる。

# Scope
- **decision**: 新規 ADR-043 **account-portal-csrf-and-step-up**: 高 sensitivity な mutation の 一覧と、step-up の条件 ("直近 N 分以内に password or MFA で再認証済み"、N=5 分) を 定める。step-up 未通過は 401 ではなく **403 + `step_up_required`** で返し、UI が 再認証 modal を出す。対象表は SCL interface annotation で機械照合する。
- **scl**: 新規 interface: StartStepUpAuthentication / CompleteStepUpAuthentication。 新規イベント: StepUpRequested / StepUpCompleted。sensitive interface に step-up 要求のアノテーションを付ける。
- **go**: AuthenticationContext に "最後に factor を提示した時刻" を持たせ (既存 auth_time / セッションの amr 履歴を利用)、step-up gate の usecase を追加する。 gate は handler 共通ミドルウェアとして sensitive ハンドラに適用する。
- **http**: `/api/account/step_up/start` / `/api/account/step_up/complete`。既存の sensitive ハンドラ (change password / MFA remove / email change / revoke_others) に gate を 差し込み、未通過時は 403 + `step_up_required`。
- **ui**: sensitive 操作で 403 `step_up_required` を受けたら再認証 modal を出し、成功後に 元の操作を再試行する共通フックを SPA に追加する。
- **documentation**: README の account portal 節に step-up の対象操作と挙動を明記する。

# Out of Scope
- WebAuthn / passkey / recovery codes factor 自体の追加 ([[wi-26-webauthn-passkey-and-recovery-codes]])。
- admin 経路の step-up (本 WI は end-user self-service に限定)。

# Verification
- [object Object]
- [object Object]
- [object Object]
- 手動: 再認証から 5 分超過後にパスワード変更を試みると 403 `step_up_required` で 再認証 modal が出る。再認証成功後に変更が完了する。

# Risk Notes
最大のリスクは step-up を必要としない mutation から sensitive 操作へチェーンできて
しまう抜け道。対象表 (ADR-043) と実ハンドラの一致を SCL annotation で機械照合し、
テストで網羅する。再認証の recency 判定に使う時刻ソース (auth_time) の取り違えにも注意。

# Completion
- **Completed At**: 2026-06-21
- **Summary**:
  高 sensitivity な self-service 操作に横断的な step-up 再認証ゲートを 2 スライスで
  導入した (ADR-043 を起草)。
    - スライス 1 (backend): recency 判定 `StepUpSatisfied` は
      `max(auth_time, step_up_at)` が 5 分以内かで通過し、新規ログイン直後は再入力を
      求めない。対象表 (ChangePassword / RemoveTotpFactor / RequestEmailChange /
      RevokeMyOtherSessions) の各ハンドラに gate を差し込み、未通過は 401 ではなく
      403 + `step_up_required`。`POST /api/account/step_up/{start,complete}` で
      password / TOTP を検証し session の `step_up_at` を刻む。`LoginSession.step_up_at`
      / `AuthenticationContext.StepUpAt` / `SessionManager.RecordStepUp` を追加。
      `StepUpRequested` / `StepUpCompleted` を emit。
    - スライス 2 (SPA): `useStepUpGuard` フックが操作を包み、403 `step_up_required` を
      受けたら再認証 modal を出し、成立後に元の操作を 1 回だけ再試行する。security /
      activity / emails / change-password の各ページに適用した。
  対象表は SCL interface の `step_up: required` 注記と実ゲートを機械照合する
  (`TestStepUpAnnotatedInterfacesMatchGatedHandlers`)。WebAuthn credential 解除など
  将来の sensitive 操作は同じ gate に乗せる ([[wi-26-webauthn-passkey-and-recovery-codes]])。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - 自動: stale session が 4 つの対象エンドポイント全てで 403 step_up_required を返す 表テスト / step_up/complete (password) で同一セッションの gate が flip する end-to-end / recency 窓 (300 秒境界) / TOTP 経路 / SCL 注記とゲートの機械照合。
