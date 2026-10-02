# Feature: ユーザーのシナリオ

## 生成

### Rule: REQ-IDMANAGEMENT-001 フェデレーションの JIT はパスワード資格情報を作らず有効な User を作成する

Primary actor: `EndUser`

#### Example: EX-IDMANAGEMENT-001-01 通常経路

- Given Authentication Context が上流のトークンまたは Assertion と、テナントの JIT ポリシーを検証済みである
- When Authentication Context が ProvisionFederatedUser を呼ぶ
- Then 対応付けたユーザー名、任意の名前・メールアドレス・属性、テナントのリソース上限、一意性を検証する
- Then `password_hash` が空の `Active` な User を作成し、`UserCreated` を発行する

#### Example: EX-IDMANAGEMENT-001-02 ユーザー名またはメールアドレスが衝突する、リソース上限を超える、属性スキーマに違反する

- Given Authentication Context が上流のトークンまたは Assertion と、テナントの JIT ポリシーを検証済みである
- When Authentication Context が ProvisionFederatedUser を呼ぶ
- Then ユーザー名またはメールアドレスが衝突する、リソース上限を超える、属性スキーマに違反する
- Then User を作成せずエラーを返す

## 利用

### Rule: REQ-IDMANAGEMENT-005 管理者はユーザー一覧をページングしながら安定して閲覧できる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-005-01 通常経路

- Given 所属テナントに `limit` を超えるユーザーが存在する
- When 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
- Then レスポンスは、絞り込みに一致する正確な総件数、総ページ数、現在のページ、ページサイズと、絞り込みに依存しない `total_users` を返す
- Then レスポンスの `Link` ヘッダー（`rel="next"`）にコンパクトなカーソルが含まれる
- When 一覧の途中で他の管理者がユーザーを 1 件削除する
- Then 削除されたユーザーは一覧対象から除外される
- When 管理者が取得済みのカーソルで次ページを取得する
- Then 削除された行を除き、既に返却済みの行との重複なく残りのユーザーが返る
- Then レスポンスの `Link` ヘッダー（`rel="prev"`）に前ページのカーソルが含まれる
- When 管理者が `rel="prev"` のカーソルで前ページを取得する
- Then そのページのユーザーが正規の並び順で返る
- When 管理者が `rel="last"` の終端カーソルで最終ページを取得する
- Then 端数を含む最終ページが返る
- When 管理者が `rel="first"` のカーソルを含まない URL で先頭ページを取得する
- Then 正規の先頭ページが返る

#### Example: EX-IDMANAGEMENT-005-02 `query` または `status` を指定する

- Given 所属テナントに `limit` を超えるユーザーが存在する
- When 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
- But `query` または `status` を指定する
- Then `ListAdminUsers` はテナント全体から条件に一致する `User` だけを返す
- And `pagination.total_items` は条件一致件数、`total_users` は削除済みを除くフィルター非依存の件数を返す
- And `query` または `status` を変更した管理者はカーソルを破棄して先頭ページから取得する

#### Example: EX-IDMANAGEMENT-005-03 条件に一致する `User` が 0 件である

- Given 所属テナントに `limit` を超えるユーザーが存在する
- When 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
- But 条件に一致する `User` が 0 件である
- Then ユーザー一覧は空で、総項目数、総ページ数、現在のページ番号は `0 / 0 / 0` を返す
- And `first`、`prev`、`next`、`last` の `Link` は返さない

#### Example: EX-IDMANAGEMENT-005-04 正確な件数の取得に失敗する

- Given 所属テナントに `limit` を超えるユーザーが存在する
- When 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
- But 正確な件数の取得に失敗する
- Then 0 件として成功させず、リクエスト全体をサーバーエラーで失敗させる

#### Example: EX-IDMANAGEMENT-005-05 実行者が TenantAdministrator ロールを持たない

- Given 所属テナントに `limit` を超えるユーザーが存在する
- When 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
- But 実行者が TenantAdministrator ロールを持たない
- Then ListAdminUsers は AccessDeniedError で拒否される

#### Example: EX-IDMANAGEMENT-005-06 カーソルが別テナントで発行された、改ざんされた、または `query` / `status` が発行時と異なる

- Given 所属テナントに `limit` を超えるユーザーが存在する
- When 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
- Then レスポンスは、絞り込みに一致する正確な総件数、総ページ数、現在のページ、ページサイズと、絞り込みに依存しない `total_users` を返す
- Then レスポンスの `Link` ヘッダー（`rel="next"`）にコンパクトなカーソルが含まれる
- When 一覧の途中で他の管理者がユーザーを 1 件削除する
- Then 削除されたユーザーは一覧対象から除外される
- When 管理者が取得済みのカーソルで次ページを取得する
- But カーソルが別テナントで発行された、改ざんされた、または `query` / `status` が発行時と異なる
- Then `ListAdminUsers` は InvalidRequestError を返す
- And 管理者は先頭ページへ戻って取得し直す

## 失効と変更

### Rule: REQ-IDMANAGEMENT-010 管理者は無効化したユーザーを再有効化できる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-010-01 通常経路

- Given 管理者がユーザー "alice" を無効化している
- When 管理者がユーザー "alice" を再有効化する
- Then ユーザー `alice` のステータスは `Active` である
- Then "UserEnabled" が発行される

## 保持と削除

### Rule: REQ-IDMANAGEMENT-011 管理者はユーザーの削除を予約し、猶予期間内に復元できる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-011-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- And ユーザー "alice" は Active である
- When 管理者 "operator" がユーザー "alice" を削除する
- Then ユーザー `alice` のステータスは `PendingDeletion` である
- Then "UserSoftDeleted" が発行される
- When 管理者 "operator" がユーザー "alice" を復元する
- Then ユーザー `alice` のステータスは `Active` である
- Then "UserRestored" が発行される

### Rule: REQ-IDMANAGEMENT-012 削除を予約したユーザーはログインを拒否される (superseded by REQ-PLATFORM-002)

引き金は IdManagement の削除予約、観測はログインの拒否であり、どちらの Context も単独では起こせない。削除の予約と復元が到達経路の開閉と対応することを、REQ-PLATFORM-002 が保証として述べる。

### Rule: REQ-IDMANAGEMENT-013 管理者はユーザーを完全削除できる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-013-01 通常経路

- Given ユーザー "alice" は PendingDeletion である
- When 管理者がユーザー "alice" を完全削除する
- Then ユーザー `alice` のステータスは `Deleted` である
- Then "UserDeleted" が発行される

#### Example: EX-IDMANAGEMENT-013-02 対象が操作者自身であり、`admin` または `system_admin` を持つ

- Given ユーザー "alice" は PendingDeletion である
- When 管理者がユーザー "alice" を完全削除する
- But 対象が操作者自身であり、`admin` または `system_admin` を持つ
- Then 削除の予約、復元、完全削除のいずれも拒否される
- And エラー "self_delete_forbidden"
