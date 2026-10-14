---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Saml が宣言する具体例 18 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範は 1 行も動いていない。
変わったのは、`docs/domain/saml/scenarios.feature.md` が宣言する 18 件のうち 17 件が、
その id を名指しするテストから到達されるようになったことと、
`POST /api/admin/v1/saml/service-providers` が IdP プロファイルの基数違反へ返す状態コードが 500 から
契約どおりの 400 `invalid_request` になったことである。台帳は 598 件から 581 件へ減った。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
