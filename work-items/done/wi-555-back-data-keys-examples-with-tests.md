---
status: completed
archived_from: 9f739489f7edcab1376afb3550bd8794b91d7cd3
---

# DataKeys が宣言する具体例 10 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範は 1 行も動いていない。
変わったのは、`docs/domain/data-keys/scenarios.feature.md` が宣言する 10 件すべてが、
その id を名指しするテストから到達されるようになったことと、具体例のとおりに振る舞っていなかった 2 か所である。
disable した DEK の版で暗号文を復号できていたのを、フェイルクローズで拒否するようにした。
MasterKey プロバイダーへの wrap の失敗を、`DataKeyUnavailableError` にあたる番兵で識別できるようにした。
番兵は `ErrDecryptionFailed` から `ErrDataKeyUnavailable` へ改名し、契約の型と名前を揃えた。
台帳は 47 件から 37 件へ減った。残した行は無い。

全文は `git show 9f739489f7edcab1376afb3550bd8794b91d7cd3:<このファイルのパス>` で読む。
