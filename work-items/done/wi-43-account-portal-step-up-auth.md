---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 高 sensitivity な self-service 操作に step-up 再認証を要求する

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

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
