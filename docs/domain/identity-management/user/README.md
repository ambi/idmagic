# ユーザー

## 概要

この文書は、テナントの人間のプリンシパルである `User` を、管理者とフェデレーションが作成し、変更し、削除する機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `User` の作成、管理者による一覧と更新、必須操作の付与、無効化と再有効化、削除の予約と復元、完全削除、プロフィール属性の検証 |
| 行為者 | 管理者（`Administrator`）、フェデレーションのログインを処理する `Authentication` |
| コードの機能スライス | `backend/idmanagement/user` |
| 扱わないもの | 本人によるセルフサービスは[アカウントのセルフサービス](../account/README.md)、CSV による一括の作成と更新は[ユーザー CSV](../user-csv/README.md)、管理 API の認可は[管理 API の認可](../admin-access/README.md)が扱う |

状態を変える操作（無効化と再有効化、削除の予約、復元、完全削除）は、章[ユーザーのライフサイクルの操作](lifecycle.md)に置く。

## モデル

`User` は、テナントに属する Aggregate root である。
型として定義する中核の項目は、アイデンティティ、認証、RBAC に要るものだけに限る。

| 項目 | 内容 |
| --- | --- |
| `sub` | 不変の識別子。削除した後も保持し、再利用しない |
| `tenant_id` | 所属するテナント |
| `preferred_username` | ユーザー名 |
| `password_hash` | パスワード資格情報。フェデレーションで作った User では空である |
| `email`、`email_verified` | メールアドレスと、その所有を確認したか |
| `mfa_enrolled` | 多要素認証を登録したか |
| `roles` | 直接付与したロール。実効ロールは所属する `Group` のロールとの和集合である |
| `name`、`given_name`、`family_name` | 表示名 |
| `lifecycle` | 状態と、状態が変わった時刻（`status_changed_at`） |
| `attributes` | 中核にないプロフィール属性の疎な対応表 |

- **判断**：OIDC と SCIM の任意項目は 25 個ほどあり、ほとんどのテナントは使わない。すべての User の型と保存領域へ加えずに、値を設定したキーだけが領域を使う `attributes` に置く。

### 値オブジェクト

| 値 | 正規化 | 比較 | 一意性の範囲 |
| --- | --- | --- | --- |
| ユーザー名 | 前後の空白を除く | 大文字と小文字を区別する | 同じテナントの削除されていない User |
| メールアドレス | 前後の空白を除く。空になる値は設定しない | 大文字と小文字を区別しない | 経路ごとに異なる。フェデレーションによる作成は重複を拒否し、管理者による作成は拒否しない |

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

- **判断**：組み込みの属性とテナント定義の属性を、`UserAttributeDef` の一つの仕組みで扱う。管理者が設定するスキーマの形を一つに保つためである。
- **判断**：`pii` のデフォルトを `true` にするのは、テナントの利便より開示の上限を優先する安全側のデフォルトだからである。

## 状態遷移

### UserLifecycle

`Active` は通常の稼働、`Disabled` は元に戻せる停止である。
削除のデフォルトは予約であり、`PendingDeletion` に入ってから猶予期間の 30 日の間は `Active` へ戻せる。
`Deleted` は、Tombstone へ置き換えて匿名化した終端状態であり、復元できない。

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

| State | 無効化 | 再有効化 | 削除の予約 | 復元 | 完全削除 | 一覧の取得 |
|---|---|---|---|---|---|---|
| Active | → Disabled | 何もしない | → PendingDeletion | 拒否：409 not_pending_deletion | → Deleted | 何もしない |
| Disabled | 何もしない | → Active | → PendingDeletion | 拒否：409 not_pending_deletion | → Deleted | 何もしない |
| PendingDeletion | 拒否：409 user_pending_deletion | 拒否：409 user_pending_deletion | 何もしない | → Active（猶予期間内）<br>拒否：409 restore_grace_expired（猶予期間後） | → Deleted | 何もしない（猶予期間内）<br>→ Deleted（猶予期間後） |
| Deleted | 拒否：404 user_not_found | 拒否：404 user_not_found | 拒否：404 user_not_found | 拒否：404 user_not_found | 何もしない | 何もしない |

