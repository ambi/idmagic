# ユーザー

## 概要

この文書は、テナントの人間のプリンシパルである `User` を、管理者とフェデレーションが作成し、変更し、削除する機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `User` の作成、管理者による一覧と更新、必須操作の付与、無効化と再有効化、削除の予約と復元、完全削除、プロフィール属性の検証 |
| 行為者 | 管理者（`Administrator`）、フェデレーションのログインを処理する `Authentication` |
| コードの機能スライス | `backend/idmanagement/user` |
| 扱わないもの | 本人によるセルフサービスは[アカウントのセルフサービス](../account/README.md)、CSV による一括の作成と更新は[ユーザー CSV](../../bulk-transfer/user-csv/README.md)、管理 API の認可は[管理 API の認可](../../common/admin-access/README.md)が扱う |

ライフサイクルの操作（無効化と再有効化、削除の予約、復元、完全削除）は、章[ユーザーのライフサイクルの操作](lifecycle.md)に置く。

## モデル

`User` は、テナントに属する Aggregate root である。
型として定義する中核の項目は、アイデンティティ、認証、RBAC に要るものだけに限る。

| 項目 | 内容 |
| --- | --- |
| `sub` | 不変の識別子。削除した後も保持し、再利用しない |
| `tenant_id` | 所属するテナント |
| `preferred_username` | テナントの削除されていない User の間で一意なユーザー名 |
| `password_hash` | パスワード資格情報。フェデレーションで作った User では空である |
| `email`、`email_verified` | メールアドレスと、その所有を確認したか |
| `mfa_enrolled` | 多要素認証を登録したか |
| `roles` | 直接付与したロール。実効ロールは所属する `Group` のロールとの和集合である |
| `name`、`given_name`、`family_name` | 表示名 |
| `lifecycle` | 状態と、状態が変わった時刻（`status_changed_at`） |
| `attributes` | 中核にないプロフィール属性の疎な対応表 |

- **判断**：OIDC と SCIM の任意項目は 25 個ほどあり、ほとんどのテナントは使わない。すべての User の型と保存領域へ加えずに、値を設定したキーだけが領域を使う `attributes` に置く。

### 属性の定義

`attributes` の各キーは、`UserAttributeDef` の定義に従う。
実効的な定義は、すべてのテナントが共有する組み込みのカタログ（OIDC §5.1 の任意のクレームと、SCIM の `enterprise:User` に相当する組織の属性）と、テナントが定義する `TenantUserAttributeSchema` の和集合である。

| 定義の項目 | 内容 |
| --- | --- |
| `key` | snake_case で、先頭は英字 |
| `type` | `string`、`number`、`boolean`、`date`、`string_array` のいずれか |
| `required` | 値を必須とするか |
| `editable_by_user` | 本人のセルフサービスで変更できるか |
| `visibility` | `private`、`self_readable`、`admin_readable`、`claim_exposed` のいずれか。relying party へ開示するのは `claim_exposed` だけである |
| `claim_name`、`oidc_scope` | `visibility` が `claim_exposed` のときに使うクレーム名とスコープ |
| `pii` | 個人情報か。デフォルトは `true` であり、保存と監査では平文ではなく SHA-256 の要約を記録する |

- **判断**：組み込みの属性とテナント定義の属性を、二つの仕組みに分けずに `UserAttributeDef` の一つの仕組みで扱う。管理者が設定するスキーマの形を一つに保つためである。
- **判断**：`pii` のデフォルトを `true` にするのは、テナントの利便より開示の上限を優先する安全側のデフォルトだからである。

## 状態遷移

### UserLifecycle

`Active` は通常の稼働、`Disabled` は元に戻せる停止である。
削除のデフォルトは予約であり、`PendingDeletion` に入ってから猶予期間の間は `Active` へ戻せる。
`Deleted` は、Tombstone へ置き換えて匿名化した終端状態であり、復元できない。

`Deleted` へ入る経路は二つあり、どちらも `purge` の指定を要求する。

