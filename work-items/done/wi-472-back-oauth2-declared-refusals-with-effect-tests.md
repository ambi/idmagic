---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# OAuth2 が宣言する未検証の拒否に効果まで確かめるテストを与え、台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。
規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 40 件が
「応答だけ確かめられている」状態から「副作用を止めたことまで確かめられている」状態へ
移ったことと、その 40 件が網羅台帳から消えたことである。
`check-spec` の網羅は「テストが名指しする id」19 件から 59 件へ増え、
`example-coverage-debt.json` は 711 件から 671 件へ、OAuth2 分は 145 件から 105 件へ減った。
実装の変更は 1 か所だけで、`support_http.CheckRateLimit` が共有カウンターへ到達できないときに
エラーを伝播して 500 を返していたのを、宣言どおり `RateLimitedError` の 429 で閉じるようにした。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
