# Feature: ユーザーの例

## Rule: REQ-IDMANAGEMENT-001 フェデレーションの JIT はパスワード資格情報を作らず有効な User を作成する

### Example: EX-IDMANAGEMENT-001-01 通常経路

- Given Authentication Context が上流のトークンまたは Assertion と、テナントの JIT ポリシーを検証済みである
- When Authentication Context が ProvisionFederatedUser を呼ぶ
- Then 対応付けたユーザー名、任意の名前・メールアドレス・属性、テナントのリソース上限、一意性を検証する
- Then `password_hash` が空の `Active` な User を作成し、`UserCreated` を発行する

### Example: EX-IDMANAGEMENT-001-02 ユーザー名またはメールアドレスが衝突する、リソース上限を超える、属性スキーマに違反する

- Given Authentication Context が上流のトークンまたは Assertion と、テナントの JIT ポリシーを検証済みである
- When Authentication Context が ProvisionFederatedUser を呼ぶ
- Then ユーザー名またはメールアドレスが衝突する、リソース上限を超える、属性スキーマに違反する
- Then User を作成せずエラーを返す

## Rule: REQ-IDMANAGEMENT-042 管理者による User の作成は、ユーザー名の一意性とパスワードポリシーを検証し、`Active` の User を作る

### Example: EX-IDMANAGEMENT-042-01 前後に空白を含むユーザー名

- When 管理者がユーザー名 " carol " の User を作成する
- Then User のユーザー名は "carol" である

### Example: EX-IDMANAGEMENT-042-02 大文字と小文字だけが異なるユーザー名

- Given テナントにユーザー名 "alice" の User がいる
- When 管理者がユーザー名 "Alice" の User を作成する
- Then 作成は成功し、二つの User が並ぶ

### Example: EX-IDMANAGEMENT-042-03 テナントのポリシーに違反するパスワード

- Given テナントのパスワードポリシーは 20 文字以上を求める
- When 管理者が 12 文字のパスワードで User を作成する
- Then 作成は拒否され、User は作られない

## Rule: REQ-IDMANAGEMENT-043 フェデレーションの JIT が作る User は、ロールを持たず、メールアドレスの重複を拒否する

### Example: EX-IDMANAGEMENT-043-01 大文字と小文字だけが異なるメールアドレス

- Given テナントにメールアドレス "alice@example.test" の User がいる
- When Authentication Context がメールアドレス "ALICE@example.test" で ProvisionFederatedUser を呼ぶ
- Then User は作られず、メールアドレスの衝突として拒否される

### Example: EX-IDMANAGEMENT-043-02 空白だけのメールアドレス

- When Authentication Context がメールアドレス "  " で ProvisionFederatedUser を呼ぶ
- Then User はメールアドレスなしで作られ、`UserCreated` の操作者は `identity-broker` である

### Example: EX-IDMANAGEMENT-043-03 規則に一致する属性を持つ JIT の User

- Given 有効な動的グループの規則は `user.department == "Engineering"` である
- When Authentication Context が `department` を "Engineering" とする User を ProvisionFederatedUser で作る
- Then 作成した User は、その動的グループに所属しない

## Rule: REQ-IDMANAGEMENT-005 管理者のユーザー一覧は、正確な件数とカーソルで、重複も欠落もなくページを返す

### Example: EX-IDMANAGEMENT-005-01 通常経路

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

### Example: EX-IDMANAGEMENT-005-02 `query` または `status` を指定する

- Given 所属テナントに `limit` を超えるユーザーが存在する
- When 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
- But `query` または `status` を指定する
- Then `ListAdminUsers` はテナント全体から条件に一致する `User` だけを返す
- And `pagination.total_items` は条件一致件数、`total_users` は削除済みを除くフィルター非依存の件数を返す
- And `query` または `status` を変更した管理者はカーソルを破棄して先頭ページから取得する

### Example: EX-IDMANAGEMENT-005-03 条件に一致する `User` が 0 件である

- Given 所属テナントに `limit` を超えるユーザーが存在する
- When 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
- But 条件に一致する `User` が 0 件である
- Then ユーザー一覧は空で、総項目数、総ページ数、現在のページ番号は `0 / 0 / 0` を返す
- And `first`、`prev`、`next`、`last` の `Link` は返さない

### Example: EX-IDMANAGEMENT-005-04 正確な件数の取得に失敗する

- Given 所属テナントに `limit` を超えるユーザーが存在する
- When 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
- But 正確な件数の取得に失敗する
- Then 0 件として成功させず、リクエスト全体をサーバーエラーで失敗させる

### Example: EX-IDMANAGEMENT-005-05 実行者が TenantAdministrator ロールを持たない

