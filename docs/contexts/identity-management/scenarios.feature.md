# Feature: IdManagement Scenarios

## Rule: REQ-IDMANAGEMENT-001 フェデレーションの JIT はパスワード資格情報を作らず有効な User を作成する

Primary actor: `EndUser`

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

## Rule: REQ-IDMANAGEMENT-002 API トークンの発行者は account スコープで自身の情報だけを操作できる

Primary actor: `SelfApiClient`

### Example: EX-IDMANAGEMENT-002-01 通常経路

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントが概要、プロフィール、データエクスポート、またはプライマリメールアドレスの変更申請を要求する
- Then `account:read` スコープは、自身の概要、プロフィール、データエクスポートの参照だけを許可する
- Then `account:write` スコープは、自身のプロフィールとプライマリメールアドレスの変更申請だけを許可する

### Example: EX-IDMANAGEMENT-002-02 `account:read` だけで変更操作を要求する

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントが概要、プロフィール、データエクスポート、またはプライマリメールアドレスの変更申請を要求する
- But `account:read` だけで変更操作を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-IDMANAGEMENT-002-03 トークンのテナントまたは `user_id` が操作対象と一致しない

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントが概要、プロフィール、データエクスポート、またはプライマリメールアドレスの変更申請を要求する
- But トークンのテナントまたは `user_id` が操作対象と一致しない
- Then 操作は AccessDeniedError で拒否される

## Rule: REQ-IDMANAGEMENT-003 メールアドレス確認画面は未認証でも CSRF 境界を確立できる

Primary actor: `EndUser`

### Example: EX-IDMANAGEMENT-003-01 通常経路

- When EndUser がメールアドレス確認の文脈を取得する
- Then レスポンスに CSRF トークンと SameSite 属性を持つ Cookie が含まれる
- When EndUser がその CSRF トークンと Cookie を添えてメールアドレスの確認を送信する
- Then メールアドレスの確認が受理される

### Example: EX-IDMANAGEMENT-003-02 CSRF トークンと Cookie が一致しない

- When EndUser がメールアドレス確認の文脈を取得する
- Then レスポンスに CSRF トークンと SameSite 属性を持つ Cookie が含まれる
- When EndUser がその CSRF トークンと Cookie を添えてメールアドレスの確認を送信する
- But CSRF トークンと Cookie が一致しない
- Then 確認は InvalidRequestError で拒否される

## Rule: REQ-IDMANAGEMENT-004 管理者は CSV を検証して有効な行だけをインポートできる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-004-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 有効な行は作成または更新され、無効な行は `rejected` として残る。各行のプロフィール、ロール、必須操作、カスタム属性は不可分に保存される

### Example: EX-IDMANAGEMENT-004-02 CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- But CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える
- Then インポートの投入は拒否される
- And エラー "csv_too_large" / "too_many_rows" / "field_too_large"

### Example: EX-IDMANAGEMENT-004-03 CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- But CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる
- Then インポートの投入は拒否される
- And エラー "invalid_header"

### Example: EX-IDMANAGEMENT-004-04 行の `id` と `preferred_username` が別の `User` を示す、識別子がない、同じ対象または同じ最終ユーザー名を複数行が示す

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then 行の `id` と `preferred_username` が別の `User` を示す、識別子がない、同じ対象または同じ最終ユーザー名を複数行が示す
- Then 対象行は `rejected` となり、安定したエラーコードを返す

### Example: EX-IDMANAGEMENT-004-05 プレビュージョブが存在しない、`queued` または `failed` である、別テナントに属する、保存済みのペイロードとダイジェストが一致しない

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- But プレビュージョブが存在しない、`queued` または `failed` である、別テナントに属する、保存済みのペイロードとダイジェストが一致しない
- Then 適用は `User` を変更せず `InvalidRequestError` または `AccessDeniedError` で拒否される

### Example: EX-IDMANAGEMENT-004-06 プレビュー後に対象 `User` の状態が別の操作で変更されている

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then プレビュー後に対象 `User` の状態が別の操作で変更されている
- Then 適用は古いプレビュー計画を実行せず、現在状態から `updated`、`unchanged`、`rejected` を再判定する

### Example: EX-IDMANAGEMENT-004-07 対象 `User` が外部の取り込み元に管理されている

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 対象 `User` が外部の取り込み元に管理されている
- Then 対象行は安定したエラーコード `source_managed` で `rejected` となり、`User` は変更されない

### Example: EX-IDMANAGEMENT-004-08 1 行の検証、保存、監査処理が途中で失敗する

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 1 行の検証、保存、監査処理が途中で失敗する
- Then その行のプロフィール、ロール、必須操作、カスタム属性は一部も保存されず、他の有効な行は適用を続ける

## Rule: REQ-IDMANAGEMENT-005 管理者はユーザー一覧をページングしながら安定して閲覧できる

Primary actor: `TenantAdministrator`

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

## Rule: REQ-IDMANAGEMENT-006 管理者はユーザー一覧を CSV に安全にエクスポートできる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-006-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- And 一覧には自テナントのユーザーが存在する
- When 管理者が列 [`preferred_username`, `email`] と `status` フィルターを指定して `/users/exports` へエクスポートを開始する
- Then エクスポートは 202 とエクスポート ID を返し、ジョブは `queued` である
- When 終端前に管理者がエクスポートを取り消す
- Then ステータスは `canceled` となり、`DataExportCanceled` が発行される
- Then `worker` プロセスが生成を開始し、`DataExportStarted` が発行される
- Then 生成が完了してステータスは `succeeded`、`downloadable` は `true` となり、`total_rows` と `byte_size` が記録される
- Then DataExportSucceeded が発行される
- When 管理者がファイルをダウンロードする
- Then 選択した機械可読キーと一致するヘッダーを持つ RFC 4180 CSV が、`Content-Disposition: attachment` で返る
- Then DataExportDownloaded が発行される

