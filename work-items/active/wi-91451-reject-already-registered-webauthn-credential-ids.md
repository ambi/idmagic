---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-07
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authentication/features/webauthn.md, requirement: REQ-AUTHENTICATION-038 }
---

# WebAuthn の登録で、既に登録された credential ID を拒否する

## 動機

[WebAuthn の登録手順](https://www.w3.org/TR/webauthn-3/#sctn-registering-a-new-credential)は、credential ID がどのユーザーにも未登録であることを確かめ、既知なら登録を失敗させるよう求める（SHOULD）。
アテステーション `none` の登録は秘密鍵の所持を証明しないため、他人の credential ID と公開鍵を手に入れた者は、それを自分の資格情報として登録できる。

現在の登録はこれを確かめず、保存のアダプターによって次の結果になる。

- PostgreSQL の `UpsertWebAuthnCredential` は `ON CONFLICT (credential_id) DO UPDATE` で、他人の資格情報の `sign_count`、`label`、`last_used_at` を上書きし、成功を返す。`sign_count` を大きくすると、持ち主の次のアサーションは複製の疑いとして拒否される。
- メモリのアダプターは、`user_id` を含めてレコードを置き換える。
- どちらでも、登録した側には 204 と `WebAuthnCredentialRegistered` が返る。

他人の credential ID は、その人のパスワードを知っていれば、第二要素のチャレンジの `allowCredentials` から得られる。

[wi-48351](../done/wi-48351-limit-webauthn-attestation-formats.md) の調査で見つけた。
go-webauthn を使う主要なソフトウェアは、どれも他人の記録を上書きしない。

| ソフトウェア | 扱い |
| --- | --- |
| Pocket ID | `credential_id` の `UNIQUE` 制約で挿入が失敗し、登録を拒否する |
| Authelia | `kid` の一意索引で挿入が失敗し、登録を拒否する |
| Teleport | 同じユーザーの中の重複を `AlreadyExists` で拒否する。記録はユーザーごとに分かれる |
| Gitea | `credential_id` は一意でなく、新しい行を追加する。検索は `user_id` で絞るため、他人の行に影響しない |

## 対象範囲

- 既に登録された credential ID の登録の結果を REQ-AUTHENTICATION-038 に書く。
- 登録の確定で、本人か他人かを問わず既存の credential ID を拒否し、既存の資格情報を変えない。
- PostgreSQL とメモリのアダプターの保存を、上書きしない挿入にそろえる。

## 対象外

- 既に上書きされた記録の検出と修復。上書きは `user_id` を変えないため、PostgreSQL の記録の持ち主は変わっていない。
- 第二要素とステップアップ認証以外の用途（パスワードレス、Discoverable Credential）。

## 設計

拒否は、多数の実装と同じく、一意性を保存の制約で保証する。
ユースケースの事前の照会だけでは、同じ credential ID の同時の登録が両方とも通るためである。
`webauthn_credentials` は既に `credential_id` を主キーにしているため、`ON CONFLICT DO UPDATE` を外せば、挿入の失敗が一意性を保証する。
ポートの `Save` は、衝突をアダプターに依存しないエラーとして返す。
メモリのアダプターも同じエラーを返し、共有契約で両者をそろえる。

着手時に次を決める。

- 拒否の応答。既存の「アテステーションを検証できない場合」の 400 と `invalid_webauthn` に含めるか、別のエラーコードにするか。別のコードにすると、他人の credential ID が登録済みであることを登録者へ知らせる。
- `UpsertWebAuthnCredential` の更新の部分を使う呼び出し元がほかにないこと。現在の呼び出し元は登録の確定だけで、`sign_count` の更新は `UpdateWebAuthnCredentialSignCount` が担う。

Gitea のように行をユーザーごとに分ける方式は採らない。
`credential_id` を主キーとする現在のスキーマとアサーションの検索を変え、W3C の SHOULD にも沿わないためである。

## 計画

1. 拒否の応答を決め、仕様の差分を書いてから `spec-change` で更新する。
2. 主要ユースケースの証拠を計画し、他人の credential ID の登録で RED を確かめる。
3. 保存の制約とポートのエラーを実装し、共有契約を両方のアダプターで通す。

## タスク

- [ ] T001 [Spec] 既存の credential ID の登録の結果を仕様に書く。
- [ ] T002 [Acceptance] 他人の credential ID の登録で、既存の資格情報が変わらないことを観測する境界で RED を確かめる。
- [ ] T003 [Adapter] PostgreSQL とメモリの保存を上書きしない挿入にし、衝突のエラーを共有契約で固定する。
- [ ] T004 [App] 登録の確定で衝突を拒否に変換する。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-spec` で仕様を確かめる。
- `mise run test-go-package -- ./backend/authentication/internal/webauthn/...` で、登録と両方のアダプターの共有契約を確かめる。DB を使うため、サンドボックスの外で実行する。

## リスク

本人が同じ認証器を二度登録しようとした場合も拒否になる。
登録の開始は既存の資格情報を `excludeCredentials` に入れるため、通常のクライアントはこの状況を作らない。