- Given 所属テナントに `limit` を超えるユーザーが存在する
- When 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
- But 実行者が TenantAdministrator ロールを持たない
- Then ListAdminUsers は AccessDeniedError で拒否される

### Example: EX-IDMANAGEMENT-005-06 カーソルが別テナントで発行された、改ざんされた、または `query` / `status` が発行時と異なる

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

## Rule: REQ-IDMANAGEMENT-044 管理者のユーザー一覧の取得は、猶予期間を過ぎた削除予約の User を先に完全削除する

### Example: EX-IDMANAGEMENT-044-01 猶予期間を過ぎた削除予約の User

- Given ユーザー "alice" は 31 日前に `PendingDeletion` になった
- When 管理者がユーザー一覧を取得する
- Then ユーザー "alice" は `Deleted` になり、操作者 `system`、理由 `auto_purge` の `UserDeleted` が発行される
- And 一覧にユーザー "alice" は含まれない

### Example: EX-IDMANAGEMENT-044-02 猶予期間ちょうどの削除予約の User

- Given ユーザー "alice" はちょうど 30 日前に `PendingDeletion` になった
- When 一覧の取得がその時刻に期限切れの User を完全削除する
- Then ユーザー "alice" は `PendingDeletion` のまま残る

## Rule: REQ-IDMANAGEMENT-010 User の無効化は `Disabled` に、再有効化は `Active` にして、それぞれイベントを発行する

### Example: EX-IDMANAGEMENT-010-01 通常経路

- Given 管理者がユーザー "alice" を無効化している
- When 管理者がユーザー "alice" を再有効化する
- Then ユーザー `alice` のステータスは `Active` である
- Then "UserEnabled" が発行される

### Example: EX-IDMANAGEMENT-010-02 記憶済みの端末を持つ User の無効化

- Given ユーザー "alice" には記憶済みの端末がある
- When 管理者がユーザー "alice" を無効化する
- Then その端末は失効している

## Rule: REQ-IDMANAGEMENT-045 管理者による User の更新は、値が変わった項目だけを記録する

### Example: EX-IDMANAGEMENT-045-01 一部の項目だけが変わる更新

- Given ユーザー "alice" の名前は "Alice"、属性 `department` は "Sales" である
- When 管理者が名前 "Alice" のまま、属性 `department` を "Engineering"、`title` を "Lead" にする
- Then `UserUpdated` の `changed_fields` は ["department", "title"] である

### Example: EX-IDMANAGEMENT-045-02 何も変わらない更新

- When 管理者がユーザー "alice" の現在と同じ名前を指定して更新する
- Then 更新は成功し、`updated_at` は変わらず、`UserUpdated` は発行されない

### Example: EX-IDMANAGEMENT-045-03 確認済みのメールアドレスを変える更新

- Given ユーザー "alice" のメールアドレスは確認済みである
- When 管理者がメールアドレスだけを "alice@new.example.test" へ変える
- Then `email_verified` は `true` のままである

## Rule: REQ-IDMANAGEMENT-046 User の無効化と再有効化は、すでにその状態なら何もせず、管理者自身の無効化と削除予約中の User を拒否する

### Example: EX-IDMANAGEMENT-046-01 無効化済みの User の無効化

- Given ユーザー "alice" は `Disabled` である
- When 管理者がユーザー "alice" を無効化する
- Then 操作は成功し、`status_changed_at` は変わらず、`UserDisabled` は発行されない

### Example: EX-IDMANAGEMENT-046-02 管理者が自分自身を無効化する

- Given 管理者 "operator" はロール `admin` を持つ
- When 管理者 "operator" が自分自身を無効化する
- Then 操作は `self_disable_forbidden` で拒否され、"operator" は `Active` のままである

### Example: EX-IDMANAGEMENT-046-04 削除予約中の User の無効化と再有効化

- Given ユーザー "alice" は `PendingDeletion` である
- When 管理者がユーザー "alice" を無効化する
- Then 操作は `user_pending_deletion` で拒否され、"alice" は `PendingDeletion` のままである
- When 管理者がユーザー "alice" を再有効化する
- Then 操作は `user_pending_deletion` で拒否され、"alice" は `PendingDeletion` のままである

## Rule: REQ-IDMANAGEMENT-047 必須操作の付与と解除は、すでにその状態なら何もしない

### Example: EX-IDMANAGEMENT-047-01 付与済みの必須操作をもう一度付与する

- Given ユーザー "alice" には必須操作 `update_password` が付いている
- When 管理者がユーザー "alice" に `update_password` を付与する
- Then 操作は成功し、`UserRequiredActionSet` は発行されない