### Example: EX-IDMANAGEMENT-006-02 選択列に `User` の許可一覧にないキー（例: `password_hash`）が含まれる

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- And 一覧には自テナントのユーザーが存在する
- When 管理者が列 [`preferred_username`, `email`] と `status` フィルターを指定して `/users/exports` へエクスポートを開始する
- But 選択列に `User` の許可一覧にないキー（例: `password_hash`）が含まれる
- Then エクスポート開始は `InvalidRequestError` で拒否される
- And エラー `invalid_columns`

### Example: EX-IDMANAGEMENT-006-03 生成が失敗する

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- And 一覧には自テナントのユーザーが存在する
- When 管理者が列 [`preferred_username`, `email`] と `status` フィルターを指定して `/users/exports` へエクスポートを開始する
- Then エクスポートは 202 とエクスポート ID を返し、ジョブは `queued` である
- When 終端前に管理者がエクスポートを取り消す
- Then ステータスは `canceled` となり、`DataExportCanceled` が発行される
- Then `worker` プロセスが生成を開始し、`DataExportStarted` が発行される
- Then 生成が失敗する
- Then ステータスは `failed`、`downloadable` は `false` となり、`error_code` が記録される
- And `DataExportFailed` が発行される
- And 不完全なファイルはダウンロードできない

### Example: EX-IDMANAGEMENT-006-04 セル値が \"=\", \"+\", \"-\", \"@\", タブ, CR, LF のいずれかで始まる

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- And 一覧には自テナントのユーザーが存在する
- When 管理者が列 [`preferred_username`, `email`] と `status` フィルターを指定して `/users/exports` へエクスポートを開始する
- Then エクスポートは 202 とエクスポート ID を返し、ジョブは `queued` である
- When 終端前に管理者がエクスポートを取り消す
- Then ステータスは `canceled` となり、`DataExportCanceled` が発行される
- Then `worker` プロセスが生成を開始し、`DataExportStarted` が発行される
- Then 生成が完了してステータスは `succeeded`、`downloadable` は `true` となり、`total_rows` と `byte_size` が記録される
- Then DataExportSucceeded が発行される
- When 管理者がファイルをダウンロードする
- But セル値が \"=\", \"+\", \"-\", \"@\", タブ, CR, LF のいずれかで始まる
- Then 数式の注入を避ける可逆な接頭辞を付けて出力し、インポート側の変換器は規定どおり接頭辞 1 文字だけを取り除く

### Example: EX-IDMANAGEMENT-006-05 保持期限を経過している

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- And 一覧には自テナントのユーザーが存在する
- When 管理者が列 [`preferred_username`, `email`] と `status` フィルターを指定して `/users/exports` へエクスポートを開始する
- Then エクスポートは 202 とエクスポート ID を返し、ジョブは `queued` である
- When 終端前に管理者がエクスポートを取り消す
- Then ステータスは `canceled` となり、`DataExportCanceled` が発行される
- Then `worker` プロセスが生成を開始し、`DataExportStarted` が発行される
- Then 生成が完了してステータスは `succeeded`、`downloadable` は `true` となり、`total_rows` と `byte_size` が記録される
- Then DataExportSucceeded が発行される
- When 管理者がファイルをダウンロードする
- But 保持期限を経過している
- Then ステータスは `expired`、`downloadable` は `false` となる
- And ファイル本体は完全削除され、ダウンロードは `InvalidRequestError` で拒否される

### Example: EX-IDMANAGEMENT-006-06 `User` エクスポートの ID を `/groups/exports` または別テナントで指定する

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- And 一覧には自テナントのユーザーが存在する
- When 管理者が列 [`preferred_username`, `email`] と `status` フィルターを指定して `/users/exports` へエクスポートを開始する
- Then エクスポートは 202 とエクスポート ID を返し、ジョブは `queued` である
- When 終端前に管理者がエクスポートを取り消す
- Then ステータスは `canceled` となり、`DataExportCanceled` が発行される
- Then `worker` プロセスが生成を開始し、`DataExportStarted` が発行される
- Then 生成が完了してステータスは `succeeded`、`downloadable` は `true` となり、`total_rows` と `byte_size` が記録される
- Then DataExportSucceeded が発行される
- When 管理者がファイルをダウンロードする
- But `User` エクスポートの ID を `/groups/exports` または別テナントで指定する
- Then 種類とテナントの境界により、取得、ダウンロード、取り消しは `AccessDeniedError` または `InvalidRequestError` で拒否される

## Rule: REQ-IDMANAGEMENT-007 管理者はエクスポートしたユーザー CSV を安全に再適用できる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-007-01 通常経路

- Given 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`User` は変更されない
- When 管理者が 1 行の `email` と `custom:department` だけを編集して再びプレビューする
- Then 変更行だけが updated、残りは unchanged と計画される
- When 管理者が成功済みプレビュージョブの ID を指定して適用する
- Then 指定した書き込み可能な列だけが更新され、指定しなかった列は維持される

### Example: EX-IDMANAGEMENT-007-02 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む

- Given 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
- But 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む
- Then 可逆な数式安全変換と RFC 4180 の引用により `decode(encode(value))` は元の値と一致する

### Example: EX-IDMANAGEMENT-007-03 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える

- Given 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
- But 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える
- Then `User` エクスポートは `csv_transfer_limit_exceeded` で失敗し、再インポートできない成功済み成果物を作らない
- And 管理者はフィルターまたは列を絞って複数の成果物に分割できる

### Example: EX-IDMANAGEMENT-007-04 `status`、`mfa_enrolled`、`created_at`、`updated_at`、`id` の値だけを編集する

- Given 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`User` は変更されない
- When 管理者が 1 行の `email` と `custom:department` だけを編集して再びプレビューする
- But `status`、`mfa_enrolled`、`created_at`、`updated_at`、`id` の値だけを編集する
- Then 読み取り専用列は受理したうえで無視し、書き込み可能な列に差分がなければ `unchanged` とする

### Example: EX-IDMANAGEMENT-007-05 カスタム属性の型、真偽値、数値、日付、必須のカスタム属性、`required_actions` のいずれかが不正である

