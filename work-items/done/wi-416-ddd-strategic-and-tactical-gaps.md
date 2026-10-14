---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# サブドメインの分類と Aggregate の境界を仕様として持つ

`mise run spec-diff` は `no normative specification change against main` を返す。規範シナリオ、規範 ID、TypeSpec シンボルはいずれも増減していない。変わったのは、それらを読むための語彙と規則である。`docs/domain/glossary.md` が `Aggregate` と `Subdomain` を定義し、`docs/design-rules.md` が Aggregate の境界の引き方、トランザクションとの対応、Repository は Aggregate root 単位という規則、`Repository` と `Store` の接尾辞が種類を区別しないという事実、そしてサブドメインの区分が左右する 3 つと左右しない 1 つ（検証の強度）を持つ。`docs/domain/structure.md` が Anti-Corruption Layer の配置規約を持ち、`docs/README.md` の索引表が全 21 Context の区分を宣言する。各 Context の `decisions.md` はその区分の理由を、`Core` 5 件の `glossary.md` は Aggregate root の名前を持つ。`SPECIFICATION_FORMAT.md` は索引表が区分を宣言することを *(checked)* の規則として記載する。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
