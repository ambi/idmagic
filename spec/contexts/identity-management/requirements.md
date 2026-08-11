# IdManagement Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-IDMANAGEMENT-001: federated JITはpassword credentialを作らずactive Userを作成する
- Actor: EndUser
- Given: Authentication context が upstream token/assertion と tenant JIT policy を検証済みである
- Then: ProvisionFederatedUser は mapped username、任意の name/email/attributes を検証する
- Then: tenant quota と一意性を検証する
- Then: password_hash が null の active User を作成して UserCreated を発行する
- Alternative (username/email が衝突する、quota 超過、または属性 schema が不正である): User を作成せずエラーを返す

### REQ-IDMANAGEMENT-002: API token発行者はaccount scope内で自身のidentity情報だけを操作できる
- Actor: SelfApiClient
- Given: client は対象 tenant の active User に固定された有効な API access token を提示している
- Then: account:read scope で自身の summary、profile、data export を参照できる
- Then: account:write scope で自身の profile と primary email change request を変更できる
- Alternative (account:read だけで変更操作を要求する): 操作は AccessDeniedError で拒否される
- Alternative (token の tenant または user_id が操作対象と一致しない): 操作は AccessDeniedError で拒否される

### REQ-IDMANAGEMENT-003: email verification画面は未認証でもCSRF境界を確立できる
- Actor: EndUser
- Then: EndUser が email verification context を取得する
- Then: 応答の CSRF token と SameSite cookie を使って email verification を送信できる
- Alternative (CSRF token と cookie が一致しない): email verification は InvalidRequestError で拒否される

### REQ-IDMANAGEMENT-004: 管理者は CSV を検証して有効な行だけをインポートできる
- Actor: TenantAdministrator
- Given: roles=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- Then: 管理者が machine-key header [id,email,roles,custom:department] を任意順で含む CSV を事前検証へ投入する
- Then: preview job は created / updated / unchanged / rejected と行番号・stable error code を返し、User は変更されない
- Then: 管理者が同一 tenant の成功済み preview job id を指定して apply を開始し、CSV は再送しない
- Then: apply は preview payload と SHA-256 を検証し、現在の repository 状態に対して同じ planner で再計画する
- Then: 有効行は create または update され、無効行は rejected として残り、各行の profile・roles・required actions・custom attributes は原子的に保存される
- Alternative (CSV が実効 UserCsvTransferPolicy の max_bytes、max_rows、max_field_bytes のいずれかを超える): インポート投入は拒否される → エラー "csv_too_large" または "too_many_rows" または "field_too_large"
- Alternative (CSV のヘッダーに未知列、重複列、password または password_hash が含まれる): インポート投入は拒否される → エラー "invalid_header"
- Alternative (行の id と preferred_username が別 User を示す、識別子が無い、同一対象または同一最終 username を複数行が示す): 対象行は rejected となり、stable error code を返す
- Alternative (preview job が存在しない、queued/failed、別 tenant、または保存 payload と digest が一致しない): apply は User を変更せず InvalidRequestError または AccessDeniedError で拒否される
- Alternative (preview 後に対象 User の状態が別操作で変更されている): apply は stale な preview plan を実行せず、現在状態から updated / unchanged / rejected を再判定する
- Alternative (対象 User が source-managed である): 対象行は source_managed の stable error code で rejected となり、User は変更されない
- Alternative (1 行の validation、保存、または監査処理が途中で失敗する): その行の profile・roles・required actions・custom attributes は一部も保存されず、別の有効行は適用を継続する

### REQ-IDMANAGEMENT-005: 管理者はユーザー一覧をページングしながら安定して閲覧できる
- Actor: TenantAdministrator
- Given: 所属テナントに limit を超えるユーザーが存在する
- Then: 管理者が ListAdminUsers を limit のみで実行して先頭ページを取得する
- Then: 応答は filter に一致する exact total items / total pages / current page / page size と filter 非依存の total_users を返す
- Then: 応答の Link response header (rel="next") から compact cursor を取得する
- Then: 一覧の途中で他の管理者がユーザーを1件削除する
- Then: 管理者が取得済みの cursor で次ページを取得する
- Then: 削除された行を除き、既に返却済みの行との重複なく残りのユーザーが返る
- Then: 応答の Link response header (rel="prev") から前ページの cursor を取得する
- Then: 管理者が前ページへ戻り、そのページの canonical order でユーザーを閲覧する
- Then: 管理者が rel="last" の end anchor cursor で端数を含む最終ページへ移動する
- Then: 管理者が rel="first" の cursor を含まない URL で先頭ページへ戻る
- Alternative (cursor が別テナントで発行された、改ざんされた、または query/status が発行時と異なる): ListAdminUsers は InvalidRequestError を返す → 管理者は先頭ページへ戻って再取得する
- Alternative (query または status を指定する): ListAdminUsers は tenant 全体を対象に条件へ一致する User だけを返す → pagination total_items は条件一致件数、total_users は削除済みを除く filter 非依存件数を返す → query/status を変更した管理者は cursor を破棄して先頭ページから取得する
- Alternative (条件に一致する User が 0 件である): users は空で total items / total pages / current page は 0 / 0 / 0 を返す → first / prev / next / last Link は返さない
- Alternative (exact count の取得に失敗する): 0 件として成功させず request 全体を server error で失敗させる
- Alternative (実行者が TenantAdministrator ロールを持たない): ListAdminUsers は AccessDeniedError で拒否される