- Given 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`User` は変更されない
- When 管理者が 1 行の `email` と `custom:department` だけを編集して再びプレビューする
- But カスタム属性の型、真偽値、数値、日付、必須のカスタム属性、`required_actions` のいずれかが不正である
- Then 対象行は安定したエラーコードで `rejected` となり、値はジョブの表示にも監査イベントにも含めない

## Rule: REQ-IDMANAGEMENT-008 管理者は特定グループのメンバー一覧を CSV にエクスポートできる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-008-01 通常経路

- Given ロール=["admin"] のユーザー "operator" がグループ "engineering" の詳細を開いている
- When 管理者が `/groups/{group_id}/members/exports` へ列 [user_id, preferred_username] を指定してエクスポートを開始する
- Then エクスポートの対象は `group_id` に閉じ、そのグループのメンバーだけを含む
- When 生成完了後、管理者がメンバーの CSV をダウンロードする
- Then 指定したグループのメンバーだけを含む CSV が返る

### Example: EX-IDMANAGEMENT-008-02 `group_id` を指定しない

- Given ロール=["admin"] のユーザー "operator" がグループ "engineering" の詳細を開いている
- When 管理者が `/groups/{group_id}/members/exports` へ列 [user_id, preferred_username] を指定してエクスポートを開始する
- But `group_id` を指定しない
- Then グループ単位の指定は必須であり、エクスポートの開始は InvalidRequestError で拒否される

### Example: EX-IDMANAGEMENT-008-03 別グループのパスでそのエクスポート ID を指定する

- Given ロール=["admin"] のユーザー "operator" がグループ "engineering" の詳細を開いている
- When 管理者が `/groups/{group_id}/members/exports` へ列 [user_id, preferred_username] を指定してエクスポートを開始する
- Then エクスポートの対象は `group_id` に閉じ、そのグループのメンバーだけを含む
- When 生成完了後、管理者がメンバーの CSV をダウンロードする
- But 別グループのパスでそのエクスポート ID を指定する
- Then グループごとに分離しているため、取得とダウンロードは InvalidRequestError で拒否される

## Rule: REQ-IDMANAGEMENT-009 管理者はエージェントを登録しクライアント資格情報をバインドできる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-009-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- Then エージェント "batch-agent" が指定した区分で登録される
- When 管理者 "operator" がエージェント "batch-agent" にクライアント資格情報をバインドする
- Then クライアント資格情報がバインドされる
- When 管理者 "operator" がエージェント "batch-agent" を無効化する
- Then エージェントは無効状態になる
- When 管理者 "operator" がエージェント "batch-agent" を再有効化する
- Then エージェント一覧に "batch-agent" が表示される

### Example: EX-IDMANAGEMENT-009-02 `kind` を指定しない

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- But `kind` を指定しない
- Then エラー "AgentKindRequiredError"
- And 区分は実行時のトークン発行可否を決めるため、既定値で補わない (REQ-OAUTH2-050)

### Example: EX-IDMANAGEMENT-009-03 `kind` が既知のどの値でもない

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- But `kind` が既知のどの値でもない
- Then エラー "InvalidAgentKindError"
- And 既知の値へ丸めない

### Example: EX-IDMANAGEMENT-009-04 別テナントのクライアント資格情報をバインドする

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- Then エージェント "batch-agent" が指定した区分で登録される
- When 管理者 "operator" がエージェント "batch-agent" にクライアント資格情報をバインドする
- But 別テナントのクライアント資格情報をバインドする
- Then テナント "acme" の Agent にテナント "default" の `client_id` を指定する
- And エラー "InvalidRequestError"

## Rule: REQ-IDMANAGEMENT-010 管理者は無効化したユーザーを再有効化できる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-010-01 通常経路

- Given 管理者がユーザー "alice" を無効化している
- When 管理者がユーザー "alice" を再有効化する
- Then ユーザー `alice` のステータスは `Active` である
- Then "UserEnabled" が発行される

## Rule: REQ-IDMANAGEMENT-011 管理者はユーザーの削除を予約し、猶予期間内に復元できる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-011-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- And ユーザー "alice" は Active である
- When 管理者 "operator" がユーザー "alice" を削除する
- Then ユーザー `alice` のステータスは `PendingDeletion` である
- Then "UserSoftDeleted" が発行される
- When 管理者 "operator" がユーザー "alice" を復元する
- Then ユーザー `alice` のステータスは `Active` である
- Then "UserRestored" が発行される

## Rule: REQ-IDMANAGEMENT-012 削除を予約したユーザーはログインを拒否される (superseded by REQ-PLATFORM-002)

引き金は IdManagement の削除予約、観測はログインの拒否であり、どちらの Context も単独では起こせない。削除の予約と復元が到達経路の開閉と対応することを、REQ-PLATFORM-002 が保証として述べる。

## Rule: REQ-IDMANAGEMENT-013 管理者はユーザーを完全削除できる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-013-01 通常経路

- Given ユーザー "alice" は PendingDeletion である
- When 管理者がユーザー "alice" を完全削除する
- Then ユーザー `alice` のステータスは `Deleted` である
- Then "UserDeleted" が発行される

### Example: EX-IDMANAGEMENT-013-02 対象が操作者自身であり、`admin` または `system_admin` を持つ

- Given ユーザー "alice" は PendingDeletion である
- When 管理者がユーザー "alice" を完全削除する
- But 対象が操作者自身であり、`admin` または `system_admin` を持つ
- Then 削除の予約、復元、完全削除のいずれも拒否される
- And エラー "self_delete_forbidden"

## Rule: REQ-IDMANAGEMENT-014 管理 API のアクセスはロールに応じて制御される

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-014-01 通常経路

- Given ロールに "admin" を持つユーザー "operator" が認証済みである
- When 管理者 "operator" が `preferred_username` "bob" のユーザーを作成する
- Then "UserCreated" が発行される
- When 管理者 "operator" がユーザー一覧を取得する
- Then レスポンスにユーザー "bob" が含まれる

