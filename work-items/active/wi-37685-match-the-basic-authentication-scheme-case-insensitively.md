---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-05
priority: p3
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/features/client.md, requirement: REQ-OAUTH2-007 }
---

# クライアント認証の `Basic` の方式名を、大文字と小文字を区別せずに照合する

## 動機

RFC 9110 §11.1 は、認証方式の名前を大文字と小文字を区別しない値とする。
`authenticateTokenClient` は `Authorization` ヘッダーを `"Basic "` の前方一致で判定し、`basic` や `BASIC` で送った `client_secret_basic` の要求を、資格情報のない要求として `invalid_client` で拒否する。
ベアラーと DPoP の方式名は、`support_http` が大文字と小文字を区別せずに照合している。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- `Basic` の方式名を大文字と小文字を区別せずに照合する。複数の認証方式の混在の判定にも同じ照合を使う。
- 小文字の方式名で送る要求を、共通のクライアント認証を通る入口（`/token`、PAR、イントロスペクション、失効）へ送る試験を加える。

## 対象外

- 組み込みの失効クライアントの判定（`handleRevoke`）は `Authorization` ヘッダーの有無だけを見ており、変えない。

## タスク

- [ ] T001 [Acceptance] 小文字の方式名の要求で RED を確かめる。
- [ ] T002 [App] 照合を直し、GREEN にする。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `client_auth_test.go` と `/token` の HTTP の試験：共通の認証処理と製品の入口の両方で観測するため。

## リスク

小さい。
