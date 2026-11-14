---
status: completed
---

# WebAuthn の登録で、既に登録された credential ID を拒否する

REQ-AUTHENTICATION-038 に、既に登録された credential ID の確定を拒否する箇条を加えた。
本人か他人かを問わず 400 と `invalid_webauthn` で拒否し、既存の資格情報を変えず、`WebAuthnCredentialRegistered` を発行しない。
応答はアテステーションの検証の失敗と区別せず、credential ID が登録済みかを登録者へ知らせない。HTTP 契約は変わらない。

一意性は保存の制約で保証する。
PostgreSQL のクエリーを `ON CONFLICT DO UPDATE` のない `InsertWebAuthnCredential` にし、一意違反をポートの `ErrWebAuthnCredentialExists` へ変換した。
メモリのアダプターも既存の鍵を置き換えず同じエラーを返し、共有契約 `RunCredentialRepository` で両者をそろえた。
`mise run spec-diff` は REQ-AUTHENTICATION-038 の変更だけを示す。

主キーはテナントを含まないため、他テナントの credential ID も拒否する。
既に上書きされた記録の検出と修復は行っていない。PostgreSQL の上書きは `user_id` を変えないが、持ち主の `sign_count` と `label` が書き換えられた行は残り得る。
HTTP の受け入れテストはメモリの保存先を使うため、PostgreSQL の上書きの防止は共有契約のアダプター試験だけが観測する。