### Example: EX-IDMANAGEMENT-014-02 `admin` ロールを持たないユーザーが管理 API を呼ぶ

- Given ロールに "admin" を持つユーザー "operator" が認証済みである
- When 管理者 "operator" が `preferred_username` "bob" のユーザーを作成する
- Then "UserCreated" が発行される
- When 管理者 "operator" がユーザー一覧を取得する
- But `admin` ロールを持たないユーザーが管理 API を呼ぶ
- Then ロールが空のユーザー "alice" が認証済みである
- And ユーザー "alice" がユーザー一覧を取得する
- And エラー "AccessDeniedError"

## Rule: REQ-IDMANAGEMENT-015 管理者はグループを作成しユーザーを所属させると有効ロールにグループ由来ロールが乗る

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-015-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が認証済みである
- And ロールが空のユーザー "alice" が同一テナントに存在する
- When 管理者 "operator" がロール=["catalog:read"] のグループ "engineering" を作成する
- Then "GroupCreated" が発行される
- When 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
- Then "GroupMemberAdded" が発行される
- When 管理者がユーザー "alice" の所属グループを取得する
- Then 実効ロールに "catalog:read" が含まれる
- Then `group_roles` は "catalog:read" を含み、`direct_roles` は空である

### Example: EX-IDMANAGEMENT-015-02 同じユーザーを同じグループへ再度所属させる

- Given ロール=["admin"] のユーザー "operator" が認証済みである
- And ロールが空のユーザー "alice" が同一テナントに存在する
- When 管理者 "operator" がロール=["catalog:read"] のグループ "engineering" を作成する
- Then "GroupCreated" が発行される
- When 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
- But 同じユーザーを同じグループへ再度所属させる
- Then 管理者 "operator" がユーザー "alice" をグループ "engineering" に再度所属させる
- And "GroupMemberAdded" は再発行されない

## Rule: REQ-IDMANAGEMENT-016 ユーザーは自分のプロフィール表示名を更新できる

Primary actor: `AuthenticatedSelf`

### Example: EX-IDMANAGEMENT-016-01 通常経路

- Given ユーザー "alice" が認証済みでマイアカウントのプロフィールを開いている
- When ユーザー "alice" が表示名を更新する
- Then 更新後のプロフィールに新しい表示名が反映される
- Then `editable_by_user=false` の属性は更新できない

## Rule: REQ-IDMANAGEMENT-017 ユーザーはメールアドレス変更を起票し確認リンクで確定できる

Primary actor: `AuthenticatedSelf`

### Example: EX-IDMANAGEMENT-017-01 通常経路

- Given ユーザー "alice" が認証済みでメールアドレス画面を開いている
- When ユーザー "alice" が新しいメールアドレスへの変更を起票する
- Then 新アドレスへ確認リンクが送られる
- When ユーザー "alice" が確認リンクのトークンで変更を確定する
- Then プライマリメールアドレスが新しいアドレスへ更新される

### Example: EX-IDMANAGEMENT-017-02 リンクを開くだけではトークンを消費しない

- Given ユーザー "alice" 宛に有効なメールアドレス変更の確認トークンが発行されている
- When ブラウザーまたはメールスキャナーが確認リンクを `GET` または `HEAD` で先読みする
- Then トークンは未使用のまま残り、プライマリメールアドレスは変わらない
- When ユーザー "alice" が同じトークンで変更を確定する
- Then プライマリメールアドレスが新しいアドレスへ更新される

### Example: EX-IDMANAGEMENT-017-03 確定済みのトークンは再利用できない

- Given ユーザー "alice" が確認トークンでメールアドレス変更を確定済みである
- When ユーザー "alice" が同じトークンでもう一度確定する
- Then エラー "InvalidRequestError"
- And プライマリメールアドレスは一度目の確定結果のまま変わらない

### Example: EX-IDMANAGEMENT-017-04 別の用途で発行されたトークンは受け付けない

- Given ユーザー "alice" 宛にメールアドレス変更ではない用途のアクショントークンが発行されている
- When ユーザー "alice" がそのトークンで変更を確定する
- Then エラー "InvalidRequestError"
- And レスポンスは無効なトークンに対するものと区別できない
- And プライマリメールアドレスは変わらず、そのトークンは未使用のまま残る

### Example: EX-IDMANAGEMENT-017-05 期限切れのトークンは受け付けない

- Given ユーザー "alice" 宛のメールアドレス変更の確認トークンが期限切れである
- When ユーザー "alice" がそのトークンで変更を確定する
- Then エラー "InvalidRequestError"
- And プライマリメールアドレスは変わらない

### Example: EX-IDMANAGEMENT-017-06 起票後に新アドレスが他のユーザーのものになっている

- Given ユーザー "alice" 宛に有効なメールアドレス変更の確認トークンが発行されている
- And 起票から確定までの間に別のユーザーが同じアドレスを自分のものとして確定している
- When ユーザー "alice" がそのトークンで変更を確定する
- Then エラー "ConflictError"
- And プライマリメールアドレスは変わらず、トークンは未使用のまま残る

## Rule: REQ-IDMANAGEMENT-018 ユーザーは自分のアカウントデータをエクスポートできる

Primary actor: `AuthenticatedSelf`

### Example: EX-IDMANAGEMENT-018-01 通常経路

- Given ユーザー "alice" が認証済みでデータとプライバシー画面を開いている
- When ユーザー "alice" がアカウントデータをエクスポートする
- Then レスポンスに自分のプロフィールと同意の一覧が含まれる

## Rule: REQ-IDMANAGEMENT-019 アカウント API は他人のリソースを返さない

Primary actor: `AuthenticatedSelf`

### Example: EX-IDMANAGEMENT-019-01 通常経路

- Given ユーザー "alice" が認証済みである
- When ユーザー "alice" のアカウント概要を取得する
- Then レスポンスは "alice" 自身のデータだけを含み、ロールは含まない

