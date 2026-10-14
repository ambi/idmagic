---
status: completed
archived_from: 9725e0fb9756e1e00d69b3c35fcffe76a79042d2
---

# 認証イベントに username / IP を PII-safe な検索属性として emit し、相関検索 UI を載せる

認証イベントの username / IP / user-agent を PII-safe な監査検索属性として emit するようにした。
`UserAuthenticated` / `AuthenticationFailed` payload に `usernameHash` / `ipTruncated` /
`ipHash` / `uaHash` を追加し、sidecar 検索属性 `actor.username` / `client.ip` は transform 済み値だけを
保存する。admin 監査ログ UI には検索属性ビルダーを追加し、username / IP / event.type / outcome /
target.id / session.id / transaction.id を AND 条件で検索できるようにした。
- username は `NormalizeUsername` 後に tenant salt 付き `SaltedHash` で保存・検索する。
- client IP は検索用に `TruncateIP` の IPv4 /24・IPv6 /48 値を保存し、payload には salt 付き
`ipHash` も保持する。
- `AuthenticationFailed.username` の平文は retention sweep で 7 日超過後に JSON null 化し、
`usernameHash` は相関検索用に保持する。
- 既存の `user_id` query param と UI の sub フィルタ名を揃え、従来のカテゴリ / 期間 / export は互換維持。

全文は `git show 9725e0fb9756e1e00d69b3c35fcffe76a79042d2:<このファイルのパス>` で読む。
