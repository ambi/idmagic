---
context: sourcing
updated_at: 2026-08-15
---

# Sourcing Specification

## Overview

外部の権威ある取り込み元から IdMagic へアイデンティティを取り込む責務を所有する。情報の正は外部にあり、IdMagic 内部のプリンシパルはその写しである。取り込み元との関連付け、外部の不変 ID との相関、取り込み処理とカーソル、外部の状態に従う削除・無効化の規則を定め、取り込み元ごとに機能単位を設ける。

この Context に入るかどうかは、通信の方向や実行時の形ではなく、永続的な関連付けを持つ外部権威が存在するかどうかで決まる。したがって、管理者による CSV インポート（IdManagement）、ログイン時のフェデレーション（Authentication）、下流システムとの台帳照合（Application または Provisioning）はいずれも対象外である。

現在の機能単位は `scim` だけである。SCIM 2.0 サーバーとして `/scim/v2/Users`、`/scim/v2/Groups` などを提供し、Okta、Google Cloud Identity、Entra ID などの外部 IdP からユーザーとグループの同期を受ける。Context のルートにはファサードと組み立てだけを置き、複数の取り込み元に実在する共通点が判明するまでは共通機構を作らない。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| IdentitySource | あるアイデンティティ集団について IdMagic の外部で権威を持つシステムと、その関係を表すテナント単位の関連付け。取り込み元の種類、資格情報と登録情報、有効・無効、属性の対応付け、削除・無効化を上流の権威にどこまで従わせるかをまとめる。取り込み元との関連付けを持たない経路は本 Context の対象外とする。 | source, アイデンティティソース, 取り込み元 |
| SourceCorrelation | 外部の取り込み元が持つ不変 ID と、IdMagic 内部のプリンシパル（User / Group）を結ぶ関連付け。取り込みを冪等にし、名前や属性が変わっても同一性を失わないための基準となる。`scim` 機能では ScimUserRef / ScimGroupRef が該当する。 | correlation link, external identity link, 相関 |
| Ingestion | IdentitySource が権威を持つ状態を IdMagic 内部のプリンシパルへ反映する処理。作成、更新、無効化、削除は IdManagement が公開する冪等なコマンドインターフェースを通じて適用し、記録元へ取り込み元固有の関心事を持ち込まない。 | ingest, 取り込み |
| IngestionRun | 1 回の取り込みを観測する単位。対象の取り込み元、開始と終了、適用件数、失敗、再開位置を持つ。実行は Jobs の永続ジョブに委ね、失敗後に再開できる粒度で観測する。`scim` 機能は外部 IdP からのリクエスト単位で適用するため IngestionRun を持たず、`directory` 以降の機能で実体を追加する。 | ingestion run, 取り込み実行 |
| SourceCursor | 前回の取り込みがどこまで進んだかを表す、取り込み元ごとの位置情報。差分取り込みと再同期の境界を決める。完全再同期は、カーソルを破棄して全件を読み直す操作として定義する。 | sync cursor, カーソル |
| SourceDrift | 上流の取り込み元が持つ正しい状態と IdMagic 内部の状態との乖離。取り込みの失敗、取り込み元での直接変更、相関情報の欠落などで生じ、検出と是正は取り込み元ごとの権威規則に従う。 | drift, 乖離 |
| ScimClient | テナント単位の Bearer トークンを提示して SCIM プロビジョニング API を呼び出す外部エージェント。`scim` 機能において IdentitySource を駆動する側である。 |  |

## Standards

### System for Cross-domain Identity Management Core Schema

RFC 7643 — https://www.rfc-editor.org/rfc/rfc7643.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7643-SERVICE-PROVIDER-CONFIG | required | MUST | ServiceProviderConfig は authenticationSchemes を含み Bearer トークン方式を広告する。 |
| RFC7643-CORE-RESOURCES | partial | MUST | User と Group リソースを SCIM Core Schema に従って表現する。 |
| RFC7643-ENTERPRISE-EXTENSION | partial | SHOULD | User リソースは Enterprise 拡張（`urn:ietf:params:scim:schemas:extension:enterprise:2.0:User`）の employeeNumber、department、manager に、Discovery と CRUD / PATCH で対応する。 |

### System for Cross-domain Identity Management Protocol

