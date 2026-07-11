---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-06-21
---

# セカンダリ / リカバリ用メール・電話の self-service 管理

## Motivation
[[wi-21-end-user-account-portal]] の account portal は primary email の
変更 (再検証フロー) までを実装した。一方、業界の "マイページ" は primary
以外に **セカンダリメール**（複数の連絡先）と **リカバリ用メール / 電話**
(アカウント回復・通知の宛先) を持つのが一般的:

- Google: 再設定用メール / 再設定用電話番号。
- Okta / Keycloak: secondary email, recovery factors。

本 WI はこれを self-service として切り出す。primary email の変更は wi-21 で
済んでいるため、本 WI は **複数連絡先のモデルと検証フロー** に集中する。

注意 (drift): [[wi-19-rich-user-attributes]] は multi-valued contacts
(`user_emails`) を一旦ドロップした (thin-core / sparse attribute 方針)。
本 WI ではまず「単数の recovery_email / recovery_phone を属性として持つ」
最小形から入り、複数 secondary email が必要になった段階で companion table を
足すか attribute の string_array で表すかを ADR で決める。

## Scope
- **decision**:
  - 新規 ADR: recovery / secondary 連絡先の格納形式。recovery_email / recovery_phone を組み込み属性 (ADR-040 の sparse attribute) として持つか、 専用 companion table を切るかを決める。検証は ADR-030 の one-time token と 同方針 (hash 保存・単発消費・期限付き) にする。
- **scl**:
  - 新規 interface: UpdateRecoveryEmail / UpdateRecoveryPhone / AddSecondaryEmail / VerifySecondaryEmail / RemoveSecondaryEmail (self)。 対応 model と検証イベント (RecoveryContactUpdated 等) を追加する。
- **go**:
  - 検証トークンストア (port + memory + postgres + migration) を email change の パターン (EmailChangeTokenStore) に倣って追加。usecase は actor.sub 固定。
- **http**:
  - `/api/account/recovery_email` (PUT) / `/api/account/recovery_phone` (PUT) / `/api/account/emails` (POST/DELETE) / `…/verify_start` / `…/verify_finish`。 全て CSRF + same-origin + 認証必須。
- **ui**:
  - AccountEmailsPage を primary / secondary / recovery に拡張し、Add / Verify / Remove / Make primary を出す。検証はメール送信のみで、リンク踏みは別ルート。
- **documentation**:
  - README の account portal 節に recovery / secondary 連絡先の扱いを追記。

## Out of Scope
- SMS による電話番号検証の実送信 (検証コード生成までで、配信は通知 WI)。
- 連絡先を使った step-up / 通知 ([[wi-43-account-portal-step-up-auth]] / 通知 WI)。

## Plan
- `User` の primary email/phone を直接増殖させず、Identity Management 所有の `RecoveryContact`（kind、normalized value、verified_at、status）として1:N管理する。ログイン識別子・通知先・recovery 手段の役割を混同しない。
- [[ADR-042-end-user-account-portal-scope]] と [[ADR-043-account-portal-csrf-and-step-up]] に従い、追加、再送、verify、primary化、削除を account portal の step-up + CSRF 対象にする。最後の利用可能な recovery contact 削除ルールは既存 MFA/recovery code と合成する。
- verification challenge は contact/action/browser transaction/tenant/user/TTL に束縛した hash のみを保存し、一回消費にする。email は [[ADR-035-smtp-email-sender-adapter]] の EmailSender、電話は provider port だけ定義し、本 WI で vendor 固定しない。
- verified secondary email の login alias/recovery利用は tenant policy で別々に opt-in し、未検証値はどちらにも使わない。値重複の許否と account enumeration 応答を SCL invariant にする。

## Tasks
- [ ] T001 [SCL] identity-management/authentication に RecoveryContact、challenge lifecycle、self-service interfaces、step-up、alias/recovery policy、events/scenarios を追加して再生成する。
- [ ] T002 [Domain] contact normalization、状態遷移、重複/最後の手段 invariant と verification challenge model を実装する。
- [ ] T003 [Persistence] contact/challenge の memory/PostgreSQL repository、tenant/user key、hash/TTL index と migration を追加する。
- [ ] T004 [Usecases] add/send/verify/resend/remove/set-primary を実装し、EmailSender/phone sender、clock、token generator、audit outbox を注入する。
- [ ] T005 [Authentication] verified contact の alias/recovery lookup を tenant policy 下で既存 login/password-reset flow に接続し、uniform response を維持する。
- [ ] T006 [Account HTTP/UI] step-up/CSRF 付き endpoint、masked list、追加・確認コード・削除画面を追加する。
- [ ] T007 [Verify] Unicode/phone normalization、重複、token replay/expiry、送信失敗、最後の回復手段、tenant 越境と account enumeration を検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-ui`
- 手動: recovery email を設定 → 確認メールのリンクで検証完了 → 再ロードで "確認済み" 表示。secondary email を追加 → 検証 → primary に昇格できる。

## Risk Notes
multi-valued contacts は wi-19 で一旦ドロップした経緯があるため、モデルの
作り方 (attribute か companion table か) を ADR で明示してから実装する。
検証トークンの単発消費・期限の取り違えが最大のリスクで、email change と同じ
テスト観点 (期限切れ・再利用拒否) を必ず置く。
