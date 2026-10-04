---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format]
change_kind: bugfix
affected_spec:
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-021 }
  - { path: docs/domain/oauth2/device/README.md, requirement: REQ-OAUTH2-027 }
---

# デバイス認可で、offline_access のない交換にリフレッシュトークンを発行せず、期限切れの拒否を受け付けない

## 動機

デバイス認可の `device_code` の交換は、スコープに `offline_access` を含まなくてもリフレッシュトークンを発行する。
REQ-OAUTH2-021 は、リフレッシュトークンを `offline_access` を付与したときだけ発行すると定め、認可コードの交換はそれに従っている。
また、期限を過ぎた `user_code` の拒否は期限を確かめずに記録を `Denied` にするが、DeviceCodeFlow の状態遷移表は `拒否：400 expired_token` を求める。

## 対象範囲

- `device_code` の交換で、`offline_access` を含むスコープのときだけリフレッシュトークンを発行する。
- 期限を過ぎた `user_code` の拒否を、承認と同じく 400 と `expired_token` で拒否する。

## 対象外

- デバイス認可のほかの振る舞い。

## 設計

認可コードの交換（`exchange_code.go`）と同じ条件をデバイスの交換に置く。
拒否の経路に承認と同じ期限の判定を置く。

## タスク

- [ ] T001 [Acceptance] 二つの食い違いを RED で確認する。
- [ ] T002 [App] GREEN にする。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run verify`

## リスク

- 既存のデバイス認可のクライアントがリフレッシュトークンに依存している場合、`offline_access` を要求するよう変える必要がある。リリースノートで知らせる。