| 経路 | 条件 |
| --- | --- |
| 管理者の明示的な完全削除 | `purge=true` を指定すれば、どの状態からでも即時に入る |
| 猶予期間の経過 | `PendingDeletion` の User は、猶予期間の 30 日を過ぎると入る |

`purge` を指定しない削除の要求は、必ず `PendingDeletion` を経由する。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 通常稼働。認証を許可するのはこの状態だけである |
| Disabled | — | 復元可能な無効化 |
| PendingDeletion | — | 削除予約。猶予期間内は復元できる |
| Deleted | terminal | Tombstone 化して匿名化した。復元できない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | UserDisabled | — | Disabled |  |
| Disabled | UserEnabled | — | Active |  |
| Active | UserSoftDeleted | — | PendingDeletion |  |
| Disabled | UserSoftDeleted | — | PendingDeletion |  |
| PendingDeletion | UserRestored | — | Active |  |
| PendingDeletion | UserDeleted | input.purge == true \|\| duration_since(status_changed_at) >= duration('2592000s') | Deleted | UserDeleted |
| Active | UserDeleted | input.purge == true | Deleted | UserDeleted |
| Disabled | UserDeleted | input.purge == true | Deleted | UserDeleted |

- **判断**：猶予期間の 30 日は、業界で一般的な 7 日から 30 日の範囲に合わせた。
- **判断**：削除を物理削除ではなく Tombstone で行う理由は、[User の削除を物理削除ではなく Tombstone で行う](../../design/decisions.md#user-の削除を物理削除ではなく-tombstone-で行う)に書く。

## 操作

### 管理者による作成

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | ユーザー名、パスワード、任意の名前、メールアドレス、属性 |
| 成功時の作用 | `Active` の User を作り、パスワードをパスワードの履歴に加える |
| 拒否 | 空のユーザー名、同名の User（409 `username_conflict`）、パスワードポリシーへの違反。どの拒否も User を作らない |
| 冪等性 | なし。同じ要求を繰り返すと、二回目は同名の User として拒否する |

#### REQ-IDMANAGEMENT-042 管理者による User の作成は、ユーザー名の一意性とパスワードポリシーを検証し、`Active` の User を作る

- ユーザー名は前後の空白を除いて保存する。空白を除いて空になるユーザー名は拒否する。
- 同じテナントの削除されていない User と同じユーザー名は、409 と `username_conflict` で拒否する。
- ユーザー名の照合は大文字と小文字を区別する。`Alice` と `alice` は別の User として作成できる。
- パスワードは、テナントのパスワードポリシーで検証し、違反する作成は User を作らない。
- 作成した User は `Active` であり、設定したパスワードをパスワードの履歴に加える。
- メールアドレスは、ほかの User と同じ値でも拒否しない。
- **担保手段**：`usecases.CreateUser`
- **要判断**：ユーザー名の大文字と小文字を区別するため、表記だけが異なる二つの User が並び得る。Agent と Group の名前は区別しない。区別をやめるかを決める。
- **要判断**：フェデレーションの JIT とメールアドレスの変更はメールアドレスの重複を拒否するが、管理者の作成は拒否しない。作成でも拒否するかを決める。

### フェデレーションによる作成

| 項目 | 内容 |
| --- | --- |
| 行為者 | フェデレーションのログインを処理する `Authentication`（`UserCreated` の操作者は `identity-broker`） |
| 入力 | 上流で検証済みの、対応付けたユーザー名、任意の名前、メールアドレス、属性 |
| 成功時の作用 | パスワード資格情報もロールもない `Active` の User を作り、`UserCreated` を発行する。動的グループの規則は評価しない |
| 拒否 | リソース上限の超過、ユーザー名またはメールアドレスの衝突（メールアドレスは大文字と小文字を区別しない）、属性スキーマへの違反。どの拒否も User を作らない |
| 冪等性 | なし |

#### REQ-IDMANAGEMENT-001 フェデレーションの JIT はパスワード資格情報を作らず有効な User を作成する

- `Authentication` が上流のトークンまたは Assertion と、テナントの JIT ポリシーを検証した後に、対応付けたユーザー名と、任意の名前、メールアドレス、属性で User を作る。
- 作る前に、テナントのリソース上限、ユーザー名とメールアドレスの一意性、属性スキーマを検証する。
- 検証に違反した場合は、User を作らずにエラーを返す。
- 作る User は `Active` であり、`password_hash` は空である。
- 作ると `UserCreated` を発行する。
- **判断**：上流が認証の権威なので、ローカルの資格情報を設定しない。設定すると、上流で無効にした後もローカルのパスワードでサインインできる経路が残る。リソース上限、一意性、属性スキーマの検証は通常の作成と共通であり、この経路だけの近道はない。
- **担保手段**：`usecases.ProvisionFederatedUser`
- **例**：EX-IDMANAGEMENT-001-01、EX-IDMANAGEMENT-001-02

#### REQ-IDMANAGEMENT-043 フェデレーションの JIT が作る User は、ロールを持たず、メールアドレスの重複を拒否する

- メールアドレスは前後の空白を除き、空になるメールアドレスは設定しない。
- 同じテナントの削除されていない User と大文字と小文字を区別せずに同じメールアドレスは、User を作らずに拒否する。
- 作る User のロールは空であり、パスワードの履歴を持たない。
- `UserCreated` の操作者は `identity-broker` とする。
- 作成の時点では、動的グループの規則を評価しない。
- **担保手段**：`usecases.ProvisionFederatedUser`
- **要判断**：管理者の作成と CSV の適用は作成した User を動的グループの規則で評価するが、JIT は評価しない。JIT で作った User は、次に規則を再評価するまで動的グループに所属しない。作成時に評価するかを決める。

### 一覧

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | `limit`、カーソル、任意の `query` と `status` |
| 成功時の作用 | 猶予期間を過ぎた削除予約の User を先に完全削除してから、一覧と件数とページのリンクを返す |
| 拒否 | 管理者でない呼び出し元（AccessDeniedError）、別テナントのカーソル、改ざんしたカーソル、絞り込みが発行時と異なるカーソル（InvalidRequestError）、正確な件数を取得できない場合（サーバーエラー） |

#### REQ-IDMANAGEMENT-005 管理者のユーザー一覧は、正確な件数とカーソルで、重複も欠落もなくページを返す

- 一覧の応答は、絞り込みに一致する User の正確な総件数、総ページ数、現在のページ、ページサイズと、絞り込みに依存しない `total_users` を返す。
- 前後のページへは、`Link` ヘッダーの `first`、`prev`、`next`、`last` のカーソルで移る。
- 一覧の途中で削除した User は、以後のページに含めない。カーソルで次のページを取得しても、返却済みの User を重複して返さない。
- `query` または `status` を指定した一覧は、テナント全体から条件に一致する User だけを返す。`pagination.total_items` は条件に一致する件数、`total_users` は削除済みを除く絞り込みに依存しない件数である。
- 条件に一致する User がない一覧は、空の一覧と、総項目数、総ページ数、現在のページ番号 `0 / 0 / 0` を返し、`Link` を一つも返さない。
- 正確な件数を取得できない場合は、0 件として成功させずに、リクエスト全体をサーバーエラーで失敗させる。
- 別テナントで発行されたカーソル、改ざんされたカーソル、発行時と `query` または `status` が異なるカーソルは、InvalidRequestError で拒否する。
- `admin` ロールを持たない呼び出し元の一覧の取得は、AccessDeniedError で拒否する。
- **担保手段**：`handlers_http.HandleListAdminUsers`
- **例**：EX-IDMANAGEMENT-005-01、EX-IDMANAGEMENT-005-06

#### REQ-IDMANAGEMENT-044 管理者のユーザー一覧の取得は、猶予期間を過ぎた削除予約の User を先に完全削除する

- 一覧の取得は、同じテナントで `PendingDeletion` になってから猶予期間の 30 日を過ぎた User を完全削除してから一覧を作る。
- 猶予期間ちょうどの時刻の User は完全削除しない。
- `PendingDeletion` になった時刻を持たない User は完全削除しない。
- この完全削除の `UserDeleted` は、操作者を `system`、理由を `auto_purge` とする。
- **担保手段**：`usecases.PurgeExpiredSoftDeleted`、`handlers_http.HandleListAdminUsers`
- **要判断**：猶予期間を過ぎた User の完全削除は、管理者がユーザー一覧を取得するまで起きない。一覧を取得しないテナントでは、期限を過ぎた User の個人情報が残り続ける。定期的なジョブで消すかを決める。

### 更新

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 変える項目。`attributes` を指定すると、属性の対応表の全体を置き換える |
| 成功時の作用 | 値が変わった項目だけを保存し、`changed_fields` にそれだけを載せた `UserUpdated` を発行する |
| 冪等性 | 同じ値での更新は、`updated_at` を進めず、イベントを発行しない |

#### REQ-IDMANAGEMENT-045 管理者による User の更新は、値が変わった項目だけを記録する

- `UserUpdated` の `changed_fields` には、値が変わった項目だけを載せる。
- 属性は、値が変わったキーごとに、キーの昇順で `changed_fields` に載せる。
- どの項目の値も変わらない更新は成功を返し、`updated_at` を進めず、`UserUpdated` を発行しない。
- `attributes` を指定した更新は、属性の対応表の全体を置き換える。指定しなかったキーは消える。
- メールアドレスだけを変える更新は、`email_verified` を変えない。
- **担保手段**：`usecases.UpdateUser`
- **要判断**：管理者がメールアドレスを別のアドレスへ変えても、変更前に確認済みであれば `email_verified` は `true` のまま残る。新しいアドレスの所有を確かめていないのに確認済みとして扱う。アドレスを変えたときに `false` へ戻すかを決める。

### 必須操作の付与と解除

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 必須操作の名前 |
| 成功時の作用 | 必須操作を付ける、または外す |
| 拒否 | 定義されていない必須操作（422 `invalid_required_action`） |
| 冪等性 | すでにその状態なら、成功を返しイベントを発行しない |

#### REQ-IDMANAGEMENT-047 必須操作の付与と解除は、すでにその状態なら何もしない

- すでに付いている必須操作の付与と、付いていない必須操作の解除は成功を返し、イベントを発行しない。
- 定義されていない必須操作の付与と解除は、422 と `invalid_required_action` で拒否する。
- **担保手段**：`usecases.SetUserRequiredAction`、`usecases.ClearUserRequiredAction`

## エラー

次のエラーは、複数の操作で同じ意味を持つ。

| エラー | 意味 | 返す操作 |
| --- | --- | --- |
| `self_disable_forbidden` | `admin` または `system_admin` を持つ管理者が、自分自身を停止しようとした | 無効化 |
| `self_delete_forbidden` | `admin` または `system_admin` を持つ管理者が、自分自身を削除しようとした | 削除の予約、復元、完全削除 |
| `username_conflict` | 同じテナントの削除されていない User が、同じユーザー名を使っている | 作成 |

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| 認可 | 管理者の操作は、操作ごとの管理権限を要求する。規則は[管理 API の認可](../../common/admin-access/README.md)が定める |
| テナント境界 | 操作の対象は、呼び出し元と同じテナントの User に限る |
| 資格情報の開示 | 管理 API の応答は、`password_hash` を含まない専用の型を通す |
| 停止した User | `Active` 以外の User は、新規のサインイン、既存のセッション、トークンの再発行、UserInfo のいずれも拒否する |

- **判断**：管理 API の応答に汎用のドメイン型を使わない。ドメイン型へ後から項目を加えたときに、その項目が管理 API から漏れる形にしないためである。