RFC 7644 — https://www.rfc-editor.org/rfc/rfc7644.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7644-RESOURCE-OPERATIONS | required | MUST | User と Group リソースに作成、参照、置換、削除の操作を提供する。 |
| RFC7644-PATCH | partial | SHOULD | User と Group リソースの部分更新を PATCH 操作で提供する。 |
| RFC7644-BEARER-AUTHORIZATION | required | MUST | SCIM プロトコルのエンドポイントは、テナント単位の Bearer トークンで認証・認可する。トークンには ApiTokens Context が発行する API アクセストークンを使用し、SCIM 操作には `scim:users:read` / `scim:users:write` / `scim:groups:read` / `scim:groups:write` のうち該当するスコープを要求する。Discovery エンドポイントは `scim:*` のいずれかで参照できる。 |
| RFC7644-ERROR-RESPONSE | required | MUST | プロトコル上の失敗は、HTTP ステータスと detail を持つ SCIM エラーレスポンスで返す。 |
| RFC7644-FILTERING | partial | SHOULD | コレクション一覧のエンドポイントは、`filter` クエリパラメーターによる絞り込みを提供する。 |

## Design

### Authorization boundary

SCIM のエンドポイントはユーザーのセッションではなく、`ApiTokens` が発行するテナント単位の API アクセストークンで認証し、操作ごとに `scim:users:*` と `scim:groups:*` のスコープを要求する。Discovery のエンドポイントは `scim:` で始まるいずれかのスコープで参照できる。

トークンはテナントを束縛する。リクエストしたレルムがトークンのテナントと一致しなければ、リソースが存在するかどうかを問わず拒否する。取り込み中に解決する参照 — Group のメンバー、Enterprise 拡張の `manager` — も同じテナント内に限り、別テナントの識別子を指す参照は `invalidValue` として拒否して保存しない。

### SCIM 2.0 inbound provisioning

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

### Design Decisions

- SCIM の多値プロフィールデータはプロトコル専用の副次的なストアに保持せず、IdMagic の正規の Aggregate へ投影する。ワイヤ表現を保存すると、正規のユーザー像が 2 つできてしまうからである。
- 未対応の複合属性と入れ子のグループは黙って捨てず、明示的に拒否する。気付かれないデータ損失と、認可上の意味を持たないグループのグラフを生まないためである。
- SCIM の `DELETE /Users/{id}` は即座の完全削除にせず、既存の論理削除の方針へ統合する。設定を誤った、あるいは誤動作した外部の同期が、回復不能な PII の喪失を引き起こさないようにするためである。

## Scenarios

### REQ-SOURCING-001: SCIM クライアントは Users と Groups のコレクションを検索できる
- ACTOR ScimClient
- GIVEN SCIM クライアントがテナントに対する有効なプロビジョニングトークンを持つ
- WHEN SCIM クライアントが `filter`、`startIndex`、`count` を指定して `/Users` と `/Groups` を GET する
  - ALT `filter` が許可されていない属性または演算子を使っている、あるいは構文が不正である → `invalidFilter` の SCIM プロトコルエラーで拒否される
  - ALT `startIndex` または `count` を整数として解釈できない、あるいは `count` が負数である → `invalidValue` の SCIM プロトコルエラーで拒否される
  - ALT プロビジョニングトークンのテナントがリクエスト先のテナントと一致しない → SCIM プロトコルエラーで拒否される
- THEN 各レスポンスは、`filter` 適用後の `totalResults`、該当ページの `Resources`、`itemsPerPage` を持つ SCIM ListResponse を返す

### REQ-SOURCING-002: 外部 IdP から SCIM でユーザーのライフサイクルを同期できる
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIM クライアントが CreateScimUser を呼び出す
  - ALT Bearer トークンが失効済み、期限切れ、または別テナントのトークンである → SCIM プロトコルエラーを返し、User を作成しない
- THEN 内部 User が作成され、ステータスが `Active` になる
- WHEN SCIM クライアントが PatchScimUser で `active=false` を指定する
  - ALT PATCH のリクエスト本文が RFC 7644 の操作要件を満たさない → `invalidValue` の ScimProtocolError を返し、User を変更しない
- THEN 内部 User が Disabled になる
- WHEN SCIM クライアントが DeleteScimUser を呼び出す
  - ALT 指定 ID が存在しない → 404 の ScimProtocolError を返す
- THEN 内部 User が PendingDeletion に遷移する

### REQ-SOURCING-003: 外部 IdP は SCIM リソースを PUT で完全に置換できる
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- GIVEN 対象 User が存在し、`name.givenName` と `active=false` を持つ
- WHEN SCIM クライアントが、`userName` だけを含むリクエスト本文で UpdateScimUser を呼び出す
  - ALT PUT のリクエスト本文に必須属性（User の `userName`、Group の `displayName`）がない → `invalidValue` の ScimProtocolError を返し、リソースを変更しない
  - ALT PUT のリクエスト本文に既存値と異なる `id` がある → 指定された `id` は無視し、サーバーが割り当てた既存の ID を維持する