## Rule: REQ-IDMANAGEMENT-020 管理者は CEL の規則で動的グループの所属を管理できる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-020-01 通常経路

- Given `department` 属性が定義され、Engineering の User と Sales の User が存在する
- And `membership_type=dynamic` のグループが存在する
- When 管理者が `user.department == "Engineering"` を保存して有効化する
- Then 全件の再評価後、Engineering の有効な User だけが動的規則を由来として所属する
- Then 実効ロールと Application の割り当ては、その所属を参照する

## Rule: REQ-IDMANAGEMENT-021 CEL の規則は保存前に選んだユーザーでプレビューできる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-021-01 通常経路

- Given 管理者が最大 100 件の User を選択している
- When 未保存の CEL 式を評価する
- Then レスポンスは一致の有無と、追加・削除・変更なしの判定を返し、属性値そのものは返さない

## Rule: REQ-IDMANAGEMENT-022 不正な CEL の規則と動的グループの手動操作は拒否される

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-022-01 通常経路

- When 管理者が、未定義の属性または許可外の関数を参照する CEL 式を保存する
- Then 保存は拒否される
- When 管理者が動的グループに対して `AddGroupMember` または `RemoveGroupMember` を手動で呼ぶ
- Then メンバーシップの変更は拒否される

## Rule: REQ-IDMANAGEMENT-023 評価できない規則は権限を付与しない

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-023-01 通常経路

- Given 有効な動的規則のバージョンが更新された
- When 新しいバージョンの動的規則を再評価する
- Then 旧バージョンのメンバーシップは直ちに実効ロールから除外される
- Then 再評価に失敗した User は、新しいバージョンのメンバーシップを得ない

## Rule: REQ-IDMANAGEMENT-024 管理者はグループの連絡先メールとカスタム属性を、テナント定義のスキーマに従って設定できる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-024-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- And テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- When "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
- Then 作成したグループの `email` と `attributes` が指定どおりに保存され、"GroupCreated" が発行される
- When "operator" が同じグループの `email` と `attributes` を更新する
- Then 更新後のグループに新しい値が反映され、"GroupUpdated" の `changed_fields` に "email" と "attributes" が含まれる

### Example: EX-IDMANAGEMENT-024-02 `email` がメールアドレスの形式を満たさない

- Given admin ロールを持つ "operator" が認証済みである
- And テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- When "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
- But `email` がメールアドレスの形式を満たさない
- Then 作成は InvalidEmailError で拒否される

### Example: EX-IDMANAGEMENT-024-03 `attributes` に未定義のキーを指定する、または定義済みのキーと型が一致しない

- Given admin ロールを持つ "operator" が認証済みである
- And テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- When "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
- But `attributes` に未定義のキーを指定する、または定義済みのキーと型が一致しない
- Then 作成は InvalidGroupAttributeError で拒否される

## Rule: REQ-IDMANAGEMENT-025 管理 API クライアントはプリンシパルの種類と操作の粒度でだけ User / Group / Agent を操作できる

Primary actor: `ManagementApiClient`

### Example: EX-IDMANAGEMENT-025-01 通常経路

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが User、Group、または Agent の操作をリクエストする
- Then `users:read` は User の参照に加えて、User を変更しない CSV エクスポートの開始、参照、ダウンロード、取り消しを許可する
- Then `groups:read` は Group の参照に加えて、Group を変更しない動的グループ規則のプレビューと CSV エクスポートを許可する
- Then `groups:write` は Group の変更に加えて、CSV インポートのプレビューと適用を許可する
- Then `agents:write` は Agent のキルと削除の両方を許可する

### Example: EX-IDMANAGEMENT-025-02 `users:read` だけで User の変更または CSV インポートを要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが User、Group、または Agent の操作をリクエストする
- But `users:read` だけで User の変更または CSV インポートを要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-IDMANAGEMENT-025-03 `groups:read` だけで Group の CSV インポートまたはその適用を要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが User、Group、または Agent の操作をリクエストする
- But `groups:read` だけで Group の CSV インポートまたはその適用を要求する
- Then 操作は AccessDeniedError で拒否され、`Group` は 1 件も作成、更新、削除されない

### Example: EX-IDMANAGEMENT-025-04 `users:*` だけで Group または Agent の操作を要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが User、Group、または Agent の操作をリクエストする
- But `users:*` だけで Group または Agent の操作を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-IDMANAGEMENT-025-05 `agents:read` だけで Agent のキルまたは削除を要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが User、Group、または Agent の操作をリクエストする
- But `agents:read` だけで Agent のキルまたは削除を要求する
- Then 操作は AccessDeniedError で拒否される

## Rule: REQ-IDMANAGEMENT-026 管理者は Group CSV を検証して有効な行だけをインポートできる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-026-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 有効な行は作成または更新され、無効な行は `rejected` として残る。各行の名前、説明、連絡先、ロール、カスタム属性、動的規則は不可分に保存される

### Example: EX-IDMANAGEMENT-026-02 CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- But CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える
- Then インポートの投入は拒否される
- And エラー "csv_too_large" / "too_many_rows" / "field_too_large"

### Example: EX-IDMANAGEMENT-026-03 CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- But CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる
- Then インポートの投入は拒否される
- And エラー "invalid_header"

### Example: EX-IDMANAGEMENT-026-04 行に `id` も `name` も無い、`id` と `name` が別の `Group` を示す、同じ対象または同じ最終 `name` を複数行が示す

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then 行に `id` も `name` も無い、`id` と `name` が別の `Group` を示す、同じ対象または同じ最終 `name` を複数行が示す
- Then 対象行は `rejected` となり、安定したエラーコードを返す

### Example: EX-IDMANAGEMENT-026-05 プレビュージョブが存在しない、`queued` または `failed` である、別テナントに属する、保存済みのペイロードとダイジェストが一致しない

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- But プレビュージョブが存在しない、`queued` または `failed` である、別テナントに属する、保存済みのペイロードとダイジェストが一致しない
- Then 適用は `Group` を変更せず `InvalidRequestError` または `AccessDeniedError` で拒否される

