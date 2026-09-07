---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-08
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/standards.md, requirement: GDPR-ERASURE }
---

# Purge が WebAuthn 資格情報とリカバリコードを消さない

## Motivation

`docs/standards.md` の `GDPR-ERASURE` は「削除要求後は法的保存義務を除く PII を定義済み期間内に消去する。消去は IdManagement の UserLifecycle Purge 遷移と Authentication の資格情報破棄が個別に担う」と宣言している。

Purge の cascade は `backend/idmanagement/user/usecases/admin_users.go` の `cascadeDeleteForSub` にある。消しているのは Consent、リフレッシュトークン、セッション、パスワード履歴、MFA 要素、信頼済みデバイス、デバイスコード、承認要求の 8 種である。**`WebAuthnCredentialRepository` と `RecoveryCodeRepository` はここに現れない。** `AdminUserDeps` がその 2 つの port を持っていないので、配線の抜けではなく型の抜けである。両 port は `DeleteAllForSub` を「anonymize cascade から呼ばれる」と自ら doc コメントに書いているが、呼んでいるのは MFA の管理者リセット (`backend/authentication/mfa/usecases/admin_reset.go`) とリカバリコードの再発行 (`backend/authentication/recovery/usecases/recovery_codes.go`) だけである。

PostgreSQL 側の `recovery_codes` には `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE` があるが、**Purge は users の行を削除せず Tombstone 化する UPDATE なので、この cascade は発火しない。** `webauthn_credentials` には外部キーそのものが無い。

結果として、消去要求を受けた利用者の WebAuthn 資格情報（credential id、公開鍵、AAGUID、ラベル、最終利用時刻）とリカバリコードのハッシュが、その利用者の id に紐付いたまま残る。

[[wi-502-back-cross-cutting-standards-rows-with-tests]] が `GDPR-ERASURE` にテストを対応付ける過程で見つけた。同項目はテストの追加と実装の修正を混ぜない方針なので、修正をここへ切り出した。同項目が足した `backend/authentication/usecases/credential_erasure_standards_test.go` は、この 2 種を観測に含めていない。

## Scope

- `AdminUserDeps` に WebAuthn 資格情報とリカバリコードの port を足し、`cascadeDeleteForSub` から消す。
- 本番の組み立て (`backend/cmd/internal/bootstrap`) で両 port を配線する。
- `backend/authentication/usecases/credential_erasure_standards_test.go` の観測へ 2 種を足す。同ファイルは `GDPR-ERASURE` を名指しているので、行の被覆がそのまま強くなる。
- 両 port の `DeleteAllForSub` の doc コメントが言う「anonymize cascade から呼ばれる」が、実際に成り立つようにする。

## Out of Scope

- `docs/standards.md` の `GDPR-ERASURE` の文面。宣言は正しく、満たしていないのは実装である。
- Tombstone 化をやめて users の行を物理削除する設計変更。外部キーの cascade に頼る形は、監査に必要な Tombstone と両立しない。
- 既存データの後追い消去。製品は未リリースなので、残留している行は存在しない。
- 監査記録の保持。[[wi-514-audit-retention-decision-and-sweep-disagree]] が持つ。

## Risk Notes

- **port を足しただけで配線を忘れる。** `AdminUserDeps` の他の port と同じく `nil` を「未配線として何もしない」と扱う形にすると、本番の組み立てで渡し忘れても誰も落ちない。`backend/cmd/internal/bootstrap` の組み立てを突き合わせ、消去の観測は本番と同じ配線で行う。
- **消したことを行の有無だけで観測する。** WebAuthn 資格情報は公開鍵と credential id を持つので、行が消えたことに加えて、その資格情報でのサインインがもう成立しないところまで読む。
- **リカバリコードの外部キーに引きずられる。** `ON DELETE CASCADE` があるので「消える」と読める。Purge は UPDATE なので発火しない。テストは Purge を通した観測にする。

## Verification

- Purge を経た利用者について、WebAuthn 資格情報とリカバリコードがどちらも読み出せない。
- 消去前は同じ経路でどちらも読み出せる。
- `backend/authentication/usecases/credential_erasure_standards_test.go` の観測に 2 種が入り、`cascadeDeleteForSub` からそれぞれの呼び出しを外すとそのテストが落ちる。
- `mise run verify`
