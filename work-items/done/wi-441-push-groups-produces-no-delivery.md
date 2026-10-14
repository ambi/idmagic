---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# `push_groups` が配信を生まない状態を解消する

`mise run spec-diff` は `no normative specification change against main` を返す。`REQ-` シナリオは動いておらず、変わったのは `docs/domain/provisioning/standards.md` に新設した `RFC7643-OUT-GROUP-RESOURCES` の行と、実装である。`push_groups` を有効にした接続で、Group の作成・更新・削除とメンバーシップの変更が下流への書き込みまで届くようになった。それまでは 4 つの経路がすべて欠けていた —— Group の変更を捕捉する通知先がどの Context にも配線されておらず、`translateTrigger` に Group の引き金が無く、属性解決が User の集約しか扱わず、`PatchGroupMembers` はクライアントにあってポートにすら載っていなかった。設定は保存され画面は有効と表示しながら、配信は 1 件も生まれていなかった。
[[wi-403-provisioning-declares-no-scim-conformance]] が置けなかった Group リソースの行を、`partial` として置けるようになった。`partial` に留めたのは、メンバーの除去を送っていないためである (下流の現在のメンバー集合を読み戻していないので、除くべき相手を知る手段が無い)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
