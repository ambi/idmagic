# Feature: IdManagement のシナリオ

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

## Rule: REQ-IDMANAGEMENT-032 `system_admin` は制御面テナントの User と Group にだけ割り当てられる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-032-01 通常経路

- Given 制御面テナント `default` にロール=["admin"] のユーザー "operator" とロールが空のユーザー "root" が存在する
- When 管理者 "operator" がロール=["system_admin"] のグループ "system-operators" を作成する
- Then "GroupCreated" が発行され、グループのロールに "system_admin" が含まれる
- When 管理者 "operator" がユーザー "root" をグループ "system-operators" に所属させる
- Then "GroupMemberAdded" が発行される
- When 管理者 "operator" がユーザー "root" の所属グループを取得する
- Then 実効ロールに "system_admin" が含まれ、`group_roles` は "system_admin" を含み、`direct_roles` は空である

### Example: EX-IDMANAGEMENT-032-02 制御面テナントの User へ直接割り当てる

- Given 制御面テナント `default` にロール=["admin"] のユーザー "operator" とロールが空のユーザー "root" が存在する
- When 管理者 "operator" がユーザー "root" のロールを ["system_admin"] へ更新する
- Then 更新は成功し、ユーザー "root" の `roles` は "system_admin" を含む

### Example: EX-IDMANAGEMENT-032-03 制御面テナント以外のテナントの User へ直接割り当てる

- Given 制御面テナントではないテナント "acme" にロール=["admin"] のユーザー "operator" とロールが空のユーザー "alice" が存在する
- When 管理者 "operator" がユーザー "alice" のロールを ["system_admin"] へ、表示名を同じ要求で更新する
- Then 更新は `InvalidRequestError` で拒否され、エラー "invalid_role" を返す
- And ユーザー "alice" の `roles`、表示名、`updated_at` はいずれも変更されない

### Example: EX-IDMANAGEMENT-032-04 制御面テナント以外のテナントの Group へ割り当てる

- Given 制御面テナントではないテナント "acme" にロール=["admin"] のユーザー "operator" が存在する
- When 管理者 "operator" がロール=["system_admin"] のグループ "escalation" を作成する
- Then 作成は `InvalidRequestError` で拒否され、エラー "invalid_role" を返す
- And グループ "escalation" は作成されず、"GroupCreated" も発行されない

### Example: EX-IDMANAGEMENT-032-05 制御面テナントの Agent へ割り当てる

- Given 制御面テナント `default` にロール=["admin"] のユーザー "operator" と在籍中の Agent "batch" が存在する
- When 管理者 "operator" が Agent "batch" のロールを ["system_admin"] へ更新する
- Then 更新は `InvalidRequestError` で拒否され、エラー "invalid_role" を返す
- And Agent "batch" の `roles` と `updated_at` は変更されず、"AgentUpdated" も発行されない

### Example: EX-IDMANAGEMENT-032-06 制御面テナント以外のテナントの User CSV の行が `system_admin` を含む

- Given 制御面テナントではないテナント "acme" にロール=["support"] のユーザー "alice" が存在する
- When 管理者が `roles` 列に "system_admin" を持つ行を含む User CSV を事前検証へ投入する
- Then 対象行は `roles` 列を指す安定したエラーコード `invalid_roles` で `rejected` となり、ユーザー "alice" は変更されない

### Example: EX-IDMANAGEMENT-032-07 制御面テナント以外のテナントの Group CSV の行が `system_admin` を含む

- Given 制御面テナントではないテナント "acme" にロール=["catalog:read"] のグループ "engineering" が存在する
- When 管理者が `roles` 列に "system_admin" を持つ行を含む Group CSV を事前検証へ投入する
- Then 対象行は `roles` 列を指す安定したエラーコード `invalid_roles` で `rejected` となり、グループ "engineering" は変更されない

### Example: EX-IDMANAGEMENT-032-08 既に `system_admin` を持つ対象へ同じ値を再送する

