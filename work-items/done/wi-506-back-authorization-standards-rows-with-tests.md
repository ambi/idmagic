---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Authorization が宣言する標準 5 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
差分は `docs/domain/authorization/standards.md` の 5 行に対する被覆の状態である。5 行すべてが
その行の `Statement` を区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json`
から消え、台帳は 6 件から 1 件になった。名指しを持つ id は 283 件から 288 件へ増えた（stash して測った）。
新設したテストは Go 5 件（部分試験を数えると 21 件）で、**製品コードは 1 行も変わっていない**。
**既存テストが行を区別できていた行は 1 つも無かった。** 5 行のうち 4 行は既存テストが拒否の効果までは
観測していたが、行が言っているのは効果ではなく**判定器へ渡す要求の形**（`AUTHZEN-FGA-EVALUATION`、
`AUTHZEN-FGA-ACTOR-CHAIN`）、**答えが出ない入力での退避先**（`AUTHZEN-FGA-FAIL-CLOSED`）、
**チェーン全体に対する積**（`RFC8693-FGA-ACTOR-AND`）であり、どれも観測されていなかった。
**観測の型が他の文書と違った。** Sourcing や WS-Federation の行は応答の形を宣言しているので HTTP の
応答を読めば足りるが、`AUTHZEN-FGA-EVALUATION` と `AUTHZEN-FGA-ACTOR-CHAIN` が言う「判定 context に
載せる」は応答にまったく現れない。`Authorizer` は差し替え可能なポートで `AUTHZEN=remote` では
`AuthZRequest` がそのまま PDP へ渡るので、この 2 行が固定しているのは要求そのものの形である。したがって
観測は、製品の組み立てへ記録用 `Authorizer` を差し、HTTP から入った要求が作った `AuthZRequest` を読む形に
なった。応答だけを読む観測では、関係を自分で判断して結果だけ渡す実装を 1 つも落とせない。
**欠陥は 1 件も見つからなかった。** 5 行はいずれも宣言した採用を満たしている。`AUTHZEN-FGA-SEARCH` は
`optional` だが実装があるので、`Adoption` の変更も要らない。
**故障注入で 1 つ、観測の分け方の限界が分かった。** `ListAccessibleResources` では走査を打ち切る判断と
結果に載せる `Truncated` が同じ変数なので、「上限で止まる」を崩す注入は必ず「打ち切りを示す」側も落とす。
2 つを分けて観測していることは、`ListedResources` の `Truncated` フィールドだけを `false` にする注入
（SEARCH/4）でしか確かめられなかった。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