### REQ-IDMANAGEMENT-006: 管理者はユーザー一覧を CSV に安全にエクスポートできる
- Actor: TenantAdministrator
- Given: roles=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- Given: 一覧には自テナントのユーザーが存在する
- Then: 管理者が列 [preferred_username, email] と status フィルタで /users/exports へエクスポートを開始する
- Then: エクスポートは 202 とエクスポート id を返し、ジョブは queued である
- Then: worker が生成を開始し DataExportStarted が発行される
- Then: 生成が完了して status は succeeded、downloadable は true、total_rows と byte_size が記録される
- Then: DataExportSucceeded が発行される
- Then: 管理者がファイルをダウンロードすると選択した machine key と一致する header の RFC 4180 CSV が content-disposition attachment で返り、DataExportDownloaded が発行される
- Alternative (選択列に User allowlist 外の key (例 password_hash) が含まれる): エクスポート開始は InvalidRequestError で拒否される → エラー "invalid_columns"
- Alternative (セル値が \"=\", \"+\", \"-\", \"@\", タブ, CR, LF のいずれかで始まる): 値は formula injection を避ける可逆 prefix で出力され、import decoder は規定どおり prefix 1 文字だけを戻す
- Alternative (生成が失敗する): status は failed、downloadable は false、error_code が記録される → DataExportFailed が発行される → 不完全ファイルはダウンロードできない
- Alternative (終端前に管理者が取消す): status は canceled になり、DataExportCanceled が発行される
- Alternative (保持期限を経過している): status は expired、downloadable は false → ファイル本体は purge され、ダウンロードは InvalidRequestError で拒否される
- Alternative (User エクスポートの id を /groups/exports や別テナントで指定する): 取得・ダウンロード・取消は AccessDeniedError または InvalidRequestError で拒否される (per-type / per-tenant 分離)

### REQ-IDMANAGEMENT-007: 管理者はエクスポートしたユーザー CSV を安全に再適用できる
- Actor: TenantAdministrator
- Given: 実効 TenantUserAttributeSchema に custom:department があり、10,000 User を含む一覧が実効 UserCsvTransferPolicy 内に収まる
- Then: 管理者が import-compatible な組み込み列、required_actions、custom:department を machine-key header でエクスポートする
- Then: worker は CSV を immutable artifact store へ streaming 出力し、job result には tenant-scoped payload reference、server-computed SHA-256、size、row count を保持する
- Then: 管理者が同じ 10,000 行 artifact を無編集で preview すると全行 unchanged となり、User は変更されない
- Then: 管理者が 1 行の email と custom:department だけを編集して再度 preview する
- Then: 変更行だけが updated、残りは unchanged と計画される
- Then: 管理者が成功済み preview job id を apply すると、指定した writable 列だけが更新され、未指定列は維持される
- Alternative (値が危険な先頭文字、既存 apostrophe、comma、quote、または改行を含む): reversible formula-safe codec と RFC 4180 quoting により decode(encode(value)) は元の値と一致する
- Alternative (生成結果が実効 UserCsvTransferPolicy のいずれかの上限を超える): User export は csv_transfer_limit_exceeded で失敗し、再 import 不能な成功 artifact を作らない → 管理者は filter または列を絞って複数 artifact に分割できる
- Alternative (status、mfa_enrolled、created_at、updated_at、または id の値だけを編集する): 読み取り専用列は受理して無視し、writable 列に差分がなければ unchanged とする
- Alternative (custom 属性の型、boolean、number、date、required custom 属性、または required_actions が不正である): 対象行は stable error code で rejected となり、値を job view、error、audit event に含めない

### REQ-IDMANAGEMENT-008: 管理者は特定グループのメンバー一覧を CSV にエクスポートできる
- Actor: TenantAdministrator
- Given: roles=["admin"] のユーザー "operator" がグループ "engineering" の詳細を開いている
- Then: 管理者が /groups/{group_id}/members/exports へ列 [user_id, preferred_username] でエクスポートを開始する
- Then: エクスポートは group_id で scope され、そのグループのメンバーだけが対象になる
- Then: 生成完了後、管理者がメンバー CSV をダウンロードする
- Alternative (group_id を指定しない (per-group 必須)): エクスポート開始は InvalidRequestError で拒否される
- Alternative (別グループの path でそのエクスポート id を指定する): 取得・ダウンロードは InvalidRequestError で拒否される (per-group 分離)

### REQ-IDMANAGEMENT-009: 管理者はエージェントを登録し client 資格情報をバインドできる
- Actor: TenantAdministrator
- Given: roles=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- Then: 管理者 "operator" がエージェント "batch-agent" を登録する
- Then: 管理者 "operator" がエージェント "batch-agent" に client 資格情報をバインドする
- Then: 管理者 "operator" がエージェント "batch-agent" を無効化する
- Then: 管理者 "operator" がエージェント "batch-agent" を再有効化する
- Then: エージェント一覧に "batch-agent" が表示される
- Alternative (別テナントの client 資格情報をバインドする): tenant_id "acme" の Agent に tenant_id "default" の client_id を指定する → エラー "InvalidRequestError"

### REQ-IDMANAGEMENT-010: 管理者は無効化したユーザーを再有効化できる
- Actor: TenantAdministrator
- Given: 管理者がユーザー "alice" を無効化している
- Then: 管理者がユーザー "alice" を再有効化する
- Then: ユーザー "alice" の status は Active である
- Then: "UserEnabled" が発行される

### REQ-IDMANAGEMENT-011: 管理者はユーザーを soft-delete し 猶予期間内に復元できる
- Actor: TenantAdministrator
- Given: roles=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- Given: ユーザー "alice" は Active である
- Then: 管理者 "operator" がユーザー "alice" を削除する
- Then: ユーザー "alice" の status は PendingDeletion である
- Then: "UserSoftDeleted" が発行される
- Then: 管理者 "operator" がユーザー "alice" を復元する
- Then: ユーザー "alice" の status は Active である
- Then: "UserRestored" が発行される

### REQ-IDMANAGEMENT-012: soft-delete されたユーザーはログインを拒否される
- Actor: EndUser
- Given: ユーザー "alice" は PendingDeletion である
- Then: ユーザー "alice" が正しいパスワードでログインを試みる
- Then: ログインは拒否される

### REQ-IDMANAGEMENT-013: 管理者はユーザーを完全削除できる
- Actor: TenantAdministrator
- Given: ユーザー "alice" は PendingDeletion である
- Then: 管理者がユーザー "alice" を完全削除する
- Then: ユーザー "alice" の status は Deleted である
- Then: "UserDeleted" が発行される
- Alternative (対象が admin 自身である): soft-delete / 復元 / 完全削除のいずれも拒否される → エラー "self_delete_forbidden"

### REQ-IDMANAGEMENT-014: ロールに応じて管理APIのアクセスが制御される
- Actor: TenantAdministrator
- Given: roles に "admin" を持つユーザー "operator" が認証済みである
- Then: 管理者 "operator" が preferred_username "bob" のユーザーを作成する
- Then: 管理者 "operator" がユーザー一覧を取得する
- Then: 応答にユーザー "bob" が含まれる
- Then: "UserCreated" が発行される
- Alternative (admin ロールを持たないユーザーが管理 API を呼ぶ): roles が空のユーザー "alice" が認証済みである → ユーザー "alice" がユーザー一覧を取得する → エラー "AccessDeniedError"

### REQ-IDMANAGEMENT-015: 管理者はグループを作成しユーザーを所属させると有効ロールにグループ由来ロールが乗る
- Actor: TenantAdministrator
- Given: roles=["admin"] のユーザー "operator" が認証済みである
- Given: roles が空のユーザー "alice" が同一テナントに存在する
- Then: 管理者 "operator" が roles=["catalog:read"] のグループ "engineering" を作成する
- Then: "GroupCreated" が発行される
- Then: 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
- Then: "GroupMemberAdded" が発行される
- Then: ユーザー "alice" の所属グループを取得すると effective_roles に "catalog:read" が含まれる
- Then: group_roles は "catalog:read" を含み direct_roles は空である
- Alternative (同じユーザーを同じグループへ再度所属させる): 管理者 "operator" がユーザー "alice" をグループ "engineering" に再度所属させる → "GroupMemberAdded" は再発行されない

### REQ-IDMANAGEMENT-016: ユーザーは自分のプロフィール表示名を更新できる
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が認証済みでマイアカウントのプロフィールを開いている
- Then: ユーザー "alice" が表示名を更新する
- Then: 更新後のプロフィールに新しい表示名が反映される
- Then: editable_by_user=false の属性は更新できない

### REQ-IDMANAGEMENT-017: ユーザーはメールアドレス変更を起票し確認リンクで確定できる
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が認証済みでメールアドレス画面を開いている
- Then: ユーザー "alice" が新しいメールアドレスへの変更を起票する
- Then: 新アドレスへ確認リンクが送られる
- Then: ユーザー "alice" が確認リンクのトークンで変更を確定する
- Then: primary email が新しいアドレスへ更新される

### REQ-IDMANAGEMENT-018: ユーザーは自分のアカウントデータをエクスポートできる
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が認証済みでデータとプライバシー画面を開いている
- Then: ユーザー "alice" がアカウントデータをエクスポートする
- Then: 応答に自分の profile と consents が含まれる

### REQ-IDMANAGEMENT-019: マイアカウントAPIは他人のリソースを返さない
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が認証済みである
- Then: ユーザー "alice" のマイアカウント概要を取得する
- Then: 応答は "alice" 自身のデータだけで roles を含まない

### REQ-IDMANAGEMENT-020: 管理者はCELルールで動的グループ所属を管理できる
- Actor: TenantAdministrator
- Given: department 属性が定義され Engineering の User と Sales の User が存在する
- Given: membership_type=dynamic のグループが存在する
- Then: 管理者が `user.department == "Engineering"` を保存して有効化する
- Then: 全件再評価後に Engineering の Active User だけが dynamic_rule source で所属する
- Then: effective_roles と application assignment はその所属を参照する

### REQ-IDMANAGEMENT-021: CELルールは保存前に選択ユーザーでプレビューできる
- Actor: TenantAdministrator
- Given: 管理者が最大100 Userを選択している
- Then: 未保存 CEL を評価する
- Then: 応答は matched と add/remove/unchanged を返し属性値を返さない

### REQ-IDMANAGEMENT-022: 不正なCELルールと動的グループの手動操作は拒否される
- Actor: TenantAdministrator
- Then: 未定義属性または許可外関数を参照する CEL の保存は拒否される
- Then: dynamic group への手動 AddGroupMember と RemoveGroupMember は拒否される

### REQ-IDMANAGEMENT-023: 評価不能なルールは権限を付与しない
- Actor: TenantAdministrator
- Given: 有効な dynamic rule の version が更新された
- Then: 旧 version の membership は直ちに effective roles から除外される
- Then: 再評価が失敗した User は新 version の membership を取得しない - "応答は全 action を blocked、reason \"trigger_not_matched\" として返す"

### REQ-IDMANAGEMENT-024: GetEmailVerificationContext
未認証で開かれる email verification 画面が確認 POST を保護する CSRF token を取得する。

### REQ-IDMANAGEMENT-025: GetAccountSummary
認証済みユーザーが自身のアカウント概要を取得する (self-service portal home)。
対象は subject.id に固定。roles を含まず、最終ログイン・パスワード変更日時・
MFA 状態・未対応 required actions を返す。admin shell の AccountContext とは
別契約。

### REQ-IDMANAGEMENT-026: GetUserProfile
認証済みユーザーが自身のプロフィールを取得する (self-service)。対象は
subject.id に固定するため cross-user 参照は構造的に発生しない。attributes は
self が読める分のみ、editable_attributes に編集可能定義を併せて返す。

### REQ-IDMANAGEMENT-027: UpdateUserProfile
認証済みユーザーが自身の表示名と editable_by_user=true の属性を更新する
(self-service)。subject.id == resource.id に固定。属性は key 単位の merge とし、
editable_by_user=false の key は拒否する。status / roles / organization の変更は
admin 専用で本経路では扱わない。

### REQ-IDMANAGEMENT-028: RequestEmailChange
認証済みユーザーが自身の primary email 変更を起票する (self-service)。新アドレスへ
ワンタイムリンクを送るだけで User.email は変えない。subject.id に固定し、tenant 内で
既に使われているアドレスは拒否する。
確認メールの文面は Tenancy の通知テンプレートカタログ
(template_key=EmailChangeConfirmation) から解決し、locale は受信者 User の locale 属性 →
テナントの default_locale → システム既定 locale の順に決める。テナント上書きがあれば
組込み既定より優先する。プレーンテキストと HTML の両方を必ず生成し、確認リンクは
リクエストの発行元 URL から組み立てる。

### REQ-IDMANAGEMENT-029: ConfirmEmailChange
新アドレスへ送ったワンタイムリンクを消費して primary email を確定する。
token が新アドレス所有の証左なので認証済みセッションは要求しない。token の
所持そのものが認可の根拠であり (self-service セッションではなく
token-possession-authenticated)、AuthenticatedSelf/session ベースの access 分類は
適用しない。確定時に email_verified=true とし、verify_email の required action が
あれば自動解除する。

### REQ-IDMANAGEMENT-030: ExportAccountData
認証済みユーザーが自身の個人データを JSON で取得する (self-service, GDPR access)。
対象は subject.id に固定。本ステージは同期生成で profile と consents を含む。

### REQ-IDMANAGEMENT-031: ListAdminUsers
管理者が削除されていないユーザーを preferred_username 昇順 (同値は id で tie-break) の
双方向 keyset pagination で一覧する。query は username、name、email、id、roles を tenant 全体で
case-insensitive contains 検索し、status とともに絞り込む。cursor は応答の Link response header
(rel="prev" / rel="next") から取得する。

### REQ-IDMANAGEMENT-032: GetAdminUser
管理者が id を指定してユーザーを取得する。

### REQ-IDMANAGEMENT-033: CreateAdminUser
管理者が Authentication context の PasswordPolicy を満たす初期パスワードでユーザーを作成する。

### REQ-IDMANAGEMENT-034: ProvisionFederatedUser
Authentication context が、検証済み upstream identity と tenant の明示 JIT policy / claim mapping に基づき password credential を持たない active User を作成する内部 published interface。tenant quota、username/email 一意性、属性 schema、UserCreated event は通常作成と同じ契約を適用する。
- Postcondition: output.user.password_hash == null
- Postcondition: output.user.lifecycle.status == Active

### REQ-IDMANAGEMENT-035: ImportAdminUsers
管理者が UTF-8 (BOM 可) CSV を streaming upload して副作用のない preview ジョブへ投入する。User export と共有する configurable transfer policy の既定は 100,000 data rows・64 MiB/file・64 KiB/field。payload は immutable artifact store に保存し、job params には tenant-scoped reference・server-computed SHA-256・size だけを保持する。ヘッダーは安定した機械キーの任意順・任意部分集合を許可し、未知・重複・password/hash 列をファイル単位で拒否する。
- Precondition: withinUserCsvTransferPolicy(input.csv)
- Postcondition: usersUnchanged(context.tenant_id)

### REQ-IDMANAGEMENT-036: ApplyAdminUserImport
管理者が同一 tenant の成功済み preview job を指定して CSV を適用する。CSV は再送せず、preview payload と source_sha256 を読み、現在の repository 状態に対して同じ deterministic planner で再計画する。ファイル全体は行単位部分成功、各行の profile・roles・required actions・custom attributes は単一 aggregate mutation として原子的に確定する。
- Precondition: previewJobTenant(input.preview_job_id) == context.tenant_id
- Precondition: previewJobStatus(input.preview_job_id) == 'succeeded'
- Precondition: previewJobDigestMatchesPayload(input.preview_job_id)
- Postcondition: everyAppliedImportRowIsAtomic(output.job.id)

### REQ-IDMANAGEMENT-037: GetAdminUserImport
管理者が同一 tenant の CSV インポートジョブと行ごとの検証・適用結果を取得する。row error は immutable result artifact に固定件数の page chunk として保存し、tenant・job・error ordinal に結合した署名付き opaque cursor と limit（既定 100、最大 200）で双方向取得する。job result JSON は操作別件数、result artifact metadata、error_total だけを保持する。

### REQ-IDMANAGEMENT-038: StartUserCsvExport
管理者が User 一覧の CSV エクスポートを非同期ジョブとして開始する。CSV ヘッダーは表示 label ではなく要求した machine key と一致する。列は組み込み User allowlist (13 個のコア列 + attr:<key> の組み込み拡張属性 27 種) と実効 TenantUserAttributeSchema の custom:<key> (テナント定義属性) の部分集合に限り、status フィルタを引き継げる。生成 payload は import と同じ UserCsvTransferPolicy と immutable artifact store を使うため、成功した User export は無編集で preview できる。202 とエクスポート id を返す。
- Precondition: size(input.request.columns) >= 1

### REQ-IDMANAGEMENT-039: ListUserExports
管理者が User エクスポート一覧を、状態・件数・期限・ダウンロード可否とともに取得する。CSV 本体や PII 値は含まない。

### REQ-IDMANAGEMENT-040: GetUserExport
管理者が User エクスポート 1 件の状態・進捗・失敗理由・期限を取得する。

### REQ-IDMANAGEMENT-041: DownloadUserExportFile
管理者が完成済み User エクスポートを content-disposition attachment でダウンロードする。succeeded かつ未期限のジョブに限る。

### REQ-IDMANAGEMENT-042: CancelUserExport
管理者が終端前の User エクスポートを取消す。既に終端の場合は拒否する。

### REQ-IDMANAGEMENT-043: StartGroupCsvExport
管理者が Group 一覧の CSV エクスポートを非同期ジョブとして開始する。列は Group allowlist の部分集合に限る。202 とエクスポート id を返す。
- Precondition: size(input.request.columns) >= 1

### REQ-IDMANAGEMENT-044: ListGroupExports
管理者が Group エクスポート一覧を、状態・件数・期限・ダウンロード可否とともに取得する。

### REQ-IDMANAGEMENT-045: GetGroupExport
管理者が Group エクスポート 1 件の状態・進捗・失敗理由・期限を取得する。

### REQ-IDMANAGEMENT-046: DownloadGroupExportFile
管理者が完成済み Group エクスポートを content-disposition attachment でダウンロードする。succeeded かつ未期限のジョブに限る。

### REQ-IDMANAGEMENT-047: CancelGroupExport
管理者が終端前の Group エクスポートを取消す。既に終端の場合は拒否する。

### REQ-IDMANAGEMENT-048: StartGroupMemberCsvExport
管理者が特定 Group のメンバー一覧の CSV エクスポートを開始する。group_id は path から取り、そのグループのメンバーだけを対象とする (Entra / Okta / Google と同じ per-group)。202 とエクスポート id を返す。
- Precondition: size(input.request.columns) >= 1

### REQ-IDMANAGEMENT-049: ListGroupMemberExports
管理者が特定 Group のメンバーエクスポート一覧を取得する。

### REQ-IDMANAGEMENT-050: GetGroupMemberExport
管理者が特定 Group のメンバーエクスポート 1 件の状態を取得する。別グループの export id は解決しない。

### REQ-IDMANAGEMENT-051: DownloadGroupMemberExportFile
管理者が完成済みメンバーエクスポートを content-disposition attachment でダウンロードする。succeeded かつ未期限のジョブに限る。

### REQ-IDMANAGEMENT-052: CancelGroupMemberExport
管理者が終端前のメンバーエクスポートを取消す。既に終端の場合は拒否する。

### REQ-IDMANAGEMENT-053: UpdateAdminUser
管理者がユーザーのプロフィール、email_verified、roles を更新する。

### REQ-IDMANAGEMENT-054: DisableAdminUser
管理者がユーザーを無効化し、新規ログインと既存セッション利用を拒否する。

### REQ-IDMANAGEMENT-055: EnableAdminUser
管理者が無効化済みユーザーを再有効化する。

### REQ-IDMANAGEMENT-056: DeleteAdminUser
管理者がユーザーを削除する。既定は SoftDelete で、status を
PendingDeletion に遷移させ PII / Consent / RefreshToken / Session を温存する
(猶予期間内は Restore 可能)。`purge=true` の場合は anonymize
cascade を即時実施し tombstone 化する。いずれも冪等で、既に同じ終着状態の
user への再呼び出しは no-op として 204 を返す。自分自身は削除できない
(admin / system_admin の自爆防止)。
- Precondition: subject.id != resource.id

### REQ-IDMANAGEMENT-057: RestoreAdminUser
管理者が PendingDeletion のユーザーを Restore する。status を Active
に戻し PII / credential を温存したまま復元する。猶予期間を過ぎている、または
PendingDeletion でない user に対しては InvalidRequestError を返す。復元後の
user を返す。自分自身への操作は use case 側で reject する (自爆防止と対称の
扱い)。
- Precondition: subject.id != resource.id

### REQ-IDMANAGEMENT-058: SetUserRequiredAction
管理者が対象ユーザーに次回ログイン時の強制アクション (RequiredAction) を
付与する。update_password を付与されたユーザーは次回ログイン時に
change-password 画面へ強制誘導される (本人の変更で自動解除)。冪等で、既に
付与済みなら現状を返す。admin 専用。

### REQ-IDMANAGEMENT-059: ClearUserRequiredAction
管理者が対象ユーザーの強制アクションを解除する。冪等で、未付与なら
現状を返す。admin 専用。本人のパスワード変更に伴う update_password の自動
解除は ChangePassword / ResetPasswordWithToken 側で発火する。

### REQ-IDMANAGEMENT-060: ListGroups
管理者が所属テナントのグループを name 昇順 (同値は id で tie-break) の双方向 keyset
pagination で一覧する。各グループは member_count を含む。cursor は応答の Link response header
(rel="prev" / rel="next") から取得する。

### REQ-IDMANAGEMENT-061: GetGroup
管理者が単一グループとそのメンバー一覧を取得する。別テナントのグループは未存在として扱う。

### REQ-IDMANAGEMENT-062: CreateGroup
管理者が所属テナントにグループを作成する。name はテナント内で一意。
- Postcondition: emitted.exists(e, e is GroupCreated)

### REQ-IDMANAGEMENT-063: UpdateGroup
管理者がグループの name / description / roles を更新する。
- Postcondition: changedFields(input.request).size() > 0 ? emitted.exists(e, e is GroupUpdated) : !emitted.exists(e, e is GroupUpdated)

### REQ-IDMANAGEMENT-064: DeleteGroup
管理者がグループを削除する。所属 membership は cascade で解除する。
- Postcondition: emitted.exists(e, e is GroupDeleted)
- Postcondition: emitted.filter(e, e is GroupMemberRemoved).size() == membersOf(input.group_id).size()

### REQ-IDMANAGEMENT-065: AddGroupMember
管理者が User をグループに所属させる。同一テナントの User のみ許可し、既所属なら no-op (冪等)。
- Postcondition: alreadyMember(input.group_id, input.user_id) ? !emitted.exists(e, e is GroupMemberAdded) : emitted.exists(e, e is GroupMemberAdded)

### REQ-IDMANAGEMENT-066: RemoveGroupMember
管理者が User をグループから外す。非所属なら no-op (冪等)。
- Postcondition: alreadyMember(input.group_id, input.user_id) ? emitted.exists(e, e is GroupMemberRemoved) : !emitted.exists(e, e is GroupMemberRemoved)

### REQ-IDMANAGEMENT-067: UpdateDynamicGroupRule
管理者が dynamic group の制限 CEL rule を検証して保存する。未知属性、型不一致、非 Boolean、許可外構文、上限超過は拒否する。
- Precondition: group(input.group_id).membership_type == dynamic

### REQ-IDMANAGEMENT-068: PreviewDynamicGroupRule
最大100 Userに未保存 CEL を副作用なしで評価し、現在所属との差分だけを返す。
- Precondition: input.request.user_ids.size() <= 100

### REQ-IDMANAGEMENT-069: EnableDynamicGroupRule
検証済み rule を有効化し、旧 version membership を即時無効化して全件再評価 Job を予約する。

### REQ-IDMANAGEMENT-070: DisableDynamicGroupRule
rule を無効化して、全 dynamic membership を認可・割当判定から即時除外する。

### REQ-IDMANAGEMENT-071: ListUserGroups
管理者が指定 User の所属グループと有効ロール内訳 (明示 / グループ由来 / union) を取得する。

### REQ-IDMANAGEMENT-072: ListAgents
管理者が所属テナントの Agent を name 昇順 (同値は id で tie-break) の双方向 keyset
pagination で一覧する。cursor は応答の Link response header (rel="prev" / rel="next") から取得する。

### REQ-IDMANAGEMENT-073: GetAgent
管理者が単一 Agent を取得する。別テナントの Agent は未存在として扱う。

### REQ-IDMANAGEMENT-074: RegisterAgent
管理者が所属テナントに Agent を登録する。name はテナント内で一意、owner_user_id 省略時は actor を所有者とする。
- Precondition: input.request.owner_user_id == null || input.request.owner_user_id in tenantUserIds(context.tenant_id)

### REQ-IDMANAGEMENT-075: UpdateAgent
管理者が Agent の name / description / kind / owner_user_id / roles を更新する。owner_user_id の変更は所有権移転。
- Precondition: input.request.owner_user_id == null || input.request.owner_user_id in tenantUserIds(context.tenant_id)

### REQ-IDMANAGEMENT-076: DisableAgent
管理者が Agent を可逆に運用停止する。

### REQ-IDMANAGEMENT-077: EnableAgent
管理者が運用停止中の Agent を再稼働する。

### REQ-IDMANAGEMENT-078: KillAgent
管理者が Agent を緊急停止する (一方向終端、復帰不能)。

### REQ-IDMANAGEMENT-079: DeleteAgent
管理者が Agent を削除する。束縛は cascade で解除する。Killed の Agent は削除できない (fail-closed。削除により束縛先 OAuth2Client が client_credentials で無制限に token を再取得できてしまうため)。
- Precondition: resource.status != "Killed"

### REQ-IDMANAGEMENT-080: BindAgentCredential
管理者が Agent に OAuth2Client を束縛する。
- Precondition: not agentCredentialBindingExistsForClient(context.tenant_id, input.request.client_id)

### REQ-IDMANAGEMENT-081: UnbindAgentCredential
管理者が Agent から OAuth2Client の束縛を解除する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Administrator | User.roles に admin を持ち、所属テナント内の管理 API を許可された認証済みユーザー。テナント境界を越える操作は SystemAdministrator に限定される。 | admin, 管理者, TenantAdmin |
| SystemAdministrator | User.roles に system_admin を持つ認証済みユーザー。テナント管理 (CRUD・disable・enable) と cross-tenant 操作を許可され、system_admin 専用のシステムコンソール (/system) から `/api/admin/tenants/*` や `/api/admin/keys/health` を呼び出せる。テナント境界を越えるため path ではなく role でゲートする。 | system_admin, システム管理者 |
| EndUser | 認証済みまたは認証を試みる一般利用者。管理ロールを持たない自己サービス操作 (account portal) の主体。 | end user, 利用者, エンドユーザー |
| UserDisablement | User.status を Disabled に遷移させて認証とセッション利用を停止する復活可能な管理操作。削除や PII purge とは異なる。 | disable user |
| UserImport | 管理者が UTF-8 CSV を使ってユーザーの作成・部分更新を事前検証し、成功済み preview に結合して非同期適用する操作。CSV は安定した機械キーのヘッダーを任意順・任意部分集合で持ち、パスワードや password_hash を含めない。 | CSV user import, bulk user import, ユーザー一括インポート |
| UserDeletion | User の tombstone 化と関連 aggregate の cascade 削除。`status` を Deleted に遷移させ PII フィールドを匿名化、`id` のみを audit のため保持する。`Deleted` は終端で復元できない。 | delete user, anonymize user, アカウント削除 |
| Deleted | User の終端状態。`status == Deleted` で PII が anonymize 済み。login / token / userinfo は active=false 相当。 | deleted |
| Delete | User を Deleted に遷移させる管理操作。tombstone 化と cascade を 1 オペレーションで実施する。 | delete |
| PendingDeletion | User の削除予約状態。`status == PendingDeletion` で PII は温存されるが認証は Disabled と同じく拒否される。猶予期間 (states.UserLifecycle の PendingDeletion → Deleted 遷移 guard) 内なら Restore で Active に戻せ、経過すると Purge で Deleted (anonymize) に落ちる。 | pending_deletion, 削除予約中 |
| SoftDelete | User を Active / Disabled から PendingDeletion に遷移させる管理操作。PII / Consent / RefreshToken / Session を残したまま削除を予約し、誤操作を 猶予期間内で救済できる。 | soft_delete, soft-delete |
| Restore | PendingDeletion の User を Active に戻す管理操作。猶予期間内でのみ可能で、PII や credential は温存されているためログインは通常どおり再開する。 | restore |
| Purge | User を Active / Disabled / PendingDeletion から Deleted に遷移させる確定削除操作。anonymize cascade を実行し、猶予期間経過後の自動 purge と admin の明示的完全削除の双方から呼ばれる。 | purge |
| Group | tenant-scoped 集約。再利用可能なロール束 (roles[]) を持ち、所属する User にそのロールを一斉付与する。階層・deny ルール・属性自動所属は持たない (union のみ)。 | group, グループ, role group, ロールグループ |
| GroupMembership | User と Group の所属関係 (GroupMember)。manual は管理者操作、dynamic は有効な CEL rule の評価結果だけから変更される。effective_roles(user) = user.roles ∪ ⋃ membership.group.roles。 | group membership, グループ所属, membership |
| DynamicGroupRule | User の core 属性と TenantUserAttributeSchema で定義された属性だけを参照し、所属可否を Boolean で返す制限 CEL 式。rule version が一致する dynamic membership だけが有効になる。 | dynamic membership rule, 動的グループルール |
| EffectiveRoles | 認可判断で用いる User の有効ロール集合。user.roles と所属 Group の roles の和集合。admin RBAC ゲートと /account 自己コンテキストで参照する。 | effective roles, 有効ロール |
| Agent | tenant-scoped な非人間 (non-human) identity principal。自身の資格情報は持たず、AgentCredentialBinding で既存 OAuth2Client に束縛してトークンを得る。owner (所有者 User の id) は必須。 | agent, エージェント, AI agent, non-human identity |
| Killed | Agent の緊急停止 (kill-switch) による一方向終端状態。Killed からは復帰できず、新規トークンを一切発行しない (fail-closed)。 | killed |
| DisableAgent | Agent を Active から Disabled に遷移させる可逆な運用停止。`/api/admin/agents/{agent_id}/disable` から発火。 | disable agent |
| EnableAgent | Disabled の Agent を Active に戻す。`/api/admin/agents/{agent_id}/enable` から発火。 | enable agent |
| KillAgent | Agent を Killed (一方向終端) に遷移させる緊急停止。`/api/admin/agents/{agent_id}/kill` から発火し、以後復帰不能。 | kill agent |
| Autonomous | Agent が人間の都度承認なしに自律実行する区分。 | autonomous |
| Supervised | Agent が人間の監督下で実行する区分。 | supervised |

## State machines

### UserLifecycle

User aggregate のライフサイクル。Active が通常稼働、
Disable で復活可能な無効化。SoftDelete で削除予約 (PendingDeletion) に入り、
猶予期間内は Restore で Active に戻せる。Purge で tombstone 化
(anonymize cascade) となる。Deleted は終端で復元できない。PendingDeletion から
Deleted への遷移 guard は、猶予期間 (30日、業界標準の 7〜30 日に合わせた既定値) を
経過したか、admin の明示的 purge=true 指定のいずれかを要求する。経過前に
purge=false で PendingDeletion の user へ呼び出した場合は DeleteAdminUser.requires
が InvalidRequestError で拒否し、本遷移は発生しない。

Initial: `Active`  
Terminal: `Deleted`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | UserDisabled | "" | Disabled |  |
| Disabled | UserEnabled | "" | Active |  |
| Active | UserSoftDeleted | "" | PendingDeletion |  |
| Disabled | UserSoftDeleted | "" | PendingDeletion |  |
| PendingDeletion | UserRestored | "" | Active |  |
| PendingDeletion | UserDeleted | input.purge == true \|\| duration_since(status_changed_at) >= duration('2592000s') | Deleted | UserDeleted |
| Active | UserDeleted | "" | Deleted |  |
| Disabled | UserDeleted | "" | Deleted |  |

### DynamicMembershipEvaluationLifecycle

全件再評価は queued から running を経て succeeded または failed へ終端する。

Initial: `queued`  
Terminal: `succeeded`, `failed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DynamicMembershipEvaluationStarted | "" | running |  |
| running | DynamicMembershipEvaluated | "" | succeeded |  |
| running | DynamicMembershipEvaluationFailed | "" | failed |  |

### AgentLifecycle

Agent aggregate のライフサイクル。Active が通常稼働、Disable で
復活可能な運用停止、Kill で一方向終端 (緊急停止) となる。Killed は終端で
復元できず、Active 以外は新規トークンを発行しない (fail-closed)。

Initial: `Active`  
Terminal: `Killed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | AgentDisabled | "" | Disabled |  |
| Disabled | AgentEnabled | "" | Active |  |
| Active | AgentKilled | "" | Killed |  |
| Disabled | AgentKilled | "" | Killed |  |

### DataExportLifecycle

リソースエクスポートのライフサイクル。queued で受理され、worker が
running で CSV を生成する。成功すると succeeded (ファイルはダウンロード可能)、
失敗すると failed (不完全ファイルはダウンロード不可) に終端する。終端前は
canceled で取消せる。succeeded は保持期限を経過すると expired へ遷移し、
ファイル本体は purge されて metadata と監査だけが残る。succeeded から expired へ
の遷移 guard は保持期限 (30日、Jobs の既定 record retention と一致) の経過を
要求する。

Initial: `queued`  
Terminal: `failed`, `canceled`, `expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DataExportStarted | "" | running |  |
| running | DataExportSucceeded | "" | succeeded |  |
| running | DataExportFailed | "" | failed |  |
| queued | DataExportCanceled | "" | canceled |  |
| running | DataExportCanceled | "" | canceled |  |
| succeeded | DataExportExpired | duration_since(completed_at) >= duration('2592000s') | expired |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
