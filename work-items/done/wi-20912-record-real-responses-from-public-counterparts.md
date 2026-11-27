---
status: completed
---

# 公開の相手から実物の応答を取り、記録の再生で検査する

公開の相手四種から 2026-11-27（UTC）に実物の応答を `curl` で取り、`testing_httprecord` の記録にして再生の検査を足した。
規範の変更はない（`spec-diff` は差分なし）。
実物の応答で既存の実装との食い違いは見つからなかった。

- HIBP の Range API：`password` の接頭辞 5BAA6 の応答（`breaches_hibp/testdata/hibp-range-api.json`）。実物は行を CRLF で区切り、最後の行に改行がなく、詰め物の行の件数が 0 である。手書きの成功と詰め物のテストを置き換えた。
- Workload の issuer の Discovery：GitHub Actions と Microsoft Entra ID（Microsoft のテナント）の文書（`issuer_metadata/testdata/`）。どちらも `issuer` に末尾のスラッシュがなく、末尾にスラッシュを付けた入力は Discovery 4.3 節に従って拒否される。
- 外部の JWKS：同じ二つの issuer の JWK Set（`tokens_jose/testdata/`）。どれも RSA だけで、Entra の鍵は `alg` を持たず、RFC 7517 にない `issuer`、`cloud_instance_name` を持つ。RSA と EC が混ざる形は実物から取れず、RFC 7517 付録 A.1 の転記が `testing_oidctwin` にあるので足さなかった。
- CIMD：VS Code、Claude Code、Claude が公開する文書（`cimd_http/testdata/`）。VS Code の `device_code` の付与と草案にないメタデータ、Claude の `jwt-bearer` の付与を受理する。手書きの成功のテストを置き換えた。

Discovery と JWKS は使うパッケージごとに記録を分け、`backend/shared` の共有の記録とモジュール間の依存を作らなかった。
再生サーバーへ任意のホストの要求を送るクライアントは `testing_httprecord.RoutedClient` にまとめ、`testing_oidctwin` と `push_http` の写しを置き換えた。
記録は一度だけの取得なので `mise` タスクにはしていない。
相手の障害、応答の形の変化、解析の変更、CIMD の版の更新を、取り直す契機にする。