- Given 制御面テナントではないテナント "acme" にロール=["system_admin"] のユーザー "legacy" が既に存在する
- When 管理者が `roles` 列に "system_admin" を持つ "legacy" の行を含む User CSV を事前検証へ投入する
- Then 対象行は `unchanged` となり、ユーザー "legacy" の `roles` は "system_admin" のまま残る

### Example: EX-IDMANAGEMENT-032-09 制御面テナント以外のテナントで User を作成する

- Given 制御面テナントではないテナント "acme" にロール=["admin"] のユーザー "operator" が存在する
- When 管理者 "operator" がロール=["system_admin"] のユーザー "mallory" を作成する
- Then 作成は `InvalidRequestError` で拒否され、エラー "invalid_role" を返す
- And ユーザー "mallory" は作成されず、"UserCreated" も発行されない

### Example: EX-IDMANAGEMENT-032-10 制御面テナント以外のテナントの Group を更新する

- Given 制御面テナントではないテナント "acme" にロール=["catalog:read"] のグループ "engineering" が存在する
- When 管理者がグループ "engineering" のロールを ["system_admin"] へ、説明を同じ要求で更新する
- Then 更新は `InvalidRequestError` で拒否され、エラー "invalid_role" を返す
- And グループ "engineering" の `roles`、説明、`updated_at` はいずれも変更されない

## Rule: REQ-IDMANAGEMENT-033 ロールは前後の空白を除き、重複を除いて昇順に保存する

- User、Group、Agent のロールは、管理 API と CSV のどちらから書いても同じ正規化を通す。
- 各ロールの前後の空白を除く。
- 空白を除いて空になるロールを一つでも含む書き込みは、`invalid_role` で拒否し、対象を変更しない。
- 同じロールを二度以上含む書き込みは、一つにまとめて保存する。
- 保存するロールの並びは、文字列の昇順とする。
- ロールの照合は大文字と小文字を区別する。`Admin` と `admin` は別のロールである。
- **担保手段**：`usecases.NormalizeRoles`

### Example: EX-IDMANAGEMENT-033-01 空白と重複を含むロール

- When 管理者がロール [" support ", "audit", "support"] を指定する
- Then 保存するロールは ["audit", "support"] である

### Example: EX-IDMANAGEMENT-033-02 空のロールを含む書き込み

- When 管理者がロール ["audit", "  "] を指定する
- Then 書き込みは `invalid_role` で拒否される

### Example: EX-IDMANAGEMENT-033-03 大文字と小文字だけが異なるロール

- When 管理者がロール ["Admin", "admin"] を指定する
- Then 保存するロールは ["Admin", "admin"] の二つである

## Rule: REQ-IDMANAGEMENT-034 CSV の見出しは機械キーと完全に一致する名前だけを受け付ける

- 見出しの先頭の UTF-8 の BOM は取り除いてから照合する。
- 見出しの各列名は、大文字と小文字、前後の空白を含めて、種別が受け付ける機械キーと完全に一致しなければならない。一致しない列名を含むファイルは `invalid_header` で拒否する。
- `password`、`password_hash`、`mfa_secret`、`token`、`recovery_code` を見出しに含むファイルは、種別によらず `invalid_header` で拒否する。
- 見出しだけの行もない空のファイルは `invalid_csv` で拒否する。
- 見出しによる拒否はファイル全体に及び、どの行も計画しない。
- **担保手段**：`domain.NewCSVReader`

### Example: EX-IDMANAGEMENT-034-01 BOM で始まる見出し

- Given ファイルは UTF-8 の BOM の直後に見出し `preferred_username` を持つ
- When 管理者がそのファイルを事前検証へ投入する
- Then 見出しは `preferred_username` として受け付けられる

### Example: EX-IDMANAGEMENT-034-02 大文字を含む列名

- When 管理者が見出し `Preferred_Username` を持つファイルを事前検証へ投入する
- Then ファイルは `invalid_header` で拒否される