- THEN `name.givenName` は空文字に、`active` は `true` にリセットされる
- THEN レスポンスは `id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location` を含む

### REQ-SOURCING-004: 外部 IdP による未対応の PATCH パスや読み取り専用属性への書き込みを拒否する
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- GIVEN 対象 User または Group が存在する
- WHEN SCIM クライアントが RFC7644-PATCH で対応するパス（User: `userName` / `name` / `active` / `emails`、Group: `displayName` / `members`）に対する replace 操作を、PatchScimUser または PatchScimGroup で送る
  - ALT `path` が対応属性の許可リスト外である、または存在しない属性を指す → `invalidPath` の ScimProtocolError を返し、リソースを変更しない
  - ALT `path` が読み取り専用の `id`、`meta`、`schemas` のいずれかを指す → `mutability` の ScimProtocolError を返し、リソースを変更しない
  - ALT `op` が `add` / `replace` / `remove` のいずれでもない → `invalidValue` の ScimProtocolError を返し、リソースを変更しない
- THEN 対象属性だけが更新され、他の属性は変化しない

### REQ-SOURCING-005: 外部 IdP から SCIM でグループとメンバーシップを同期できる
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIM クライアントが CreateScimGroup を呼び出す
- THEN グループが作成される
- WHEN SCIM クライアントが PatchScimGroup でメンバー追加を指定する
  - ALT 追加対象の User が別テナントに属する → ScimProtocolError を返し、メンバーシップを作成しない
  - ALT メンバーの `type` が `User` 以外である → `invalidValue` の ScimProtocolError を返し、Group を変更しない
- THEN GroupMembership が同期され User の有効ロールが更新される
- THEN Group レスポンスの各メンバーは `type=User` を持つ

### REQ-SOURCING-006: SCIM の複数メールアドレスを正規メールアドレスへ決定的に投影する
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で複数の `emails` を送る
  - ALT `primary=true` の要素が複数ある、要素がオブジェクトでない、`value` が空または文字列でない、`type` が文字列でない、`primary` が真偽値でない → `invalidValue` の ScimProtocolError を返し、User を変更しない
  - ALT `primary` がない → `type` が大文字と小文字を区別せず `work` と一致する最初の要素を選ぶ
  - ALT `primary` も `work` もない → 通信上で最初の要素を選ぶ
- THEN `primary=true` の要素が 1 件あれば、その `value` だけを `User.email` に保存する
- THEN User レスポンスは、保存済みメールアドレスがある場合だけ、`type=work`、`primary=true` の要素を 1 つ返す
- WHEN SCIM クライアントが CreateScimUser または UpdateScimUser で `phoneNumbers` または `addresses` を送る
  - THEN `invalidValue` の ScimProtocolError を返し、User を変更しない
- WHEN SCIM クライアントが PatchScimUser の `path` に `phoneNumbers` または `addresses` を指定する
  - THEN `invalidPath` の ScimProtocolError を返し、User を変更しない
- WHEN SCIM クライアントが GetScimSchemas を呼び出す
- THEN サーバーは、対応する 1 件のメールアドレスへの投影と User メンバーだけを広告し、`phoneNumbers`、`addresses`、Group メンバーは広告しない

### REQ-SOURCING-007: SCIM Enterprise 拡張の User 組織属性に対応する
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で Enterprise 拡張の `employeeNumber`、`department`、`manager` を送る
  - ALT `employeeNumber` または `department` が文字列でない → `invalidValue` の ScimProtocolError を返し、User を変更しない
  - ALT `manager` の値（`value` オブジェクトまたは文字列）が空である、あるいは同じテナント内に存在しない SCIM User を指す → `invalidValue` の ScimProtocolError を返し、User を変更しない
- THEN 対応する値は `idmanagement.User.Attributes` の `employee_number`、`department`、`manager_sub` に永続化される（`manager` には解決した内部の `User.sub` を保存する）
- THEN User レスポンスは、いずれかの Enterprise 拡張属性を保持する場合だけ `schemas` に Enterprise 拡張 URN を含み、対応する属性オブジェクトを返す
- WHEN SCIM クライアントが GetScimSchemas または GetScimResourceTypes を呼び出す
- THEN サーバーは Enterprise 拡張スキーマを `/Schemas` に、`/ResourceTypes` の User エントリーの `schemaExtensions` に広告する
