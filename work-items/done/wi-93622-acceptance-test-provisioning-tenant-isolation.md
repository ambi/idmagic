---
status: completed
archived_from: 8fde8e6d80edf2c191cea8b3913e305b829be78d
---

# テナントをまたぐ Provisioning 管理参照を公開 HTTP 境界で NotFound にする

`mise run spec-diff` は main に対して規範仕様の変更がないと報告した。
`EX-PROVISIONING-015-01` を、default テナントの接続と配信を acme の管理者がセッションで参照する HTTP の境界テストで被覆した。越境の参照は接続と配信のどちらも、存在しない id と同じ 404 `provisioning_not_found` になり、接続先 URL と配信 id を返さない。
現行実装は既に境界を守っていたため、製品コードは変更していない。
`testing_stack` に、realm のテナントの管理者セッションを製品と同じ `SessionManager` で作る `AdminSessionCookie` を足した。PostgreSQL の配信の `Find` が別テナントで nil を返すことを単体テストで固定し、`tools/check/example-coverage-debt.json` から項目を削除した。

全文は `git show 8fde8e6d80edf2c191cea8b3913e305b829be78d:<このファイルのパス>` で読む。
