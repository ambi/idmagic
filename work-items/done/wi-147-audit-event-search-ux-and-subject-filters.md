---
status: completed
archived_from: 97830e45aec6c951607df66464d19852604150dd
---

# 監査イベント検索の条件 UI とユーザー相関検索を、管理者が意味を誤解しない形に整理する

既存の完了記録を監査し、ADR-104、SCL、Go domain event、UI の間に残っていた契約不整合を解消した。
ADR-104 は username だけでなく IP / User-Agent / device fingerprint の hash / truncate も撤回する
決定へ訂正し、位置情報は country code only を維持した。Authentication SCL は平文の `ip` /
`userAgent` / `deviceFingerprint` へ統一し、派生 HTML / JSON Schema / OpenAPI を再生成した。
OAuth2 の確定ユーザーイベントと `AuthenticationStepFailed` の Go 型から残存 `UsernameHash` を除去し、
JSON field 集合を固定する domain regression tests を追加した。監査検索 route は URL search を React
`key` としてページを再生成し、同一 route の履歴移動でも URL / loader の条件・結果を復元する。

全文は `git show 97830e45aec6c951607df66464d19852604150dd:<このファイルのパス>` で読む。
