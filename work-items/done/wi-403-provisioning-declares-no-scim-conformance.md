---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# Provisioning に `standards.md` を置き、外向き SCIM の準拠範囲を宣言する

`mise run spec-diff` は `no normative specification change against main` を返した。この道具が見ているのは規範シナリオ、状態遷移の行、TypeSpec の宣言の 3 つで、`standards.md` の行は対象外だからである。**本 work item の成果物は全体が規範の差分に映らなかった。** これ自体が発見であり、[[wi-442-spec-diff-does-not-see-standards-rows]] へ切り出した。
意味上の差分は次のとおり。`docs/domain/provisioning/standards.md` が新設され、外向き SCIM クライアントとしての準拠範囲が 12 行の規範として存在するようになった。RFC 7643 に 3 行（`RFC7643-OUT-CORE-RESOURCES`、`RFC7643-OUT-EXTERNAL-ID`、`RFC7643-OUT-SCHEMA-EXTENSIONS`）、RFC 7644 に 9 行（`RFC7644-OUT-RESOURCE-OPERATIONS`、`-PATCH`、`-DISCOVERY`、`-FILTERING`、`-ERROR-RESPONSE`、`-AUTHENTICATION`、`-BULK`、`-SORT`、`-ETAG`）。うち 4 行が `excluded` であり、送らないと決めたものが初めて記述された。`SPECIFICATION_FORMAT.md` §5 には、標準を提供する側ではなく消費する側で `Adoption` をどう読むかの段落が加わった。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
