---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-05
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/features/authorization.md, requirement: REQ-OAUTH2-005 }
  - { path: docs/modules/oauth2/features/authorization.md, requirement: REQ-OAUTH2-009 }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC9126-PAR }
---

# `request_uri` の外に重ねた認可パラメーターを、拒否せずに無視する

## 動機

RFC 9126 §4 は、PAR の `request_uri` を使う認可要求を RFC 9101 の形で組み立てると定める。
RFC 9101 §6.3 は、認可サーバーが Request Object の中のパラメーターだけを使い、同じパラメーターがクエリにあっても使わないことを MUST とする。
§5 は、互換性のためにクエリへ重ねてよいとも述べる。
OpenID Connect のクライアントの多くは、`response_type` と `scope` をクエリにも付ける。

`/authorize` は、`request_uri` と `client_id` 以外の引数を付けた要求を `invalid_request` で拒否する。
EX-OAUTH2-005-04 がこの拒否を規範にし、`TestAuthorizationRefusesRequestURIMixedWithFrontChannelParameters` が固定している。
RFC 9101 は `standards.md` に宣言されていない。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- `standards.md` に RFC 9101 の節を加え、`request_uri` の外のパラメーターを使わないことを宣言する。
- `request_uri` の外のパラメーターを無視し、PAR で保存した値だけで認可要求を処理する。`client_id` の一致の検査は残す。
- EX-OAUTH2-005-04 を、重ねたパラメーターが結果に影響しない例へ改める。
- `response_type` と `scope` をクエリに重ねた要求を `/authorize` へ送る試験を加える。

## 対象外

- `request` パラメーターと外部の `request_uri` による Request Object。[[wi-293-request-object-jar-and-jarm-signed-authorization-messages]] が扱う。

## 設計

FAPI 2.0 のクライアントに、クエリの重複を誤りとして返す扱いを残すかを T001 で確かめる。
FAPI 2.0 Security Profile が拒否を求めないなら、プロファイルによらず無視する。

## タスク

- [ ] T001 [Spec] RFC 9101 の節を加え、EX-OAUTH2-005-04 を改める。
- [ ] T002 [Acceptance] 重ねた要求で RED を確かめる。
- [ ] T003 [App] `handleAuthorize` で重ねた値を捨て、GREEN にする。既存の特性化を置き換える。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `/authorize` のブラウザーの流れの受け入れ試験：重ねた `scope` が発行するトークンへ影響しないことを観測するため。

## リスク

重ねた値を誤って PAR の値へ混ぜると、PAR が守る認可要求の完全性を失う。
試験では、クエリへ PAR と異なる `scope` と `redirect_uri` を置き、結果が PAR の値に従うことを確かめる。