### Example: EX-IDMANAGEMENT-026-06 プレビュー後に対象 `Group` の状態が別の操作で変更されている

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then プレビュー後に対象 `Group` の状態が別の操作で変更されている
- Then 適用は古いプレビュー計画を実行せず、現在状態から `updated`、`unchanged`、`rejected` を再判定する

### Example: EX-IDMANAGEMENT-026-07 既存 `Group` の `membership_type` を現在値と異なる値へ変更する

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 既存 `Group` の `membership_type` を現在値と異なる値へ変更する
- Then 対象行は安定したエラーコード `immutable_membership_type` で `rejected` となり、その `Group` の `membership_type`、ロール、メンバーシップはいずれも変更されない

### Example: EX-IDMANAGEMENT-026-08 対象 `Group` が外部の取り込み元に管理されている、または所有権を判定できない

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 対象 `Group` が外部の取り込み元に管理されている、または所有権を判定できない
- Then 対象行は安定したエラーコード `source_managed` で `rejected` となり、`Group` は変更されない

### Example: EX-IDMANAGEMENT-026-09 `membership_type=manual` の `Group` に動的規則の式を与える、最終状態の式が空のまま規則を有効化する、または式が未定義の属性か許可外の関数を参照する

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then `membership_type=manual` の `Group` に動的規則の式を与える、最終状態の式が空のまま規則を有効化する、または式が未定義の属性か許可外の関数を参照する
- Then 対象行は安定したエラーコード `invalid_dynamic_rule` で `rejected` となり、その `Group` の動的規則は保存されない

### Example: EX-IDMANAGEMENT-026-10 `email` がメールアドレスの形式を満たさない、`custom:<key>` の値が宣言された型の字句形に合わない、またはテナントスキーマに無い `custom:<key>` 列を与える

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then `email` がメールアドレスの形式を満たさない、`custom:<key>` の値が宣言された型の字句形に合わない、またはテナントスキーマに無い `custom:<key>` 列を与える
- Then 対象行は安定したエラーコードで `rejected` となり、その `Group` の連絡先も属性も変更されず、値はジョブの表示にも監査イベントにも含めない

### Example: EX-IDMANAGEMENT-026-11 1 行の検証、保存、監査処理が途中で失敗する

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている
- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 1 行の検証、保存、監査処理が途中で失敗する
- Then その行の名前、説明、連絡先、ロール、カスタム属性、動的規則は一部も保存されず、他の有効な行は適用を続ける

## Rule: REQ-IDMANAGEMENT-027 管理者はエクスポートしたグループ CSV を安全に再適用できる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-027-01 通常経路

- Given 実効 `TenantGroupAttributeSchema` に `custom:cost_center` があり、10,000 件の `Group` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列と `custom:cost_center` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- Then `lifecycle_action` 列は全行で空として出力される
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`Group` は 1 件も作成、更新、削除されない
- When 管理者が 1 行の `roles`、`email`、`custom:cost_center` だけを編集して再びプレビューする
- Then 変更行だけが `updated`、残りは `unchanged` と計画される
- When 管理者が成功済みプレビュージョブの ID を指定して適用する
- Then 指定した書き込み可能な列だけが更新され、指定しなかった列は維持される

### Example: EX-IDMANAGEMENT-027-02 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む

- Given 実効 `TenantGroupAttributeSchema` に `custom:cost_center` があり、10,000 件の `Group` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列と `custom:cost_center` を機械可読ヘッダーでエクスポートする
- But 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む
- Then 可逆な数式安全変換と RFC 4180 の引用により `decode(encode(value))` は元の値と一致する

### Example: EX-IDMANAGEMENT-027-03 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える

- Given 実効 `TenantGroupAttributeSchema` に `custom:cost_center` があり、10,000 件の `Group` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列と `custom:cost_center` を機械可読ヘッダーでエクスポートする
- But 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える
- Then `Group` エクスポートは `csv_transfer_limit_exceeded` で失敗し、再インポートできない成功済み成果物を作らない
- And 管理者は列を絞って複数の成果物に分割できる

### Example: EX-IDMANAGEMENT-027-04 `id`、`created_at`、`updated_at` の値だけを編集する

- Given 実効 `TenantGroupAttributeSchema` に `custom:cost_center` があり、10,000 件の `Group` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列と `custom:cost_center` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- Then `lifecycle_action` 列は全行で空として出力される
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`Group` は 1 件も作成、更新、削除されない
- When 管理者が 1 行の `roles`、`email`、`custom:cost_center` だけを編集して再びプレビューする
- But `id`、`created_at`、`updated_at` の値だけを編集する
- Then 読み取り専用列は受理したうえで無視し、書き込み可能な列に差分がなければ `unchanged` とする

### Example: EX-IDMANAGEMENT-027-05 `roles`、`membership_type`、`email`、または `custom:<key>` の値が不正である

- Given 実効 `TenantGroupAttributeSchema` に `custom:cost_center` があり、10,000 件の `Group` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列と `custom:cost_center` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- Then `lifecycle_action` 列は全行で空として出力される
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`Group` は 1 件も作成、更新、削除されない
- When 管理者が 1 行の `roles`、`email`、`custom:cost_center` だけを編集して再びプレビューする
- But `roles`、`membership_type`、`email`、または `custom:<key>` の値が不正である
- Then 対象行は安定したエラーコードで `rejected` となり、値はジョブの表示にも監査イベントにも含めない

## Rule: REQ-IDMANAGEMENT-028 管理者は CSV の明示的な行操作でグループを削除できる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-028-01 通常経路

