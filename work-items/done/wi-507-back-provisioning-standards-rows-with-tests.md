---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Provisioning が宣言する標準の最後の 1 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無く、
**製品コードも 1 行も変わっていない**。差分は `docs/domain/provisioning/standards.md` の
`RFC7643-OUT-GROUP-RESOURCES` に対する被覆の状態である。この 1 行が
`tools/check/standards-coverage-debt.json` から消え、**台帳の `untested` は空になった**。台帳に残る
文書はもう無く、名指しを持つ id は 288 件から 289 件へ増えた。
**12 行のテストはこの行に触れてすらいなかった。** 名指しを持つ 12 行はすべて User の送出経路にあり、
`schemas` を観測する `TestClient_SendsSchemasOnResourceRepresentations` すら `fullLifecycleRequests`
という User 専用の経路を歩く。Group に触れるテストは別に 7 件あったが、作成は `POST /Groups` と応答の
id しか見ず本文を読まない。**この 1 行が残ったのは忘れられたからではなく、Group の送出の本文を読む
観測がどこにも無かったからである。**
**`partial` の境界の両側を、行の書きぶりどおりに観測した。** 内側は
`TestClient_GroupResourceBody_IsSchemasPlusMappedAttributes` が、Group の URN 1 要素の `schemas`、
対応付けが解決した属性だけで組み立てた本文、その本文に `members` が無いこと、メンバーシップが別の
`PatchOp` の増分 `add` として届くことを固定する。外側は
`TestE2E_GroupMembership_NeverSendsRemovalWhenAMemberLeaves` が持つ。**行が言う外側は拒否ではなく
非提供である**ため、メンバーが抜けた配信は成功したままで、`remove` が 1 件も現れず、残ったメンバーだけの
`add` が届く、という 3 つを同時に観測する必要があった。成功を主張しない観測では、除去を失敗として
扱う実装（Group から 1 人外しただけで配信が dead_letter に落ちる）を落とせない。
既に行の節を観測していた 2 件（`TestE2E_GroupMembership_PatchesOnlyProvisionedMembers`、
`TestE2E_GroupChange_ReachesRealDownstream`）には注記を足し、名指しを実在の観測へ繋いだ。
**欠陥を 1 件見つけ、切り出した。** 行が言う「`displayName` の取得元は `display_name_source` が選ぶ」を
読む production の経路が無い。`GroupAttributeSource` は接続を参照せず、`display_name` を常に
`Group.Name` から解決する。管理画面は自由入力を受け付け、設定は保存され、配信は成功し、観測できる差だけが
無い。この節だけはテストが書けないので
[[wi-536-group-display-name-source-selects-nothing]] へ切り出した。行の他の節はすべて宣言どおりに
実装されているので、`Adoption` の変更は要らない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