### Example: EX-IDMANAGEMENT-034-03 秘密情報の列名

- When 管理者が見出しに `mfa_secret` を含むファイルを事前検証へ投入する
- Then ファイルは `invalid_header` で拒否される

## Rule: REQ-IDMANAGEMENT-035 列の数が見出しと異なる行は、その行だけを拒否する

- 列の数が見出しの列の数と異なる行は、`invalid_column_count` で `rejected` とし、続く行の解析と計画を続ける。
- 転送ポリシーの上限を超えたとき、または RFC 4180 として読めないときは、その位置でファイル全体の解析を止める。
- **担保手段**：`CSVReader.Next`

### Example: EX-IDMANAGEMENT-035-01 列の足りない行を挟むファイル

- Given ファイルの見出しは二列であり、2 行目のデータだけが一列である
- When 管理者がそのファイルを事前検証へ投入する
- Then 2 行目は `invalid_column_count` で `rejected` となり、1 行目と 3 行目は計画される

## Rule: REQ-IDMANAGEMENT-036 先頭のアポストロフィーは、数式の保護として付けたものだけを取り除く

- インポートは、セルがアポストロフィーで始まり、その次の文字がアポストロフィーまたは `=`、`+`、`-`、`@`、タブ、CR、LF のときだけ、先頭の一文字を取り除く。
- それ以外のアポストロフィーで始まるセルは、そのまま読む。
- **担保手段**：`domain.DecodeCSVCell`

### Example: EX-IDMANAGEMENT-036-01 保護として付けたアポストロフィー

- When インポートが `'=SUM(A1)` のセルを読む
- Then 値は `=SUM(A1)` である

### Example: EX-IDMANAGEMENT-036-02 値の一部であるアポストロフィー

- When インポートが `'abc` のセルを読む
- Then 値は `'abc` のまま変わらない

## Rule: REQ-IDMANAGEMENT-037 CSV の転送ポリシーのデフォルト値と上限の境界

| 上限 | デフォルト値 |
| --- | --- |
| データ行の数 | 100,000 行 |
| 成果物の大きさ | 64 MiB |
| 一つの項目の大きさ | 64 KiB |

- データ行の数は、見出しの行を除いて数える。
- どの上限も、上限と等しい値は受け付け、上限を超えた値を拒否する。
- 上限は一つの成果物に対するもので、テナントの User や Group の件数を制限しない。
- **担保手段**：`domain.DefaultCSVTransferPolicy`、`domain.NewCSVReader`、`domain.NewCSVWriter`

### Example: EX-IDMANAGEMENT-037-01 上限と等しい行の数

- Given 転送ポリシーのデータ行の上限は 2 行である
- When 見出しと 2 行のデータを持つファイルを読む
- Then 2 行とも読み取られる

### Example: EX-IDMANAGEMENT-037-02 上限を超える行の数

- Given 転送ポリシーのデータ行の上限は 2 行である
- When 見出しと 3 行のデータを持つファイルを読む
- Then 3 行目で `too_many_rows` となり解析を止める

## Rule: REQ-IDMANAGEMENT-038 属性値のセルは型ごとの正規の字句形だけを受け付ける

| 型 | 受け付ける字句形 |
| --- | --- |
| `string` | 任意の文字列 |
| `date` | `YYYY-MM-DD` の暦日 |
| `number` | 最短の十進表記。`1.0` や `01` は受け付けない |
| `boolean` | `true` または `false` |
| `string_array` | 空白を含まない JSON の文字列配列 |

- エクスポートは、同じ字句形で値を書き出す。
- 空のセルは、任意の属性ではその属性を消す意味とし、必須の属性では拒否する。
- User の CSV と Group の CSV は同じ字句形を使う。
- **担保手段**：`domain.ParseAttributeCell`、`domain.FormatAttributeCell`

### Example: EX-IDMANAGEMENT-038-01 正規の字句形の値

- Given 属性 `level` の型は `number` である
- When インポートが `1.5` のセルを読む
- Then 値は数値の 1.5 である

