---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# 外向き SCIM が送るリソース本文に `schemas` を入れる

`mise run spec-diff` は `no normative specification change against main` を返す。`REQ-` シナリオは動いておらず、変わったのは `RFC7643-OUT-CORE-RESOURCES` の `Statement` と、下流へ送る本文である。作成 (POST) と置換 (PUT) のリソース表現が `schemas` を持つようになった。User は `urn:ietf:params:scim:schemas:core:2.0:User`、Group は `...:Group` の 1 要素で、属性対応付けの外側 (`withSchemas`) で載せている。PATCH の本文は従来どおり `PatchOp` の URN を持ち、その `Operations[].value` は部分断片なので `schemas` を持たない。行の `Adoption` は `partial` のままとした。`schemas` の欠落は解けたが、対応付けで書ける範囲 (拡張スキーマを送らないこと) の制約は変わっていないためである。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
