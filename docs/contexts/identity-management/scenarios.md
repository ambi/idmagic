# IdManagement Scenarios

### REQ-IDMANAGEMENT-001: フェデレーションの JIT はパスワード資格情報を作らず有効な User を作成する
- ACTOR EndUser
- GIVEN Authentication Context が上流のトークンまたは Assertion と、テナントの JIT ポリシーを検証済みである
- WHEN Authentication Context が ProvisionFederatedUser を呼ぶ
- THEN 対応付けたユーザー名、任意の名前・メールアドレス・属性、テナントのリソース上限、一意性を検証する
  - ALT ユーザー名またはメールアドレスが衝突する、リソース上限を超える、属性スキーマに違反する → User を作成せずエラーを返す
- THEN `password_hash` が空の `Active` な User を作成し、`UserCreated` を発行する

### REQ-IDMANAGEMENT-002: API トークンの発行者は account スコープで自身の情報だけを操作できる
- ACTOR SelfApiClient
- GIVEN クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- WHEN クライアントが概要、プロフィール、データエクスポート、またはプライマリメールアドレスの変更申請を要求する
  - ALT `account:read` だけで変更操作を要求する → 操作は AccessDeniedError で拒否される
  - ALT トークンのテナントまたは `user_id` が操作対象と一致しない → 操作は AccessDeniedError で拒否される
- THEN `account:read` スコープは、自身の概要、プロフィール、データエクスポートの参照だけを許可する
- THEN `account:write` スコープは、自身のプロフィールとプライマリメールアドレスの変更申請だけを許可する

### REQ-IDMANAGEMENT-003: メールアドレス確認画面は未認証でも CSRF 境界を確立できる
- ACTOR EndUser
- WHEN EndUser がメールアドレス確認の文脈を取得する
- THEN レスポンスに CSRF トークンと SameSite 属性を持つ Cookie が含まれる
- WHEN EndUser がその CSRF トークンと Cookie を添えてメールアドレスの確認を送信する
  - ALT CSRF トークンと Cookie が一致しない → 確認は InvalidRequestError で拒否される
- THEN メールアドレスの確認が受理される