- **判断**：猶予期間の 30 日は、業界で一般的な 7 日から 30 日の範囲に合わせた。
- **判断**：削除を物理削除ではなく Tombstone で行う理由は、[User の削除を物理削除ではなく Tombstone で行う](../design/decisions.md#user-の削除を物理削除ではなく-tombstone-で行う)に書く。

## 操作

### 管理者による作成

#### REQ-IDMANAGEMENT-042 管理者による User の作成は、ユーザー名の一意性とパスワードポリシーを検証し、`Active` の User を作る

- 管理者が User を作成したとき、`Active` の User を作り、設定したパスワードをパスワードの履歴に加える。
- ユーザー名は、[値オブジェクト](#値オブジェクト)の定義で正規化して保存し、比較する。
- 同じテナントの削除されていない User と同じユーザー名を指定された場合は、409 と `username_conflict` で拒否し、User を作らない。
- 前後の空白を除くと空になるユーザー名を指定された場合は、拒否し、User を作らない。
- テナントのパスワードポリシーに違反するパスワードを指定された場合は、拒否し、User を作らない。
- ほかの User と同じメールアドレスを指定された場合は、拒否しない。
- **例**：EX-IDMANAGEMENT-042-01、EX-IDMANAGEMENT-042-02

### フェデレーションによる作成

#### REQ-IDMANAGEMENT-001 フェデレーションの JIT はパスワード資格情報を作らず有効な User を作成する

- `Authentication` が上流のトークンまたは Assertion とテナントの JIT ポリシーを検証した後に `ProvisionFederatedUser` を呼んだとき、対応付けたユーザー名と、任意の名前、メールアドレス、属性で、`password_hash` が空の `Active` の User を作り、`UserCreated` を発行する。
- テナントのリソース上限を超える場合、ユーザー名またはメールアドレスが衝突する場合、属性スキーマに違反する場合は、User を作らずにエラーを返す。
- **判断**：上流が認証の権威なので、ローカルの資格情報を設定しない。設定すると、上流で無効にした後もローカルのパスワードでサインインできる経路が残る。リソース上限、一意性、属性スキーマの検証は通常の作成と共通であり、この経路だけの近道はない。

#### REQ-IDMANAGEMENT-043 フェデレーションの JIT が作る User は、ロールを持たず、メールアドレスの重複を拒否する

- フェデレーションの JIT が作る User は、ロールが空であり、パスワードの履歴を持たない。
- メールアドレスは、[値オブジェクト](#値オブジェクト)の定義で正規化して保存する。
- フェデレーションの JIT が User を作ったとき、`UserCreated` の操作者を `identity-broker` とする。
- フェデレーションの JIT が User を作ったとき、動的グループの規則を評価しない。
- 同じテナントの削除されていない User と、大文字と小文字を区別せずに同じメールアドレスを指定された場合は、User を作らずに拒否する。
- **例**：EX-IDMANAGEMENT-043-01、EX-IDMANAGEMENT-043-03

### 一覧

#### REQ-IDMANAGEMENT-005 管理者のユーザー一覧は、正確な件数とカーソルで、重複も欠落もなくページを返す

- 管理者がユーザー一覧を取得したとき、絞り込みに一致する User の正確な総件数、総ページ数、現在のページ、ページサイズと、絞り込みに依存しない `total_users` を返す。
- ユーザー一覧の応答は、前後のページへ移る `first`、`prev`、`next`、`last` のカーソルを `Link` ヘッダーで返す。
- 一覧の途中で削除した User は、以後のページに含めない。カーソルで次のページを取得したとき、返却済みの User を重複して返さない。
- `query` または `status` を指定されたとき、テナント全体から条件に一致する User だけを返し、`pagination.total_items` を条件に一致する件数、`total_users` を削除済みを除く絞り込みに依存しない件数とする。
- 条件に一致する User がない場合は、空の一覧と、総項目数、総ページ数、現在のページ番号 `0 / 0 / 0` を返し、`Link` を一つも返さない。
- 正確な件数を取得できない場合は、0 件として成功させずに、リクエスト全体をサーバーエラーで失敗させる。
- 別テナントで発行されたカーソル、改ざんされたカーソル、発行時と `query` または `status` が異なるカーソルを受け取った場合は、InvalidRequestError で拒否する。
- `admin` ロールを持たない呼び出し元が一覧を要求した場合は、AccessDeniedError で拒否する。
- **例**：EX-IDMANAGEMENT-005-01、EX-IDMANAGEMENT-005-03

#### REQ-IDMANAGEMENT-044 管理者のユーザー一覧の取得は、猶予期間を過ぎた削除予約の User を先に完全削除する

- 管理者がユーザー一覧を取得したとき、同じテナントで `PendingDeletion` になってから猶予期間の 30 日を過ぎた User を完全削除してから一覧を作る。
- 猶予期間の終わりの時刻ちょうどの User と、`PendingDeletion` になった時刻を記録していない User は、完全削除しない。
- 一覧の取得が User を完全削除したとき、`UserDeleted` の操作者を `system`、理由を `auto_purge` とする。
- **例**：EX-IDMANAGEMENT-044-01、EX-IDMANAGEMENT-044-02

### 更新

#### REQ-IDMANAGEMENT-045 管理者による User の更新は、値が変わった項目だけを記録する

- 管理者が User を更新したとき、値が変わった項目だけを保存し、それだけを `changed_fields` に載せた `UserUpdated` を発行する。属性は、値が変わったキーごとに、キーの昇順で載せる。
- 管理者がどの項目の値も変えない更新を要求した場合は、成功を返し、`updated_at` を進めず、`UserUpdated` を発行しない。
- 管理者が `attributes` を指定したとき、属性の対応表の全体を置き換える。指定しなかったキーは消える。
- 管理者がメールアドレスだけを変えたとき、`email_verified` を変えない。
- **例**：EX-IDMANAGEMENT-045-01、EX-IDMANAGEMENT-045-03

### 必須操作の付与と解除

#### REQ-IDMANAGEMENT-047 必須操作の付与と解除は、すでにその状態なら何もしない

- 管理者がすでに付いている必須操作を付与した場合、または付いていない必須操作を解除した場合は、成功を返し、イベントを発行しない。
- 定義されていない必須操作を指定された場合は、422 と `invalid_required_action` で拒否する。

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| 認可 | 管理者の操作は、操作ごとの管理権限を要求する。規則は[管理 API の認可](../admin-access/README.md)が定める |
| テナント境界 | 操作の対象は、呼び出し元と同じテナントの User に限る |
| 資格情報の開示 | 管理 API の応答は、`password_hash` を含まない専用の型を通す |
| 停止した User | `Active` 以外の User は、新規のサインイン、既存のセッション、トークンの再発行、UserInfo のいずれも拒否する |

- **判断**：管理 API の応答に汎用のドメイン型を使わない。ドメイン型へ後から項目を加えたときに、その項目が管理 API から漏れる形にしないためである。
