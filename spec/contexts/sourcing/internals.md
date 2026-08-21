# Sourcing Internals

## SCIM 2.0 inbound provisioning

各テナントは `/realms/{realm_id}/scim/v2` を持ち、テナントを特定できるテナントごとの Bearer トークンで認証する。全体で 1 つのトークンを共有する方式は、テナント間の分離を破るため採用しない。サーバーは `/Users` と `/Groups`（GET、POST、GET/{id}、PUT/{id}、PATCH/{id}、DELETE/{id}）、`/ServiceProviderConfig`、`/ResourceTypes`、`/Schemas` を実装する。

属性は User と Group の Aggregate へ直接対応付ける。

| SCIM | IdMagic |
| --- | --- |
| `Users.id` | `User.sub` |
| `Users.userName` | `User.preferred_username` |
| `Users.name.formatted` / `displayName` | `User.name` |
| `Users.emails` | `User.email`（primary、work、通信上の順序の優先順位で 1 件へ投影） |
| `Users.active` | `UserLifecycle.status == Active` |
| `Groups.id` | `Group.id` |
| `Groups.displayName` | `Group.name` |
| `Groups.members` | `GroupMember` のメンバーシップ |
| enterprise extension `employeeNumber` | `User.Attributes["employee_number"]` |
| enterprise extension `department` | `User.Attributes["department"]` |
| enterprise extension `manager.value` | `User.Attributes["manager_sub"]`(内部 `User.sub`。SCIM id 経由で解決) |

SCIM の `emails` は複数値を持てるワイヤ表現だが、IdMagic の正規 User は認証と通知に使う単一のメールアドレスだけを持つ。User の変更では、すべてのメールアドレス要素を検証した後、`primary=true`、大文字と小文字を区別しない `type=work`、ワイヤ上の先頭という優先順位で 1 件へ射影する。`primary` が複数ある場合や不正な要素がある場合は、変更全体を拒否する。User のレスポンスは、正規メールアドレスがある場合だけ、単一の `type=work, primary=true` エントリーへ正規化して返す。入力配列全体を損失なく往復できることは保証しない。

スキーマの Discovery では実装済みの範囲だけを広告する。`phoneNumbers` と `addresses` は保存も広告もせず、POST / PUT の本文では `invalidValue`、PATCH のパスでは `invalidPath` として拒否し、気付かないままデータが失われることを防ぐ。Group のメンバーシップは直接所属する User だけを表す。メンバー種別は省略するか `User` だけを受け付け、レスポンスでは常に `type=User` を返す。認可上の意味を持たない Group 間の入れ子構造は保存しない。

Enterprise 拡張は `employeeNumber`、`department`、`manager` の 3 属性だけに対応する。`costCenter`、`division`、`organization` は対象外とする。値は `idmanagement.User.Attributes` の既存の組み込みキー（`employee_number`、`department`、`manager_sub`）を再利用し、idmanagement 側のモデル変更を不要にする。`manager` は SCIM ID への参照として受け取り、`ScimUserRef` を介して同一テナント内の `User.sub` へ解決する。テナントをまたぐ参照や存在しない SCIM ID は `invalidValue` として拒否し、テナント境界を越えた参照を保存しない。レスポンスの `schemas` には、いずれかの Enterprise 拡張属性を保持する場合だけ Enterprise 拡張の URN を含め、対応する属性オブジェクトを返す。PATCH の `path` は、`employeeNumber` などの単純名と、Enterprise 拡張 URN で修飾した完全パスの両方を受け付ける。

PATCH や PUT で `active` を `false` にすると `User.lifecycle.status` は `Disabled` へ遷移し、`true` に戻すと `Active` へ戻る。`DELETE /Users/{id}` は完全な削除を行わず、プラットフォームの他の部分と同じ論理削除 (`PendingDeletion`、30 日の猶予、その後に匿名化を伴う連鎖的な完全削除) を行う。これにより設定を誤った、あるいは誤動作した外部の同期が、回復不能な PII の喪失を引き起こすことはない。SCIM の削除を既存の論理削除の方針に統合するものであり、迂回するものではない。`DELETE /Groups/{id}` は即時かつ完全である。group は PII を持たないからである。
