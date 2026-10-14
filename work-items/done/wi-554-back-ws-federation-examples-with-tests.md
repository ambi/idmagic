---
status: completed
archived_from: 03d2c0da9cc6610c2c564e50fe33c845c8a50fb4
---

# WsFederation が宣言する具体例 11 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範は 1 行も動いていない。
変わったのは、`docs/domain/ws-federation/scenarios.feature.md` が宣言する 11 件のうち 9 件が、
その id を名指しするテストから到達されるようになったことと、
能動 STS `/trust/usernamemixed` が返す RSTR の assertion が IdP の証明書で検証できるようになったことである。
修正前は `requests_wstrust.BuildRSTR` の整形が署名済み assertion の内部に空白を足し、署名を壊していた。
台帳は 57 件から 48 件へ減った。残した 2 件には `blocked_by` と `finding` を書いた。

全文は `git show 03d2c0da9cc6610c2c564e50fe33c845c8a50fb4:<このファイルのパス>` で読む。