### Example: EX-IDMANAGEMENT-038-02 正規でない字句形の値

- Given 属性 `level` の型は `number` である
- When インポートが `1.0` のセルを読む
- Then セルは拒否される

### Example: EX-IDMANAGEMENT-038-03 任意の属性の空のセル

- Given 属性 `nickname` は任意である
- When インポートが空のセルを読む
- Then その属性を消す

## Rule: REQ-IDMANAGEMENT-039 エクスポートの開始は、許可した列と絞り込みだけを受け付ける

- 列は一つ以上を指定し、同じ列を二度含めない。違反する開始は `invalid_columns` で拒否する。
- 絞り込みのキーは、User のエクスポートでは `status` だけ、Group のエクスポートでは受け付けない。それ以外のキーを含む開始は `invalid_filter` で拒否する。
- `status` の値は前後の空白と大文字と小文字を区別せず、User の状態のどれでもない値を `invalid_filter` で拒否する。
- メンバーシップのエクスポートの対象 Group は、パスの `group_id` だけで決める。本文の絞り込みは対象を変えない。
- 拒否した開始はジョブを作らず、`DataExportRequested` を発行しない。
- 実行中のジョブの上限を超える開始は、429 と `active_job_quota_exceeded` で拒否する。
- **担保手段**：`usecases.StartDataExport`、`handlers_http.HandleStartUserExport`

### Example: EX-IDMANAGEMENT-039-01 重複した列

- When 管理者が列 [`email`, `email`] で User のエクスポートを開始する
- Then 開始は `invalid_columns` で拒否され、エクスポートは作られない

### Example: EX-IDMANAGEMENT-039-02 許可していない絞り込み

- When 管理者が絞り込み `{"email": "a@example.test"}` で User のエクスポートを開始する
- Then 開始は `invalid_filter` で拒否され、エクスポートは作られない

### Example: EX-IDMANAGEMENT-039-03 大文字の状態の絞り込み

- When 管理者が絞り込み `{"status": " Active "}` で User のエクスポートを開始する
- Then 開始は受け付けられる

## Rule: REQ-IDMANAGEMENT-040 エクスポートの一覧は新しい順に、テナントの直近 200 件から返す

- 一覧は、テナントのエクスポートを作成の新しい順に並べる。
- 一覧は、テナントのすべての種類のエクスポートのうち新しい 200 件を読み、そこから要求した種類と Group に属するものだけを返す。
- **担保手段**：`usecases.ListDataExports`
- **要判断**：別の種類のエクスポートが新しい 200 件を占めると、要求した種類の古いエクスポートは、保持期限内でも一覧に現れない。種類ごとに 200 件を返すか、ページングにするかを決める。

### Example: EX-IDMANAGEMENT-040-01 新しい順

- Given テナントに User のエクスポートが二つあり、二つ目のほうが新しい
- When 管理者が User のエクスポートの一覧を取得する
- Then 二つ目、一つ目の順に返る

### Example: EX-IDMANAGEMENT-040-02 別の種類が新しい 200 件を占める

- Given テナントに古い User のエクスポートが一つあり、その後に Group のエクスポートが 200 件ある
- When 管理者が User のエクスポートの一覧を取得する
- Then 一覧は空である

## Rule: REQ-IDMANAGEMENT-041 終了したエクスポートは取り消せない

- `succeeded`、`failed`、`canceled`、`expired` のエクスポートの取り消しは、409 と `data_export_not_cancelable` で拒否する。
- 拒否した取り消しは状態を変えず、`DataExportCanceled` を発行しない。
- **担保手段**：`usecases.CancelDataExport`

### Example: EX-IDMANAGEMENT-041-01 取り消し済みのエクスポートをもう一度取り消す

- Given 管理者が User のエクスポートを取り消している
- When 管理者が同じエクスポートをもう一度取り消す
- Then 409 と `data_export_not_cancelable` を返し、`DataExportCanceled` は再発行されない