### REQ-IDMANAGEMENT-004: 管理者は CSV を検証して有効な行だけをインポートできる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- WHEN 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
  - ALT CSV が実効 `UserCsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える → インポートの投入は拒否される → エラー "csv_too_large" / "too_many_rows" / "field_too_large"
  - ALT CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる → インポートの投入は拒否される → エラー "invalid_header"
- THEN プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
  - ALT 行の `id` と `preferred_username` が別の `User` を示す、識別子がない、同じ対象または同じ最終ユーザー名を複数行が示す → 対象行は `rejected` となり、安定したエラーコードを返す
- WHEN 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
  - ALT プレビュージョブが存在しない、`queued` または `failed` である、別テナントに属する、保存済みのペイロードとダイジェストが一致しない → 適用は `User` を変更せず `InvalidRequestError` または `AccessDeniedError` で拒否される
- THEN CSV は再送されず、保存済みのプレビューペイロードが使われる
- THEN 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
  - ALT プレビュー後に対象 `User` の状態が別の操作で変更されている → 適用は古いプレビュー計画を実行せず、現在状態から `updated`、`unchanged`、`rejected` を再判定する
- THEN 有効な行は作成または更新され、無効な行は `rejected` として残る。各行のプロフィール、ロール、必須操作、カスタム属性は不可分に保存される
  - ALT 対象 `User` が外部の取り込み元に管理されている → 対象行は安定したエラーコード `source_managed` で `rejected` となり、`User` は変更されない
  - ALT 1 行の検証、保存、監査処理が途中で失敗する → その行のプロフィール、ロール、必須操作、カスタム属性は一部も保存されず、他の有効な行は適用を続ける

### REQ-IDMANAGEMENT-005: 管理者はユーザー一覧をページングしながら安定して閲覧できる
- ACTOR TenantAdministrator
- GIVEN 所属テナントに `limit` を超えるユーザーが存在する
- WHEN 管理者が `ListAdminUsers` を `limit` だけ指定して実行し、先頭ページを取得する
  - ALT `query` または `status` を指定する → `ListAdminUsers` はテナント全体から条件に一致する `User` だけを返す → `pagination.total_items` は条件一致件数、`total_users` は削除済みを除くフィルター非依存の件数を返す → `query` または `status` を変更した管理者はカーソルを破棄して先頭ページから取得する
  - ALT 条件に一致する `User` が 0 件である → ユーザー一覧は空で、総項目数、総ページ数、現在のページ番号は `0 / 0 / 0` を返す → `first`、`prev`、`next`、`last` の `Link` は返さない
  - ALT 正確な件数の取得に失敗する → 0 件として成功させず、リクエスト全体をサーバーエラーで失敗させる
  - ALT 実行者が TenantAdministrator ロールを持たない → ListAdminUsers は AccessDeniedError で拒否される
- THEN レスポンスは、絞り込みに一致する正確な総件数、総ページ数、現在のページ、ページサイズと、絞り込みに依存しない `total_users` を返す
- THEN レスポンスの `Link` ヘッダー（`rel="next"`）にコンパクトなカーソルが含まれる
- WHEN 一覧の途中で他の管理者がユーザーを 1 件削除する
- THEN 削除されたユーザーは一覧対象から除外される
- WHEN 管理者が取得済みのカーソルで次ページを取得する
  - ALT カーソルが別テナントで発行された、改ざんされた、または `query` / `status` が発行時と異なる → `ListAdminUsers` は InvalidRequestError を返す → 管理者は先頭ページへ戻って取得し直す
- THEN 削除された行を除き、既に返却済みの行との重複なく残りのユーザーが返る
- THEN レスポンスの `Link` ヘッダー（`rel="prev"`）に前ページのカーソルが含まれる
- WHEN 管理者が `rel="prev"` のカーソルで前ページを取得する
- THEN そのページのユーザーが正規の並び順で返る
- WHEN 管理者が `rel="last"` の終端カーソルで最終ページを取得する
- THEN 端数を含む最終ページが返る
- WHEN 管理者が `rel="first"` のカーソルを含まない URL で先頭ページを取得する
- THEN 正規の先頭ページが返る

### REQ-IDMANAGEMENT-006: 管理者はユーザー一覧を CSV に安全にエクスポートできる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- GIVEN 一覧には自テナントのユーザーが存在する
- WHEN 管理者が列 [`preferred_username`, `email`] と `status` フィルターを指定して `/users/exports` へエクスポートを開始する
  - ALT 選択列に `User` の許可一覧にないキー（例: `password_hash`）が含まれる → エクスポート開始は `InvalidRequestError` で拒否される → エラー `invalid_columns`
- THEN エクスポートは 202 とエクスポート ID を返し、ジョブは `queued` である
- WHEN 終端前に管理者がエクスポートを取り消す
- THEN ステータスは `canceled` となり、`DataExportCanceled` が発行される
- THEN `worker` プロセスが生成を開始し、`DataExportStarted` が発行される
- THEN 生成が完了してステータスは `succeeded`、`downloadable` は `true` となり、`total_rows` と `byte_size` が記録される
  - ALT 生成が失敗する → ステータスは `failed`、`downloadable` は `false` となり、`error_code` が記録される → `DataExportFailed` が発行される → 不完全なファイルはダウンロードできない
- THEN DataExportSucceeded が発行される
- WHEN 管理者がファイルをダウンロードする
  - ALT セル値が \"=\", \"+\", \"-\", \"@\", タブ, CR, LF のいずれかで始まる → 数式の注入を避ける可逆な接頭辞を付けて出力し、インポート側の変換器は規定どおり接頭辞 1 文字だけを取り除く
  - ALT 保持期限を経過している → ステータスは `expired`、`downloadable` は `false` となる → ファイル本体は完全削除され、ダウンロードは `InvalidRequestError` で拒否される
  - ALT `User` エクスポートの ID を `/groups/exports` または別テナントで指定する → 種類とテナントの境界により、取得、ダウンロード、取り消しは `AccessDeniedError` または `InvalidRequestError` で拒否される
- THEN 選択した機械可読キーと一致するヘッダーを持つ RFC 4180 CSV が、`Content-Disposition: attachment` で返る
- THEN DataExportDownloaded が発行される

### REQ-IDMANAGEMENT-007: 管理者はエクスポートしたユーザー CSV を安全に再適用できる
- ACTOR TenantAdministrator
- GIVEN 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `UserCsvTransferPolicy` の上限内に収まる
- WHEN 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
  - ALT 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む → 可逆な数式安全変換と RFC 4180 の引用により `decode(encode(value))` は元の値と一致する
  - ALT 生成結果が実効 `UserCsvTransferPolicy` のいずれかの上限を超える → `User` エクスポートは `csv_transfer_limit_exceeded` で失敗し、再インポートできない成功済み成果物を作らない → 管理者はフィルターまたは列を絞って複数の成果物に分割できる
- THEN `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- WHEN 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- THEN 全行が `unchanged` となり、`User` は変更されない
- WHEN 管理者が 1 行の `email` と `custom:department` だけを編集して再びプレビューする
  - ALT `status`、`mfa_enrolled`、`created_at`、`updated_at`、`id` の値だけを編集する → 読み取り専用列は受理したうえで無視し、書き込み可能な列に差分がなければ `unchanged` とする
  - ALT カスタム属性の型、真偽値、数値、日付、必須のカスタム属性、`required_actions` のいずれかが不正である → 対象行は安定したエラーコードで `rejected` となり、値はジョブの表示にも監査イベントにも含めない
