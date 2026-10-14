---
status: completed
archived_from: 2c4d62bd910232b33df0daa84f70a8e6f20e732b
---

# Tenancy が宣言する具体例 39 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `main` に対して「no normative specification change」を返した。
Tenancy が宣言する対象 39 件を実装と照合し、実装と一致する 37 件を `//spec:covers` を持つテストへ結び付け、被覆台帳から外した。
足りなかった拒否経路、秘密値の不在、永続化値、locale fallback、UI 表示などの観測も補強した。
実装と一致しない 2 件は [[wi-578-align-cross-tenant-branding-asset-refusal]] と [[wi-579-align-branding-asset-gateway-example]] に切り出し、台帳へ `blocked_by` と `finding` を残した。

全文は `git show 2c4d62bd910232b33df0daa84f70a8e6f20e732b:<このファイルのパス>` で読む。