- Given グループ "engineering" が存在し、ユーザー "alice" が所属している
- And 管理者はエクスポートした CSV の "engineering" の行だけに `lifecycle_action=delete` を書き込んでいる
- When 管理者がその CSV を事前検証へ投入する
- Then プレビューは削除される `Group` の件数と、巻き込まれるメンバーシップの件数を他の操作と分けて返し、`Group` もメンバーシップも変更されない
- Then CSV に現れない `Group` は削除対象に含まれない
- When 管理者が削除を含むことを確認したうえで適用する
- Then "engineering" は削除され、"alice" のメンバーシップは同じトランザクションで解除され、"GroupMemberRemoved" と "GroupDeleted" が発行される
- Then "alice" の実効ロールから "engineering" のロールが外れる
- Then 同じファイルの他の行の作成と更新は削除の影響を受けずに適用される

### Example: EX-IDMANAGEMENT-028-02 `lifecycle_action` に `delete` 以外の値を書く

- Given グループ "engineering" が存在し、ユーザー "alice" が所属している
- And 管理者はエクスポートした CSV の "engineering" の行だけに `lifecycle_action=delete` を書き込んでいる
- When 管理者がその CSV を事前検証へ投入する
- But `lifecycle_action` に `delete` 以外の値を書く
- Then 対象行は安定したエラーコード `invalid_lifecycle_action` で `rejected` となり、既知の値へ丸めない

### Example: EX-IDMANAGEMENT-028-03 `lifecycle_action=delete` の行が、現在状態に対して差分のある書き込み可能な列も同時に持つ

- Given グループ "engineering" が存在し、ユーザー "alice" が所属している
- And 管理者はエクスポートした CSV の "engineering" の行だけに `lifecycle_action=delete` を書き込んでいる
- When 管理者がその CSV を事前検証へ投入する
- But `lifecycle_action=delete` の行が、現在状態に対して差分のある書き込み可能な列も同時に持つ
- Then 対象行は安定したエラーコード `conflicting_lifecycle_action` で `rejected` となり、その `Group` は更新も削除もされない

### Example: EX-IDMANAGEMENT-028-04 `lifecycle_action=delete` の行が既存の `Group` を指さない

- Given グループ "engineering" が存在し、ユーザー "alice" が所属している
- And 管理者はエクスポートした CSV の "engineering" の行だけに `lifecycle_action=delete` を書き込んでいる
- When 管理者がその CSV を事前検証へ投入する
- But `lifecycle_action=delete` の行が既存の `Group` を指さない
- Then 対象行は安定したエラーコード `target_not_found` で `rejected` となり、`Group` は作成されない

### Example: EX-IDMANAGEMENT-028-05 対象 `Group` が外部の取り込み元に管理されている、または所有権を判定できない

- Given グループ "engineering" が存在し、ユーザー "alice" が所属している
- And 管理者はエクスポートした CSV の "engineering" の行だけに `lifecycle_action=delete` を書き込んでいる
- When 管理者がその CSV を事前検証へ投入する
- But 対象 `Group` が外部の取り込み元に管理されている、または所有権を判定できない
- Then 対象行は安定したエラーコード `source_managed` で `rejected` となり、`Group` は削除されない

## Rule: REQ-IDMANAGEMENT-029 管理者は 1 つのグループのメンバーシップ CSV を検証し、行ごとに宣言した所属状態だけを適用できる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-029-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない
- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- Then プレビュージョブは `added`、`removed`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、メンバーシップは変更されない
- Then 解除される件数は他の操作と分けて返される
- When 管理者が解除を含むことを確認したうえで、同じテナントかつ同じグループの成功済みプレビュージョブの ID を指定して適用する
- Then CSV は再送されず、保存済みのプレビューペイロードとその SHA-256 が検証される
- Then `membership_state=present` の "bob" の行はメンバーシップを追加し、"GroupMemberAdded" を発行する
- Then `membership_state=absent` の "alice" の行は手動メンバーシップを解除し、"GroupMemberRemoved" を発行する
- Then "alice" の実効ロールから "engineering" のロールが外れ、"bob" の実効ロールにそれが加わる

### Example: EX-IDMANAGEMENT-029-02 CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない
- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- But CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える
- Then インポートの投入は拒否される
- And エラー "csv_too_large" / "too_many_rows" / "field_too_large"

### Example: EX-IDMANAGEMENT-029-03 CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない
- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- But CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる
- Then インポートの投入は拒否される
- And エラー "invalid_header"

### Example: EX-IDMANAGEMENT-029-04 CSV のヘッダーに `membership_state` が無い

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない
- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- But CSV のヘッダーに `membership_state` が無い
- Then ファイル全体が安定したエラーコード "invalid_header" で拒否され、メンバーシップは 1 件も追加も解除もされない

### Example: EX-IDMANAGEMENT-029-05 `membership_state` のセルが空である、または `present` と `absent` のどちらでもない

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない
- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- But `membership_state` のセルが空である、または `present` と `absent` のどちらでもない
- Then 対象行は安定したエラーコード `invalid_membership_state` で `rejected` となり、既知の値へ丸めず、その User のメンバーシップは変更されない

### Example: EX-IDMANAGEMENT-029-06 行に `user_id` も `preferred_username` も無い、両者が別の User を示す、対象 User がテナントに存在しない、同じ User を複数行が示す

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない
- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- But 行に `user_id` も `preferred_username` も無い、両者が別の User を示す、対象 User がテナントに存在しない、同じ User を複数行が示す
- Then 対象行は安定したエラーコード `missing_identifier` / `identifier_mismatch` / `target_not_found` / `duplicate_target` で `rejected` となり、メンバーシップは変更されない

### Example: EX-IDMANAGEMENT-029-07 行の `group_id` または `group_name` がパスのグループ以外を示す

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない
- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- But 行の `group_id` または `group_name` がパスのグループ以外を示す
- Then 対象行は安定したエラーコード `group_mismatch` で `rejected` となり、示された別のグループのメンバーシップも変更されない