- THEN 変更行だけが updated、残りは unchanged と計画される
- WHEN 管理者が成功済みプレビュージョブの ID を指定して適用する
- THEN 指定した書き込み可能な列だけが更新され、指定しなかった列は維持される

### REQ-IDMANAGEMENT-008: 管理者は特定グループのメンバー一覧を CSV にエクスポートできる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" がグループ "engineering" の詳細を開いている
- WHEN 管理者が `/groups/{group_id}/members/exports` へ列 [user_id, preferred_username] を指定してエクスポートを開始する
  - ALT `group_id` を指定しない → グループ単位の指定は必須であり、エクスポートの開始は InvalidRequestError で拒否される
- THEN エクスポートの対象は `group_id` に閉じ、そのグループのメンバーだけを含む
- WHEN 生成完了後、管理者がメンバーの CSV をダウンロードする
  - ALT 別グループのパスでそのエクスポート ID を指定する → グループごとに分離しているため、取得とダウンロードは InvalidRequestError で拒否される
- THEN 指定したグループのメンバーだけを含む CSV が返る

### REQ-IDMANAGEMENT-009: 管理者はエージェントを登録しクライアント資格情報をバインドできる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- WHEN 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
  - ALT `kind` を指定しない → エラー "AgentKindRequiredError" → 区分は実行時のトークン発行可否を決めるため、既定値で補わない (REQ-OAUTH2-050)
  - ALT `kind` が既知のどの値でもない → エラー "InvalidAgentKindError" → 既知の値へ丸めない
- THEN エージェント "batch-agent" が指定した区分で登録される
- WHEN 管理者 "operator" がエージェント "batch-agent" にクライアント資格情報をバインドする
  - ALT 別テナントのクライアント資格情報をバインドする → テナント "acme" の Agent にテナント "default" の `client_id` を指定する → エラー "InvalidRequestError"
- THEN クライアント資格情報がバインドされる
- WHEN 管理者 "operator" がエージェント "batch-agent" を無効化する
- THEN エージェントは無効状態になる
- WHEN 管理者 "operator" がエージェント "batch-agent" を再有効化する
- THEN エージェント一覧に "batch-agent" が表示される

