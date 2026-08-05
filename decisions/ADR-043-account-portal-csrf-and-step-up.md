---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-043: account portal の高 sensitivity 操作に step-up 再認証を要求する

## コンテキスト

[[wi-21-end-user-account-portal]] の self mutation は CSRF + same-origin と、操作ごとの
所持証明 (TOTP 解除時の有効コード等) で守られている。しかしセッション cookie が乗っ取られた
場合、攻撃者はパスワード変更・MFA factor の解除・primary email 変更・全セッション失効
といった「アカウントの支配権を奪う操作」をそのまま実行できてしまう。

Google ("確認のためもう一度ログイン")、Okta の re-authentication、Keycloak の `max_age`
相当のように、実運用 IdP は高 sensitivity 操作の直前に**直近の再認証**を要求する。本 ADR は
その横断ゲートを account portal に持ち込み、対象操作の表と recency 条件を確定する。

## 決定

[[ADR-042]] が account portal の trust boundary (self/admin) を定め、self mutation を
CSRF + same-origin で保護したのを受け、**高 sensitivity な self-service 操作**
(`ChangePassword` / `RemoveTotpFactor` / `RequestEmailChange` / `RevokeMyOtherSessions`) に
横断的な step-up 再認証ゲートを追加する。個別セッションの失効や TOTP の登録は対象外とする
(相対的に低 sensitivity)。SCL では対象 interface に `annotations: { step_up: required }` を付け、
実ハンドラとの一致をテストで機械照合し、対象表と実装の drift を防ぐ。recency 条件は「直近
`StepUpRecencySeconds` (5 分) 以内に password または MFA で (再)認証済み」とし、新規ログイン
直後はそのまま step-up 済みとして扱う (Google 同様、ログイン直後に再入力を求めない)。未通過時は
401 ではなく 403 + `step_up_required` を返す — 認証済みだがこの操作には再認証が要る、という
セマンティクスを表現するため。recency は `LoginSession.step_up_at` に永続化し、session に紐づく
ため cookie が別端末へ移っても閉じたままになる。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の Account portal trust boundary and step-up セクションにある。

## 影響

- step-up は actor 本人の再認証であり、対象 sub を変えない (self RBAC を逸脱しない)。
- gate は handler 共通ヘルパ (`requireStepUpSub` / `requireStepUpSession`) として
  sensitive ハンドラに差し込む。CSRF + same-origin ([[ADR-042]]) は引き続き全 mutation に
  掛かり、step-up はその上乗せである。
- `StepUpRequested` / `StepUpCompleted` は監査イベントに残るため、再認証の試行と成立を
  後から追跡できる。検証材料 (パスワード / コード) は payload に含めない。
- 後続: 横断ゲートに依存する sensitive 操作 (WebAuthn credential 解除等) は対象表に
  追記して一貫させる。admin 経路の step-up は本 ADR の範囲外 (end-user self-service に限定)。

## 参照

- [[wi-43-account-portal-step-up-auth]] — 本 ADR を導く WI。
- [[ADR-042]] — account portal の trust boundary と CSRF + same-origin の土台。
- [[wi-21-end-user-account-portal]] — step-up を載せる self-service 操作群。
- [[wi-26-webauthn-passkey-and-recovery-codes]] — 横断ゲートを将来利用する sensitive 操作。
