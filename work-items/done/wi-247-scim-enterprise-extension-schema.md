---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# SCIM enterprise extension schema (urn:...:extension:enterprise:2.0:User) に対応する

SCIM enterprise extension (`urn:ietf:params:scim:schemas:extension:enterprise:2.0:User`) の
`employeeNumber`・`department`・`manager` サブセットに discovery (`/Schemas`、
`/ResourceTypes` の `schemaExtensions`) と CRUD/PATCH (`CreateScimUser`・
`UpdateScimUser`・`PatchScimUser`) で対応した。当初 Plan のとおり
`idmanagement.User.Attributes` の既存 builtin キー (`employee_number`・`department`・
`manager_sub`) をそのまま再利用し、idmanagement 側のモデル変更は不要だった。
`manager` は SCIM id 参照として受け取り、`ScimRepo.FindUserRefByScimID` (tenant-scoped)
で内部 `User.sub` へ validate-first で解決する。存在しない・別 tenant の SCIM id は
`invalidValue` にして拒否し、そのユーザーは作成・更新されない(Risk Notes の tenant
境界懸念に対応)。PATCH の `path` は bare 名 (`employeeNumber` など) と enterprise
extension URN で修飾した完全パスの両方を受け付け、`manager` の PATCH value は
`{value: "..."}` オブジェクトと素の文字列(Entra ID が送る形)の両方を受け付ける。
response の `schemas` と拡張オブジェクトは、いずれかの enterprise extension 属性を
保持する場合だけ付与する(値が無ければ何も広告しない)。
`docs/domain/sourcing/SPECIFICATION.md` に Standards 行
(`RFC7643-ENTERPRISE-EXTENSION`, partial)、Design 節への属性マッピングと
tenant-scoped manager 解決の説明、新規 scenario `REQ-SOURCING-007` を追加した。
`main.tsp` の該当 op docstring も同期した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
