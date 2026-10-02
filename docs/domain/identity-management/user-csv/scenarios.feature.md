# Feature: ユーザー CSV のシナリオ

## 入力

### Rule: REQ-IDMANAGEMENT-004 管理者は CSV を検証して有効な行だけをインポートできる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-004-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 有効な行は作成または更新され、無効な行は `rejected` として残る。各行のプロフィール、ロール、必須操作、カスタム属性は不可分に保存される

#### Example: EX-IDMANAGEMENT-004-02 CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- But CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える
- Then インポートの投入は拒否される
- And エラー "csv_too_large" / "too_many_rows" / "field_too_large"

#### Example: EX-IDMANAGEMENT-004-03 CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- But CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる
- Then インポートの投入は拒否される
- And エラー "invalid_header"

#### Example: EX-IDMANAGEMENT-004-04 行の `id` と `preferred_username` が別の `User` を示す、識別子がない、同じ対象または同じ最終ユーザー名を複数行が示す

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then 行の `id` と `preferred_username` が別の `User` を示す、識別子がない、同じ対象または同じ最終ユーザー名を複数行が示す
- Then 対象行は `rejected` となり、安定したエラーコードを返す

#### Example: EX-IDMANAGEMENT-004-05 プレビュージョブが存在しない、`queued` または `failed` である、別テナントに属する、保存済みのペイロードとダイジェストが一致しない

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- But プレビュージョブが存在しない、`queued` または `failed` である、別テナントに属する、保存済みのペイロードとダイジェストが一致しない
- Then 適用は `User` を変更せず `InvalidRequestError` または `AccessDeniedError` で拒否される

#### Example: EX-IDMANAGEMENT-004-06 プレビュー後に対象 `User` の状態が別の操作で変更されている

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then プレビュー後に対象 `User` の状態が別の操作で変更されている
- Then 適用は古いプレビュー計画を実行せず、現在状態から `updated`、`unchanged`、`rejected` を再判定する

#### Example: EX-IDMANAGEMENT-004-07 対象 `User` が外部の取り込み元に管理されている

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 対象 `User` が外部の取り込み元に管理されている
- Then 対象行は安定したエラーコード `source_managed` で `rejected` となり、`User` は変更されない

#### Example: EX-IDMANAGEMENT-004-08 1 行の検証、保存、監査処理が途中で失敗する

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- When 管理者が機械可読なヘッダー [id, email, roles, custom:department] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 1 行の検証、保存、監査処理が途中で失敗する
- Then その行のプロフィール、ロール、必須操作、カスタム属性は一部も保存されず、他の有効な行は適用を続ける

## 結果

### Rule: REQ-IDMANAGEMENT-006 管理者はユーザー一覧を CSV に安全にエクスポートできる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-006-01 通常経路

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

#### Example: EX-IDMANAGEMENT-006-02 選択列に `User` の許可一覧にないキー（例: `password_hash`）が含まれる

- Given ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- And 一覧には自テナントのユーザーが存在する
- When 管理者が列 [`preferred_username`, `email`] と `status` フィルターを指定して `/users/exports` へエクスポートを開始する
- But 選択列に `User` の許可一覧にないキー（例: `password_hash`）が含まれる
- Then エクスポート開始は `InvalidRequestError` で拒否される
- And エラー `invalid_columns`

#### Example: EX-IDMANAGEMENT-006-03 生成が失敗する

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

#### Example: EX-IDMANAGEMENT-006-04 セル値が \"=\", \"+\", \"-\", \"@\", タブ, CR, LF のいずれかで始まる

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

#### Example: EX-IDMANAGEMENT-006-05 保持期限を経過している

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

#### Example: EX-IDMANAGEMENT-006-06 `User` エクスポートの ID を `/groups/exports` または別テナントで指定する

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

### Rule: REQ-IDMANAGEMENT-007 管理者はエクスポートしたユーザー CSV を安全に再適用できる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-007-01 通常経路

- Given 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`User` は変更されない
- When 管理者が 1 行の `email` と `custom:department` だけを編集して再びプレビューする
- Then 変更行だけが updated、残りは unchanged と計画される
- When 管理者が成功済みプレビュージョブの ID を指定して適用する
- Then 指定した書き込み可能な列だけが更新され、指定しなかった列は維持される

#### Example: EX-IDMANAGEMENT-007-02 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む

- Given 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
- But 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む
- Then 可逆な数式安全変換と RFC 4180 の引用により `decode(encode(value))` は元の値と一致する

#### Example: EX-IDMANAGEMENT-007-03 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える

- Given 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
- But 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える
- Then `User` エクスポートは `csv_transfer_limit_exceeded` で失敗し、再インポートできない成功済み成果物を作らない
- And 管理者はフィルターまたは列を絞って複数の成果物に分割できる

#### Example: EX-IDMANAGEMENT-007-04 `status`、`mfa_enrolled`、`created_at`、`updated_at`、`id` の値だけを編集する

- Given 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`User` は変更されない
- When 管理者が 1 行の `email` と `custom:department` だけを編集して再びプレビューする
- But `status`、`mfa_enrolled`、`created_at`、`updated_at`、`id` の値だけを編集する
- Then 読み取り専用列は受理したうえで無視し、書き込み可能な列に差分がなければ `unchanged` とする

#### Example: EX-IDMANAGEMENT-007-05 カスタム属性の型、真偽値、数値、日付、必須のカスタム属性、`required_actions` のいずれかが不正である

- Given 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる
- When 管理者がインポート可能な組み込み列、`required_actions`、`custom:department` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`User` は変更されない
- When 管理者が 1 行の `email` と `custom:department` だけを編集して再びプレビューする
- But カスタム属性の型、真偽値、数値、日付、必須のカスタム属性、`required_actions` のいずれかが不正である
- Then 対象行は安定したエラーコードで `rejected` となり、値はジョブの表示にも監査イベントにも含めない