### Example: EX-IDMANAGEMENT-029-08 プレビュージョブが存在しない、`queued` または `failed` である、別テナントまたは別グループに属する、保存済みのペイロードとダイジェストが一致しない

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない
- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- Then プレビュージョブは `added`、`removed`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、メンバーシップは変更されない
- When 管理者が解除を含むことを確認したうえで、同じテナントかつ同じグループの成功済みプレビュージョブの ID を指定して適用する
- But プレビュージョブが存在しない、`queued` または `failed` である、別テナントまたは別グループに属する、保存済みのペイロードとダイジェストが一致しない
- Then 適用はメンバーシップを変更せず `InvalidRequestError`、`AccessDeniedError`、または `GroupMembershipImportNotFoundError` で拒否される

### Example: EX-IDMANAGEMENT-029-09 プレビュー後にメンバーシップが別の操作で変更されている

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない
- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- Then プレビュージョブは `added`、`removed`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、メンバーシップは変更されない
- When 管理者が解除を含むことを確認したうえで、同じテナントかつ同じグループの成功済みプレビュージョブの ID を指定して適用する
- Then プレビュー後にメンバーシップが別の操作で変更されている
- Then 適用は古いプレビュー計画を実行せず、現在のメンバーシップから `added`、`removed`、`unchanged`、`rejected` を再判定する

### Example: EX-IDMANAGEMENT-029-10 1 行のメンバーシップ確定または監査記録が途中で失敗する

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない
- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- Then プレビュージョブは `added`、`removed`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、メンバーシップは変更されない
- When 管理者が解除を含むことを確認したうえで、同じテナントかつ同じグループの成功済みプレビュージョブの ID を指定して適用する
- Then 1 行のメンバーシップ確定または監査記録が途中で失敗する
- Then その行のメンバーシップも監査記録も一部すら残らず、他の有効な行は適用を続ける

## Rule: REQ-IDMANAGEMENT-030 管理者はエクスポートしたメンバーシップ CSV を、分割しても大量解除にならない形で再適用できる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-030-01 通常経路

- Given 手動グループ "engineering" が 10,000 件の手動メンバーシップを持ち、実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- Then `membership_state` 列は全行で `present` として出力される
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、メンバーシップは 1 件も追加も解除もされない
- When 管理者がその成果物を 2 つのファイルへ分け、片方だけをプレビューして適用する
- Then もう一方のファイルにしか現れないメンバーシップは解除されない
- Then CSV に現れない User は、そのグループの所属についても一切変更されない

### Example: EX-IDMANAGEMENT-030-02 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む

- Given 手動グループ "engineering" が 10,000 件の手動メンバーシップを持ち、実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列を機械可読ヘッダーでエクスポートする
- But 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む
- Then 可逆な数式安全変換と RFC 4180 の引用により `decode(encode(value))` は元の値と一致する

### Example: EX-IDMANAGEMENT-030-03 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える

- Given 手動グループ "engineering" が 10,000 件の手動メンバーシップを持ち、実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列を機械可読ヘッダーでエクスポートする
- But 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える
- Then メンバーシップのエクスポートは `csv_transfer_limit_exceeded` で失敗し、再インポートできない成功済み成果物を作らない

### Example: EX-IDMANAGEMENT-030-04 `source`、`created_at`、`group_name` の値だけを編集する

- Given 手動グループ "engineering" が 10,000 件の手動メンバーシップを持ち、実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列を機械可読ヘッダーでエクスポートする
- Then `membership_state` 列は全行で `present` として出力される
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- But `source`、`created_at`、`group_name` の値だけを編集する
- Then 読み取り専用列は受理したうえで無視し、`group_name` が別のグループを指す場合にだけ行を拒否する

## Rule: REQ-IDMANAGEMENT-031 メンバーシップ CSV は動的規則と外部の取り込み元が所有する所属を上書きしない

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-031-01 対象グループの `membership_type` が `dynamic` である

- Given ロール=["admin"] のユーザー "operator" がメンバーシップ CSV を用意している
- When 管理者がそのグループの事前検証へ CSV を投入する
- But 対象グループの `membership_type` が `dynamic` である
- Then ファイル全体が安定したエラーコード `dynamic_group` で拒否され、メンバーシップは 1 件も追加も解除もされない

### Example: EX-IDMANAGEMENT-031-02 対象 User の現在のメンバーシップの `source` が `dynamic_rule` である

- Given ロール=["admin"] のユーザー "operator" がメンバーシップ CSV を用意している
- When 管理者がそのグループの事前検証へ CSV を投入する
- But 対象 User の現在のメンバーシップの `source` が `dynamic_rule` である
- Then 対象行は安定したエラーコード `dynamic_membership` で `rejected` となり、`present` でも `absent` でもそのメンバーシップは変更されない

### Example: EX-IDMANAGEMENT-031-03 対象グループが外部の取り込み元に管理されている、または所有権を判定できない

- Given ロール=["admin"] のユーザー "operator" がメンバーシップ CSV を用意している
- When 管理者がそのグループの事前検証へ CSV を投入する
- But 対象グループが外部の取り込み元に管理されている、または所有権を判定できない
- Then ファイル全体が安定したエラーコード `source_managed` で拒否され、メンバーシップは 1 件も追加も解除もされない

### Example: EX-IDMANAGEMENT-031-04 対象 User が外部の取り込み元に管理されている、または所有権を判定できない

- Given ロール=["admin"] のユーザー "operator" がメンバーシップ CSV を用意している
- When 管理者がそのグループの事前検証へ CSV を投入する
- But 対象 User が外部の取り込み元に管理されている、または所有権を判定できない
- Then 対象行は安定したエラーコード `source_managed` で `rejected` となり、その User のメンバーシップは変更されない

### Example: EX-IDMANAGEMENT-031-05 対象グループがテナントに存在しない、または適用の直前に削除されている

- Given ロール=["admin"] のユーザー "operator" がメンバーシップ CSV を用意している
- When 管理者がそのグループの事前検証へ CSV を投入する
- But 対象グループがテナントに存在しない、または適用の直前に削除されている
- Then ファイル全体が安定したエラーコード `target_not_found` で拒否され、グループは作成されず、メンバーシップも作られない
