---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# `spec/` と `docs/` の境界を、名前から読める形にする

人が書く文書を 1 つの根へ集めた。`docs/` が正規文書 134 件（直下 10 件と `contexts/<context>/` 124 件）と `operations/runbooks/` を持ち、`spec/` は TypeSpec 43 件・`tspconfig.yaml`・OpenAPI ベースラインだけになった。2 つの木は `contexts/<context>/` で対応する。DOCUMENTATION_GUIDE §4 は構成図を差し替えたうえで、判定の理由——深さは重要度の裏返しにする、境界の判定は §5.9、`docs/` 直下に生成物が混ざらない——を 3 つの小節として明示した。§10 と §11 には配置を書き、§11 には runbook だけが読み手で分かれる理由（当番担当者は原因が分からない状態で呼び出される）を、手順一般の規則ではないと限定したうえで書いた。`ROOT_DOCUMENTS` には `product-overview.md` を足し、ガイドが挙げる名前を検査が拒否する食い違いを解消した。検査・生成・スキーマ・タスクの経路と、294 箇所の参照を張り替えた。規範的な要素は 1 件も動いていない。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