### REQ-IDMANAGEMENT-010: 管理者は無効化したユーザーを再有効化できる
- ACTOR TenantAdministrator
- GIVEN 管理者がユーザー "alice" を無効化している
- WHEN 管理者がユーザー "alice" を再有効化する
- THEN ユーザー `alice` のステータスは `Active` である
- THEN "UserEnabled" が発行される

### REQ-IDMANAGEMENT-011: 管理者はユーザーの削除を予約し、猶予期間内に復元できる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- GIVEN ユーザー "alice" は Active である
- WHEN 管理者 "operator" がユーザー "alice" を削除する
- THEN ユーザー `alice` のステータスは `PendingDeletion` である
- THEN "UserSoftDeleted" が発行される
- WHEN 管理者 "operator" がユーザー "alice" を復元する
- THEN ユーザー `alice` のステータスは `Active` である
- THEN "UserRestored" が発行される

### REQ-IDMANAGEMENT-012: 削除を予約したユーザーはログインを拒否される
- ACTOR EndUser
- GIVEN ユーザー "alice" は PendingDeletion である
- WHEN ユーザー "alice" が正しいパスワードでログインを試みる
- THEN ログインは拒否される

### REQ-IDMANAGEMENT-013: 管理者はユーザーを完全削除できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" は PendingDeletion である
- WHEN 管理者がユーザー "alice" を完全削除する
  - ALT 対象が操作者自身であり、`admin` または `system_admin` を持つ → 削除の予約、復元、完全削除のいずれも拒否される → エラー "self_delete_forbidden"
- THEN ユーザー `alice` のステータスは `Deleted` である
- THEN "UserDeleted" が発行される

### REQ-IDMANAGEMENT-014: 管理 API のアクセスはロールに応じて制御される
- ACTOR TenantAdministrator
- GIVEN ロールに "admin" を持つユーザー "operator" が認証済みである
- WHEN 管理者 "operator" が `preferred_username` "bob" のユーザーを作成する
- THEN "UserCreated" が発行される
- WHEN 管理者 "operator" がユーザー一覧を取得する
  - ALT `admin` ロールを持たないユーザーが管理 API を呼ぶ → ロールが空のユーザー "alice" が認証済みである → ユーザー "alice" がユーザー一覧を取得する → エラー "AccessDeniedError"
- THEN レスポンスにユーザー "bob" が含まれる

### REQ-IDMANAGEMENT-015: 管理者はグループを作成しユーザーを所属させると有効ロールにグループ由来ロールが乗る
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が認証済みである
- GIVEN ロールが空のユーザー "alice" が同一テナントに存在する
- WHEN 管理者 "operator" がロール=["catalog:read"] のグループ "engineering" を作成する
- THEN "GroupCreated" が発行される
- WHEN 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
  - ALT 同じユーザーを同じグループへ再度所属させる → 管理者 "operator" がユーザー "alice" をグループ "engineering" に再度所属させる → "GroupMemberAdded" は再発行されない
- THEN "GroupMemberAdded" が発行される
- WHEN 管理者がユーザー "alice" の所属グループを取得する
- THEN 実効ロールに "catalog:read" が含まれる
- THEN `group_roles` は "catalog:read" を含み、`direct_roles` は空である

### REQ-IDMANAGEMENT-016: ユーザーは自分のプロフィール表示名を更新できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでマイアカウントのプロフィールを開いている
- WHEN ユーザー "alice" が表示名を更新する
- THEN 更新後のプロフィールに新しい表示名が反映される
- THEN `editable_by_user=false` の属性は更新できない

### REQ-IDMANAGEMENT-017: ユーザーはメールアドレス変更を起票し確認リンクで確定できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでメールアドレス画面を開いている
- WHEN ユーザー "alice" が新しいメールアドレスへの変更を起票する
- THEN 新アドレスへ確認リンクが送られる
- WHEN ユーザー "alice" が確認リンクのトークンで変更を確定する
- THEN プライマリメールアドレスが新しいアドレスへ更新される

