---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# クライアントの登録とプロファイル選択が宣言する標準 6 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。
規範の宣言も、本項目における製品コードも変えていない。
宣言済みの 6 行は、正式な HTTP 入口から観測できるようになり、標準の被覆台帳から外れた。
うち `FAPI2-*` の 4 行は宣言した採用を満たしておらず、[[wi-532-fapi-security-profile-selection-applies-no-constraint]] へ切り出したうえで、そちらが実装と同時に消化した。
本項目が直接消化したのは `RFC7591-REGISTER` と `RFC8176-AMR` の 2 行である。
台帳は 23 件から 17 件になり、テストが名指す id は 268 件から 270 件へ増えた。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
