---
status: completed
archived_from: 2c4d62bd910232b33df0daa84f70a8e6f20e732b
---

# realm 配下の branding アセット転送例を現行 gateway と一致させる

`mise run spec-diff` は REQ-TENANCY-004 のシナリオ変更を報告した。
`EX-TENANCY-004-03` は、ゲートウェイが realm 配下の `logo_url` を転送せず画像取得に失敗するという記述から、管理画面からゲートウェイ越しにアップロードしたロゴの realm 配下の `logo_url` をゲートウェイ越しに GET すると、backend へ転送されて同じ PNG が `image/png` で返るという成功例へ変わった。
`EX-TENANCY-004-01` と重なっていた配色とフッターの手順は具体例から外した。
製品コードは変更せず、`tools/check/example-coverage-debt.json` から `EX-TENANCY-004-03` の項目を削除した。

全文は `git show 2c4d62bd910232b33df0daa84f70a8e6f20e732b:<このファイルのパス>` で読む。