### REQ-IDMANAGEMENT-018: ユーザーは自分のアカウントデータをエクスポートできる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでデータとプライバシー画面を開いている
- WHEN ユーザー "alice" がアカウントデータをエクスポートする
- THEN レスポンスに自分のプロフィールと同意の一覧が含まれる

### REQ-IDMANAGEMENT-019: アカウント API は他人のリソースを返さない
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みである
- WHEN ユーザー "alice" のアカウント概要を取得する
- THEN レスポンスは "alice" 自身のデータだけを含み、ロールは含まない

### REQ-IDMANAGEMENT-020: 管理者は CEL の規則で動的グループの所属を管理できる
- ACTOR TenantAdministrator
- GIVEN `department` 属性が定義され、Engineering の User と Sales の User が存在する
- GIVEN `membership_type=dynamic` のグループが存在する
- WHEN 管理者が `user.department == "Engineering"` を保存して有効化する
- THEN 全件の再評価後、Engineering の有効な User だけが動的規則を由来として所属する
- THEN 実効ロールと Application の割り当ては、その所属を参照する

### REQ-IDMANAGEMENT-021: CEL の規則は保存前に選んだユーザーでプレビューできる
- ACTOR TenantAdministrator
- GIVEN 管理者が最大 100 件の User を選択している
- WHEN 未保存の CEL 式を評価する
- THEN レスポンスは一致の有無と、追加・削除・変更なしの判定を返し、属性値そのものは返さない

### REQ-IDMANAGEMENT-022: 不正な CEL の規則と動的グループの手動操作は拒否される
- ACTOR TenantAdministrator
- WHEN 管理者が、未定義の属性または許可外の関数を参照する CEL 式を保存する
- THEN 保存は拒否される
- WHEN 管理者が動的グループに対して `AddGroupMember` または `RemoveGroupMember` を手動で呼ぶ
- THEN メンバーシップの変更は拒否される

### REQ-IDMANAGEMENT-023: 評価できない規則は権限を付与しない
- ACTOR TenantAdministrator
- GIVEN 有効な動的規則のバージョンが更新された
- WHEN 新しいバージョンの動的規則を再評価する
- THEN 旧バージョンのメンバーシップは直ちに実効ロールから除外される
- THEN 再評価に失敗した User は、新しいバージョンのメンバーシップを得ない

### REQ-IDMANAGEMENT-024: 管理者はグループの連絡先メールとカスタム属性を、テナント定義のスキーマに従って設定できる
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- GIVEN テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- WHEN "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
  - ALT `email` がメールアドレスの形式を満たさない → 作成は InvalidEmailError で拒否される
  - ALT `attributes` に未定義のキーを指定する、または定義済みのキーと型が一致しない → 作成は InvalidGroupAttributeError で拒否される
- THEN 作成したグループの `email` と `attributes` が指定どおりに保存され、"GroupCreated" が発行される
- WHEN "operator" が同じグループの `email` と `attributes` を更新する
- THEN 更新後のグループに新しい値が反映され、"GroupUpdated" の `changed_fields` に "email" と "attributes" が含まれる

### REQ-IDMANAGEMENT-025: 管理 API クライアントはプリンシパルの種類と操作の粒度でだけ User / Group / Agent を操作できる
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API アクセストークンを提示している
- GIVEN トークンの発行者は対象操作に必要なロールを今も持っている
- WHEN クライアントが User、Group、または Agent の操作をリクエストする
  - ALT `users:read` だけで User の変更または CSV インポートを要求する → 操作は AccessDeniedError で拒否される
  - ALT `users:*` だけで Group または Agent の操作を要求する → 操作は AccessDeniedError で拒否される
  - ALT `agents:read` だけで Agent のキルまたは削除を要求する → 操作は AccessDeniedError で拒否される
- THEN `users:read` は User の参照に加えて、User を変更しない CSV エクスポートの開始、参照、ダウンロード、取り消しを許可する
- THEN `groups:read` は Group の参照に加えて、Group を変更しない動的グループ規則のプレビューと CSV エクスポートを許可する
- THEN `agents:write` は Agent のキルと削除の両方を許可する
