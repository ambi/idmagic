---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Sourcing が宣言する標準 5 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
差分は `docs/domain/sourcing/standards.md` の 5 行に対する被覆の状態である。5 行すべてが
その行の `Statement` を区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json`
から消え、台帳は 11 件から 6 件になった。名指しを持つ id は 276 件から 281 件へ増えた。
新設したテストは Go 5 件（部分試験を数えると 24 件）で、**製品コードは 1 行も変わっていない**。
**既存テストが行を区別できていたのは 5 行のうち 1 行だけだった。** `RFC7643-SERVICE-PROVIDER-CONFIG`
は `TestScimInboundProvisioning` の途中で `authenticationSchemes` を読んでおり、観測としては足りていた。
残る 4 行は、Group の削除と単体参照が HTTP 境界に無く、拒否が防いだ効果を 1 件も観測しておらず、
誤り応答の形を `scimType` 以外まったく読んでおらず、採用の境界が Discovery 側にしか無かった。
**欠陥は 1 件も見つからなかった。** 5 行はいずれも宣言した採用を満たしている。
**起票時の Motivation が入出力を取り違えていたので直した。** Sourcing は SCIM のサービス提供者として
受け取る側であり、下流へ送るのは Provisioning である。入口の向きは、どこにテストを置くかを決める前提
なので、そのままにはできない。
既存テストを 1 件整理した。`TestScimInboundProvisioning` の 3 番目の区画にあった
`authenticationSchemes` の確認は、行に対応する試験へ移した。同じ判断を強さの違う 2 つのテストが
観測する状態を残さないためである。同区画はトークンが受け付けられることの確認として残る。
**標準の行が言っていないことを 1 つ見つけた。** User の DELETE は soft delete なので、削除後も
`GET /scim/v2/Users/{id}` は 200 を返す（Group は実削除なので 404 になる）。`RFC7644-RESOURCE-OPERATIONS`
の `Statement` は「4 操作を提供する」までしか言っておらず、削除後の到達性を宣言していないので、
この非対称は行に対する欠陥ではない。行の `Statement` を SCIM の適合範囲まで広げるかは規範の変更であり、
本項目の Out of Scope である。テストは行が言う範囲、つまり削除が User の lifecycle に及んだことを観測する。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
