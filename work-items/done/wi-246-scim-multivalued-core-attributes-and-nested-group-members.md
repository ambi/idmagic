---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# SCIM core 属性の対応範囲を明示し、emails を単一 primary email へ決定的に投影する

`REQ-SOURCING-006` を追加し、SCIM emails を全 element 検証後に primary、work、wire order の順で
単一 `User.Email` へ投影する current design と HTTP contract を specification-first で確定した。
POST/PUT/PATCH は同じ domain pure function を利用し、複数 primary、不正 value/type/primary を
`invalidValue` で mutation 前に拒否する。GET/LIST/各 mutation response は email がある場合だけ
単一 `{type: "work", primary: true}` entry を返す。`phoneNumbers` / `addresses` は POST/PUT で
`invalidValue`、PATCH で `invalidPath` とし、silent discard を廃止した。Group member は type 省略または
User のみに限定し、response と schema discovery は User を canonical type / reference type として返す。
User/Group の canonical model、DB schema、認可計算は変更せず、SCIM 専用 profile table、shadow document、
Group-in-Group graph は追加していない。multi-valued email/phone/address の lossless storage、phone/address の
canonical attribute mapping、nested group の保存・展開・認可継承、enterprise/custom schema、bracket
filter/PATCH path は Out of Scope のままである。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