### Example: EX-IDMANAGEMENT-047-02 定義されていない必須操作

- When 管理者がユーザー "alice" に必須操作 `reboot` を付与する
- Then 操作は `invalid_required_action` で拒否される

## Rule: REQ-IDMANAGEMENT-011 User の削除の予約は `PendingDeletion` にして `UserSoftDeleted` を発行し、予約済みの User には何もしない

### Example: EX-IDMANAGEMENT-011-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- And ユーザー "alice" は Active である
- When 管理者 "operator" がユーザー "alice" を削除する
- Then ユーザー `alice` のステータスは `PendingDeletion` である
- Then "UserSoftDeleted" が発行される

### Example: EX-IDMANAGEMENT-011-02 削除予約済みの User の削除の予約

- Given ユーザー "alice" は `PendingDeletion` である
- When 管理者がユーザー "alice" の削除を予約する
- Then 操作は成功し、`UserSoftDeleted` は再発行されない

## Rule: REQ-IDMANAGEMENT-013 特権を持つ管理者自身を対象にする削除の予約、復元、完全削除は拒否する

### Example: EX-IDMANAGEMENT-013-02 対象が操作者自身であり、`admin` または `system_admin` を持つ

- Given ユーザー "alice" は PendingDeletion である
- When 管理者がユーザー "alice" を完全削除する
- But 対象が操作者自身であり、`admin` または `system_admin` を持つ
- Then 削除の予約、復元、完全削除のいずれも拒否される
- And エラー "self_delete_forbidden"

### Example: EX-IDMANAGEMENT-013-03 削除予約済みの管理者自身の削除の予約

- Given 管理者 "operator" はロール `admin` を持ち、`PendingDeletion` である
- When 管理者 "operator" が自分自身の削除を予約する
- Then 操作は `self_delete_forbidden` で拒否される

## Rule: REQ-IDMANAGEMENT-048 User の削除の予約は記録を残し、理由とともに下流へ削除として通知する

### Example: EX-IDMANAGEMENT-048-02 下流への通知

- When 管理者が理由 "left the company" でユーザー "alice" の削除を予約する
- Then `UserSoftDeleted` の理由は "left the company" である
- And 下流のプロビジョニングへ User の削除として通知する

## Rule: REQ-IDMANAGEMENT-049 削除を予約した User の復元は、猶予期間の終わりの時刻まで `Active` に戻す

### Example: EX-IDMANAGEMENT-049-04 削除を予約した User の復元

- Given ユーザー "alice" は `PendingDeletion` である
- When 管理者 "operator" がユーザー "alice" を復元する
- Then ユーザー `alice` のステータスは `Active` である
- Then "UserRestored" が発行される

### Example: EX-IDMANAGEMENT-049-01 猶予期間の終わりちょうどの復元

- Given ユーザー "alice" はちょうど 30 日前に `PendingDeletion` になった
- When 管理者がユーザー "alice" を復元する
- Then ユーザー "alice" は `Active` になる

### Example: EX-IDMANAGEMENT-049-02 猶予期間を過ぎた復元

- Given ユーザー "alice" は 30 日と 1 秒前に `PendingDeletion` になった
- When 管理者がユーザー "alice" を復元する
- Then 復元は `restore_grace_expired` で拒否され、"alice" は `PendingDeletion` のままである

### Example: EX-IDMANAGEMENT-049-03 復元は下流へ通知しない

- Given ユーザー "alice" は `PendingDeletion` である
- When 管理者がユーザー "alice" を復元する
- Then 下流のプロビジョニングへの通知はない

## Rule: REQ-IDMANAGEMENT-050 User の完全削除は User を匿名化して `Deleted` にし、削除済みの User には何もしない

### Example: EX-IDMANAGEMENT-050-03 削除予約中の User の完全削除

- Given ユーザー "alice" は PendingDeletion である
- When 管理者がユーザー "alice" を完全削除する
- Then ユーザー `alice` のステータスは `Deleted` である
- Then "UserDeleted" が発行される

### Example: EX-IDMANAGEMENT-050-01 有効な User の完全削除

- Given ユーザー "alice" は `Active` で、ロールと属性とセッションを持つ
- When 管理者がユーザー "alice" を完全削除する
- Then "alice" のユーザー名は `deleted:<sub>` になり、ロールと属性は空になり、セッションは消える
- And テナントの User の使用量は一つ減る

### Example: EX-IDMANAGEMENT-050-02 削除済みの User の完全削除

- Given ユーザー "alice" は `Deleted` である
- When 管理者がユーザー "alice" を完全削除する
- Then 操作は成功し、`UserDeleted` は再発行されない
